<template>
  <div class="page-padding h-full flex flex-col overflow-hidden">
    <!-- Page Header -->
    <div class="flex items-center justify-between section-gap shrink-0">
      <div>
        <h2 class="text-[17px] font-semibold text-[var(--text-primary)]">备份管理</h2>
        <p class="text-[13px] text-[var(--text-tertiary)] mt-0.5">管理数据库备份与定时计划</p>
      </div>
      <div class="flex items-center gap-2">
        <button class="btn-secondary h-[32px] text-[13px] max-sm:hidden" @click="handleImport">
          <Upload class="h-3.5 w-3.5 mr-1.5" />导入
        </button>
        <button class="btn-secondary h-[32px] text-[13px] max-sm:hidden" @click="openScheduleDialog()">
          <Clock class="h-3.5 w-3.5 mr-1.5" />新建定时
        </button>
        <button class="btn-primary h-[32px] text-[13px]" @click="handleCreate">
          <Plus class="h-3.5 w-3.5 mr-1.5" />创建备份
        </button>
      </div>
    </div>

    <div class="content-card flex flex-col flex-1 min-h-0">
      <!-- Tab Switcher -->
      <div class="flex items-center justify-between px-5 pt-4 pb-0 shrink-0">
        <div class="inline-flex items-center gap-1 p-1 rounded-xl bg-[var(--muted)]">
          <button
            :class="activeMainTab === 'records' ? 'tab-active px-4 py-1.5 text-[13px]' : 'tab-inactive px-4 py-1.5 text-[13px]'"
            @click="activeMainTab = 'records'"
          >备份记录
          </button>
          <button
            :class="activeMainTab === 'scheduled' ? 'tab-active px-4 py-1.5 text-[13px]' : 'tab-inactive px-4 py-1.5 text-[13px]'"
            @click="activeMainTab = 'scheduled'"
          >定时备份
          </button>
        </div>
      </div>

      <!-- Filter Bar (records tab) -->
      <div v-if="activeMainTab === 'records'" class="flex items-center gap-2 px-5 pt-3 pb-2 shrink-0 flex-wrap">
        <div class="flex h-[32px] items-center rounded-lg border border-[var(--border)] bg-[var(--surface)] px-2.5 gap-1.5">
          <Search class="h-3.5 w-3.5 text-[var(--text-tertiary)]" />
          <Input v-model="searchQuery" placeholder="搜索备份..." class="border-0 shadow-none h-[28px] text-[13px] w-[140px] bg-transparent" @input="onSearchInput" />
        </div>
        <Select v-model="databaseFilter" @update:model-value="onFilterChange">
          <SelectTrigger class="h-[32px] w-[150px] text-[13px] border-[var(--border)]">
            <Database class="h-3.5 w-3.5 mr-1" />
            <SelectValue placeholder="全部数据库" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部数据库</SelectItem>
            <SelectItem v-for="db in allDbOptions" :key="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name" :value="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name">
              <div class="flex items-center gap-2">
                <Badge :class="db.type === 'redis' ? 'bg-amber-500/5 text-amber-600 border-amber-500/20' : 'bg-blue-500/5 text-blue-600 border-blue-500/20'" class="text-[10px] px-1.5 py-0 rounded-full">
                  {{ db.type === 'redis' ? 'R' : 'M' }}
                </Badge>
                {{ db.name }}
                <span class="text-[var(--text-tertiary)] text-xs ml-auto">{{ db.host || '本地' }}</span>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
        <div class="flex items-center gap-1">
          <button
            v-for="opt in levelOptions" :key="opt.value"
            :class="backupLevelFilter === opt.value ? 'pill pill-active' : 'pill pill-default'"
            @click="backupLevelFilter = opt.value; onFilterChange()"
          >
            <Database v-if="opt.value === 'mysql'" class="h-3 w-3 shrink-0" />
            <HardDrive v-else-if="opt.value === 'redis'" class="h-3 w-3 shrink-0" />
            <Settings v-else-if="opt.value === 'system'" class="h-3 w-3 shrink-0" />
            {{ opt.label }}
          </button>
        </div>
        <div class="flex items-center gap-1.5 ml-auto">
          <button class="btn-ghost h-[30px] text-[12px]" @click="loadBackups">
            <RefreshCw class="h-3.5 w-3.5" />
          </button>
          <button v-if="hasActiveFilters" class="btn-ghost h-[30px] text-[12px] text-[var(--text-tertiary)]" @click="clearAllFilters">
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <!-- Content Area -->
      <div class="flex-1 min-h-0 px-5 pb-4">
        <!-- Records Tab -->
        <div v-if="activeMainTab === 'records'" class="h-full flex flex-col">
          <div class="flex-1 min-h-0 overflow-y-auto">
            <!-- Loading -->
            <div v-if="loading" class="flex items-center justify-center py-16">
              <Loader2 class="h-5 w-5 text-[var(--accent)] animate-spin mr-2" />
              <span class="text-[13px] text-[var(--text-secondary)]">加载中...</span>
            </div>
            <!-- Empty -->
            <div v-else-if="pageItems.length === 0" class="empty-state">
              <div class="empty-state-icon"><Inbox class="h-8 w-8" /></div>
              <p class="empty-state-text">暂无备份记录</p>
            </div>
            <!-- Table -->
            <Table v-else :class="{ 'opacity-40 pointer-events-none': loading }">
              <TableHeader>
                <TableRow class="hover:bg-transparent border-b border-[var(--border-subtle)]">
                  <TableHead class="w-10 text-[12px] font-medium text-[var(--text-tertiary)] h-9">
                    <input type="checkbox" class="h-[14px] w-[14px] rounded border-[var(--border)] accent-[var(--accent)] cursor-pointer" :checked="isAllSelected" :indeterminate="isPartialSelected" @change="toggleSelectAll" />
                  </TableHead>
                  <TableHead class="min-w-[220px] text-[12px] font-medium text-[var(--text-tertiary)] h-9 cursor-pointer select-none hover:text-[var(--text-primary)] transition-colors" @click="toggleSort('name')">
                    <span class="flex items-center gap-1">
                      备份名称
                      <ArrowUpDown v-if="sortField !== 'name'" class="h-3 w-3 opacity-30" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-[var(--accent)]" />
                      <ArrowDown v-else class="h-3 w-3 text-[var(--accent)]" />
                    </span>
                  </TableHead>
                  <TableHead class="w-[72px] text-[12px] font-medium text-[var(--text-tertiary)] h-9">状态</TableHead>
                  <TableHead class="min-w-[100px] text-[12px] font-medium text-[var(--text-tertiary)] h-9">目标数据库</TableHead>
                  <TableHead class="w-[80px] text-[12px] font-medium text-[var(--text-tertiary)] h-9 cursor-pointer select-none hover:text-[var(--text-primary)] transition-colors" @click="toggleSort('fileSize')">
                    <span class="flex items-center gap-1">
                      大小
                      <ArrowUpDown v-if="sortField !== 'fileSize'" class="h-3 w-3 opacity-30" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-[var(--accent)]" />
                      <ArrowDown v-else class="h-3 w-3 text-[var(--accent)]" />
                    </span>
                  </TableHead>
                  <TableHead class="min-w-[145px] text-[12px] font-medium text-[var(--text-tertiary)] h-9 cursor-pointer select-none hover:text-[var(--text-primary)] transition-colors" @click="toggleSort('createdAt')">
                    <span class="flex items-center gap-1">
                      创建时间
                      <ArrowUpDown v-if="sortField !== 'createdAt'" class="h-3 w-3 opacity-30" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-[var(--accent)]" />
                      <ArrowDown v-else class="h-3 w-3 text-[var(--accent)]" />
                    </span>
                  </TableHead>
                  <TableHead class="text-center min-w-[200px] text-[12px] font-medium text-[var(--text-tertiary)] h-9">
                    <span class="inline-flex items-center gap-1">
                      操作
                      <button
                        v-if="sortField"
                        class="ml-1 inline-flex items-center gap-0.5 text-[11px] text-[var(--text-tertiary)] hover:text-[var(--accent)] transition-colors"
                        @click.stop="resetSort"
                      >
                        <RotateCcw class="h-2.5 w-2.5" />重置
                      </button>
                    </span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="row in pageItems" :key="row.id" :class="{ 'bg-[var(--accent-soft)]': selectedIds.has(row.id) }" class="hover:bg-[var(--surface)] transition-colors duration-150 border-b border-[var(--border-subtle)]">
                  <TableCell>
                    <input type="checkbox" class="h-[14px] w-[14px] rounded border-[var(--border)] accent-[var(--accent)] cursor-pointer" :checked="selectedIds.has(row.id)" @change="toggleSelect(row)" />
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center gap-2">
                      <div class="h-7 w-7 rounded-lg flex items-center justify-center shrink-0" :class="row.backupLevel === 'system' ? 'bg-[var(--accent-soft)]' : row.backupLevel === 'redis' ? 'bg-amber-500/10' : 'bg-blue-500/10'">
                        <Settings v-if="row.backupLevel === 'system'" class="h-3.5 w-3.5 text-[var(--text-tertiary)]" />
                        <HardDrive v-else-if="row.backupLevel === 'redis'" class="h-3.5 w-3.5 text-amber-500" />
                        <Database v-else class="h-3.5 w-3.5 text-blue-500" />
                      </div>
                      <div class="flex flex-col gap-0.5">
                        <div class="flex items-center gap-1.5">
                          <span class="text-[13px] text-[var(--text-primary)] truncate max-w-[180px] inline-block" :title="row.name">{{ row.name }}</span>
                          <Badge v-if="row.backupType === 'import'" variant="outline" class="text-[10px] py-0 bg-emerald-500/10 text-emerald-500 border-emerald-500/20 shrink-0">导入</Badge>
                          <Badge v-else-if="row.backupType === 'scheduled'" variant="outline" class="text-[10px] py-0 bg-[var(--accent-soft)] text-[var(--accent)] border-[color-mix(in_srgb,var(--accent)_20%,transparent)] shrink-0 font-medium">定时</Badge>
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span v-if="row.status === 'success'" class="badge-status badge-status-success whitespace-nowrap">
                      <span class="h-1.5 w-1.5 rounded-full bg-[var(--success)]"></span>成功
                    </span>
                    <span v-else class="badge-status badge-status-error whitespace-nowrap">
                      <span class="h-1.5 w-1.5 rounded-full bg-[var(--danger)]"></span>失败
                    </span>
                  </TableCell>
                  <TableCell><span class="text-[13px] text-[var(--text-primary)]" v-if="row.database">{{ row.database }}</span><span class="text-[13px] text-[var(--text-tertiary)]" v-else>-</span></TableCell>
                  <TableCell><span class="text-[13px] font-mono-data text-[var(--text-primary)]">{{ formatSize(row.fileSize) }}</span></TableCell>
                  <TableCell><span class="text-[13px] text-[var(--text-secondary)]">{{ formatLogTime(row.createdAt) }}</span></TableCell>
                  <TableCell>
                    <div class="flex items-center justify-center gap-1">
                      <button class="btn-ghost h-[28px] text-[11px] gap-1 text-[var(--text-secondary)] hover:text-[var(--text-primary)]" @click="handleRestore(row)">
                        <RotateCcw class="h-3.5 w-3.5 shrink-0" />恢复
                      </button>
                      <button class="btn-ghost h-[28px] text-[11px] gap-1 text-[var(--text-secondary)] hover:text-[var(--text-primary)]" @click="handleDownload(row)">
                        <Download class="h-3.5 w-3.5 shrink-0" />下载
                      </button>
                      <button class="btn-ghost-danger h-[28px] text-[11px] gap-1" @click="confirmDeleteTarget = row; showDeleteDialog = true">
                        <Trash2 class="h-3.5 w-3.5 shrink-0" />
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <!-- Pagination -->
          <div v-if="pageItems.length > 0" class="flex items-center justify-between pt-3 shrink-0">
            <span class="text-[12px] text-[var(--text-tertiary)]">共 {{ totalItems }} 条，第 {{ currentPage }}/{{ totalPages }} 页</span>
            <div class="flex items-center gap-1">
              <button class="btn-ghost h-7 w-7 p-0 text-[12px]" :disabled="currentPage <= 1" @click="goToPage(currentPage - 1)">
                <ChevronLeft class="h-3.5 w-3.5" />
              </button>
              <template v-for="p in visiblePageNumbers" :key="p">
                <span v-if="p === '...'" class="px-1 text-[12px] text-[var(--text-tertiary)]">...</span>
                <button
                  v-else
                  :class="p === currentPage ? 'pagination-active' : 'btn-ghost'"
                  class="h-7 min-w-[28px] px-1 text-[12px]"
                  @click="goToPage(p)"
                >{{ p }}</button>
              </template>
              <button class="btn-ghost h-7 w-7 p-0 text-[12px]" :disabled="currentPage >= totalPages" @click="goToPage(currentPage + 1)">
                <ChevronRight class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>

        <!-- Scheduled Tab -->
        <div v-if="activeMainTab === 'scheduled'" class="h-full flex flex-col">
          <div class="flex-1 min-h-0 overflow-y-auto pt-3">
            <!-- Empty -->
            <div v-if="scheduleList.length === 0" class="empty-state">
              <div class="empty-state-icon"><Inbox class="h-8 w-8" /></div>
              <p class="empty-state-text">暂无定时备份计划</p>
            </div>
            <!-- Card List -->
            <div v-else class="grid gap-3">
              <div v-for="row in scheduleList" :key="row.id" class="content-card-interactive hover-lift p-4 flex items-center gap-4">
                <!-- Icon -->
                <div class="h-10 w-10 rounded-xl flex items-center justify-center shrink-0" :class="row.backupLevel === 'system' ? 'bg-[var(--accent-soft)]' : row.backupLevel === 'redis' ? 'bg-amber-500/10' : 'bg-blue-500/10'">
                  <Settings v-if="row.backupLevel === 'system'" class="h-5 w-5 text-[var(--text-tertiary)]" />
                  <HardDrive v-else-if="row.backupLevel === 'redis'" class="h-5 w-5 text-amber-500" />
                  <Database v-else class="h-5 w-5 text-blue-500" />
                </div>
                <!-- Info -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-[13px] font-medium text-[var(--text-primary)] truncate">{{ row.name }}</span>
                    <Badge v-if="row.backupLevel === 'system'" variant="outline" class="text-[10px] py-0 bg-[var(--accent-soft)] text-[var(--text-secondary)] border-[var(--border)]">系统</Badge>
                    <Badge v-else-if="row.backupLevel === 'redis'" variant="outline" class="text-[10px] py-0 bg-amber-500/10 text-amber-600 border-amber-500/20">Redis</Badge>
                    <Badge v-else variant="outline" class="text-[10px] py-0 bg-blue-500/10 text-blue-600 border-blue-500/20">MySQL</Badge>
                  </div>
                  <div class="flex items-center gap-3 text-[12px] text-[var(--text-tertiary)]">
                    <span v-if="row.database">{{ row.database }}</span>
                    <span v-else>-</span>
                    <span class="text-[var(--border)]">·</span>
                    <span>{{ row.label }}</span>
                    <span class="text-[var(--border)]">·</span>
                    <span>保留 {{ row.retainCount || 7 }} 份</span>
                  </div>
                </div>
                <!-- Last Run -->
                <div class="text-right shrink-0 mr-2">
                  <p class="text-[12px] text-[var(--text-secondary)]">{{ row.lastRun || '尚未执行' }}</p>
                  <p class="text-[11px] text-[var(--text-tertiary)]">上次执行</p>
                </div>
                <!-- Switch -->
                <Switch
                  :model-value="row.enabled"
                  @update:model-value="(val) => { row.enabled = val; toggleSchedule(row) }"
                />
                <!-- Actions -->
                <div class="flex items-center gap-1 shrink-0">
                  <button class="btn-ghost h-[28px] text-[11px] gap-1 text-[var(--text-secondary)] hover:text-[var(--text-primary)]" @click="openScheduleDialog(row)">
                    <FileText class="h-3.5 w-3.5 shrink-0" />编辑
                  </button>
                  <button class="btn-ghost-danger h-[28px] text-[11px] gap-1" @click="confirmDeleteTarget = row; showDeleteScheduleDialog = true">
                    <Trash2 class="h-3.5 w-3.5 shrink-0" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Floating Batch Action Bar -->
    <Transition name="fade">
      <div v-if="selectedBackups.length > 0" class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 px-5 py-3 rounded-2xl bg-[var(--surface)] border border-[var(--border)] shadow-xl shadow-black/10">
        <span class="text-[13px] text-[var(--text-secondary)]">已选 <span class="font-semibold text-[var(--text-primary)]">{{ selectedBackups.length }}</span> 项</span>
        <div class="w-px h-5 bg-[var(--border)]"></div>
        <button class="btn-danger h-[30px] text-[13px]" @click="showBatchDeleteDialog = true">批量删除</button>
        <button class="btn-ghost h-[30px] text-[13px] text-[var(--text-tertiary)]" @click="selectedBackups = []">取消</button>
      </div>
    </Transition>

    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="sm:max-w-[520px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">创建备份</DialogTitle>
        </div>
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">备份类型 <span class="text-[var(--danger)]">*</span></label>
            <div class="inline-flex h-9 items-center rounded-lg bg-[var(--surface)] p-1 text-[var(--text-tertiary)]">
              <button
                v-for="opt in createLevelOptions"
                :key="opt.value"
                :class="[
                  'inline-flex items-center justify-center rounded-md px-3 py-1 text-sm font-medium transition-all',
                  createForm.backupLevel === opt.value
                    ? 'bg-[var(--background)] text-[var(--text-primary)] shadow'
                    : 'hover:text-[var(--text-primary)]'
                ]"
                @click="createForm.backupLevel = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
            <p v-if="createForm.backupLevel === 'system'" class="text-xs text-[var(--text-tertiary)] mt-1">
              备份当前所有配置（数据库列表、远程服务器、备份记录、定时计划）
            </p>
          </div>
          <div v-if="createForm.backupLevel !== 'system'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">目标数据库 <span class="text-[var(--danger)]">*</span></label>
            <Select v-model="createForm.targetDb">
              <SelectTrigger class="border-[var(--border)] shadow-none">
                <template v-if="selectedCreateDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedCreateDb.name }}
                    <Badge
                      variant="outline"
                      :class="selectedCreateDb.type === 'redis' ? 'bg-amber-500/5 text-amber-600 border-amber-500/20' : 'bg-blue-500/5 text-blue-600 border-blue-500/20'"
                      class="text-[10px] px-1.5 py-0 shrink-0"
                    >
                      {{ selectedCreateDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </Badge>
                    <span class="text-[var(--text-tertiary)] text-xs shrink-0">{{ selectedCreateDb.host || '本地' }}</span>
                  </span>
                </template>
                <SelectValue v-else placeholder="选择数据库" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="db in filteredCreateDbs"
                  :key="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name"
                  :value="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name"
                >
                  <div class="flex items-center justify-between w-full gap-2">
                    <span class="flex-1 truncate">{{ db.name }}</span>
                    <div class="flex items-center gap-2 shrink-0">
                      <Badge
                        variant="outline"
                        :class="db.type === 'redis' ? 'bg-amber-500/5 text-amber-600 border-amber-500/20' : 'bg-blue-500/5 text-blue-600 border-blue-500/20'"
                        class="text-[10px] px-1.5 py-0"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </Badge>
                      <span class="text-[var(--text-tertiary)] text-xs">{{ db.host || '本地' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <!-- MySQL 备份：选择具体要备份的数据库名 -->
          <div v-if="createForm.backupLevel === 'mysql'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">选择要备份的数据库 <span class="text-[var(--danger)]">*</span></label>
            <Select v-model="createForm.targetMysqlDbName">
              <SelectTrigger class="border-[var(--border)] shadow-none">
                <SelectValue placeholder="选择要备份的数据库" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="db in mysqlDatabaseOptions"
                  :key="db"
                  :value="db"
                >
                  {{ db === '__ALL__' ? '全部' : db }}
                </SelectItem>
              </SelectContent>
            </Select>
            <p v-if="loadingMysqlDatabases" class="text-xs text-[var(--text-tertiary)] flex items-center gap-1 mt-1">
              <Loader2 class="h-3 w-3 animate-spin" /> 正在加载数据库列表...
            </p>
            <p v-else-if="createForm.targetDb && mysqlDatabaseOptions.length === 0" class="text-xs text-[var(--warning)] mt-1">
              该连接下没有可用的数据库
            </p>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">备份名称</label>
            <Input v-model="createForm.name" placeholder="留空自动生成" class="border-[var(--border)] shadow-none" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">描述信息</label>
            <Textarea v-model="createForm.description" placeholder="可选" :rows="2" class="border-[var(--border)] shadow-none" />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <button class="btn-secondary" @click="showCreateDialog = false">取消</button>
          <button class="btn-primary" @click="submitCreate" :disabled="creating">
            <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
            确认创建
          </button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showScheduleDialog">
      <DialogContent class="sm:max-w-[520px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">{{ editScheduleData ? '编辑定时计划' : '新建定时备份' }}</DialogTitle>
        </div>
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">计划名称 <span class="text-[var(--danger)]">*</span></label>
            <Input v-model="scheduleForm.name" placeholder="如：每日数据库备份" class="border-[var(--border)] shadow-none" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">备份类型 <span class="text-[var(--danger)]">*</span></label>
            <div class="inline-flex h-9 items-center rounded-lg bg-[var(--surface)] p-1 text-[var(--text-tertiary)]">
              <button
                v-for="opt in createLevelOptions"
                :key="opt.value"
                :class="[
                  'inline-flex items-center justify-center rounded-md px-3 py-1 text-sm font-medium transition-all',
                  scheduleForm.backupLevel === opt.value
                    ? 'bg-[var(--background)] text-[var(--text-primary)] shadow'
                    : 'hover:text-[var(--text-primary)]'
                ]"
                @click="scheduleForm.backupLevel = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
          <div v-if="scheduleForm.backupLevel !== 'system'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">目标连接 <span class="text-[var(--danger)]">*</span></label>
            <Select v-model="scheduleForm.targetDb" @update:model-value="onScheduleDbChange">
              <SelectTrigger class="border-[var(--border)] shadow-none">
                <template v-if="selectedScheduleDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedScheduleDb.name }}
                    <Badge
                      variant="outline"
                      :class="selectedScheduleDb.type === 'redis' ? 'bg-amber-500/5 text-amber-600 border-amber-500/20' : 'bg-blue-500/5 text-blue-600 border-blue-500/20'"
                      class="text-[10px] px-1.5 py-0 shrink-0"
                    >
                      {{ selectedScheduleDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </Badge>
                    <span class="text-[var(--text-tertiary)] text-xs shrink-0">{{ selectedScheduleDb.host || '本地' }}</span>
                  </span>
                </template>
                <SelectValue v-else placeholder="选择数据库连接" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="db in filteredScheduleDbs"
                  :key="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name"
                  :value="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name"
                >
                  <div class="flex items-center justify-between w-full gap-2">
                    <span class="flex-1 truncate">{{ db.name }}</span>
                    <div class="flex items-center gap-2 shrink-0">
                      <Badge
                        variant="outline"
                        :class="db.type === 'redis' ? 'bg-amber-500/5 text-amber-600 border-amber-500/20' : 'bg-blue-500/5 text-blue-600 border-blue-500/20'"
                        class="text-[10px] px-1.5 py-0"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </Badge>
                      <span class="text-[var(--text-tertiary)] text-xs">{{ db.host || '本地' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="scheduleForm.backupLevel === 'mysql' && scheduleForm.targetDb" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">目标数据库 <span class="text-[var(--danger)]">*</span></label>
            <Select v-model="scheduleForm.targetMysqlDbName">
              <SelectTrigger class="border-[var(--border)] shadow-none">
                <SelectValue placeholder="选择数据库" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="db in scheduleMysqlDatabaseOptions"
                  :key="db"
                  :value="db"
                >
                  {{ db === '__ALL__' ? '全部' : db }}
                </SelectItem>
              </SelectContent>
            </Select>
            <p v-if="loadingScheduleMysqlDatabases" class="text-xs text-[var(--text-tertiary)] mt-1 flex items-center gap-1">
              <Loader2 class="h-3 w-3 animate-spin" /> 正在加载数据库列表...
            </p>
            <p v-else-if="scheduleForm.targetDb && scheduleMysqlDatabaseOptions.length === 0" class="text-xs text-[var(--warning)] mt-1">
              该连接下没有可用的数据库
            </p>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">执行周期 <span class="text-[var(--danger)]">*</span></label>
            <Select v-model="scheduleForm.cron">
              <SelectTrigger class="border-[var(--border)] shadow-none">
                <SelectValue placeholder="选择周期" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">每天</SelectItem>
                <SelectItem value="weekly">每周</SelectItem>
                <SelectItem value="monthly">每月</SelectItem>
                <SelectItem value="6">每6小时</SelectItem>
                <SelectItem value="12">每12小时</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-[var(--text-primary)]">保留份数</label>
            <Input v-model="scheduleForm.retainCount" type="number" :min="1" :max="30" class="w-32 border-[var(--border)] shadow-none" />
            <p class="text-xs text-[var(--text-tertiary)]">超过保留数量后自动删除最早的备份</p>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <button class="btn-secondary" @click="showScheduleDialog = false">取消</button>
          <button class="btn-primary" @click="submitSchedule" :disabled="scheduling">
            <Loader2 v-if="scheduling" class="h-4 w-4 animate-spin" />
            确认
          </button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-[var(--text-secondary)]">
            确定要删除备份 "{{ confirmDeleteTarget?.name }}" 吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button class="btn-secondary" @click="showDeleteDialog = false">取消</button>
          <button class="btn-danger" @click="handleDelete(confirmDeleteTarget); showDeleteDialog = false">确定删除</button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showBatchDeleteDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">批量删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-[var(--text-secondary)]">
            确定要删除选中的 {{ selectedBackups.length }} 个备份吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button class="btn-secondary" @click="showBatchDeleteDialog = false">取消</button>
          <button class="btn-danger" @click="handleBatchDelete(); showBatchDeleteDialog = false">确定删除</button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showDeleteScheduleDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-[var(--text-secondary)]">
            确定要删除计划 "{{ confirmDeleteTarget?.name }}" 吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button class="btn-secondary" @click="showDeleteScheduleDialog = false">取消</button>
          <button class="btn-danger" @click="deleteSchedule(confirmDeleteTarget); showDeleteScheduleDialog = false">确定删除</button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showRestoreDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-[var(--text-primary)]">恢复确认</DialogTitle>
          <DialogDescription class="text-[13px] text-[var(--text-secondary)]">
            {{ restoreMessage }}
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <button class="btn-secondary" @click="showRestoreDialog = false">取消</button>
          <button class="btn-danger" @click="doRestore" :disabled="restoring">
            <RefreshCw v-if="restoring" class="h-4 w-4 animate-spin" />
            确定恢复
          </button>
        </div>
      </DialogContent>
    </Dialog>

    <BackupDialog
      v-model="showImportDialog"
      :database="database"
      :import-mode="true"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'BackupView' })
import { ref, onMounted, onActivated, watch, computed, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { sourceParam } from '@/lib/instance'
import { formatLogTime } from '@/lib/utils'
import { useMessage } from '../composables/useMessage'
import {
  Upload, Clock, Plus, Settings, Database,
  HardDrive, Inbox, Loader2, Search, X,
  ArrowUpDown, ArrowUp, ArrowDown,
  ChevronLeft, ChevronRight, RotateCcw, RefreshCw, FileText,
  Download, Trash2
} from 'lucide-vue-next'
import BackupDialog from './BackupDialog.vue'

const completeProgress = inject('completeProgress')

import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Switch } from '@/components/ui/Switch.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'

const STORAGE_KEY = 'backup_view_filters'

const store = useAppContext()
const { connectionId } = storeToRefs(store)

const { success, error, warning } = useMessage()

const database = computed(() => {
  if (!store.current) return null
  return {
    id: store.serverId,
    name: store.current.name || '',
    type: store.current.type,
    host: store.current.host,
    port: store.current.port,
    isRemote: store.current.isRemote,
    username: store.current.userName,
  }
})

const props = defineProps({
  navRequest: { type: Object, default: null }
})

const emit = defineEmits(['close-import', 'navAccepted'])

const backupList = ref([])
const scheduleList = ref([])
const availableDatabases = ref([])
const instanceDatabases = ref([])
const databaseFilter = ref('all')
const showCreateDialog = ref(false)
const showScheduleDialog = ref(false)
const showImportDialog = ref(false)
const creating = ref(false)
const scheduling = ref(false)
const activeMainTab = ref('records')
const backupLevelFilter = ref('')
const editScheduleData = ref(null)
const selectedBackups = ref([])

const showDeleteDialog = ref(false)
const showBatchDeleteDialog = ref(false)
const showDeleteScheduleDialog = ref(false)
const showRestoreDialog = ref(false)
const confirmDeleteTarget = ref(null)
const restoreRow = ref(null)
const restoring = ref(false)

const sortField = ref('createdAt')
const sortOrder = ref('desc')
const currentPage = ref(1)
const pageSize = ref(20)
const totalItems = ref(0)
const loading = ref(false)
const searchQuery = ref('')
let searchTimer = null

const levelOptions = [
  { label: '全部', value: '' },
  { label: 'MySQL', value: 'mysql' },
  { label: 'Redis', value: 'redis' },
  { label: '系统', value: 'system' }
]

const createLevelOptions = [
  { label: 'MySQL备份', value: 'mysql' },
  { label: 'Redis备份', value: 'redis' },
  { label: '系统备份', value: 'system' }
]

const createForm = ref({
  name: '',
  description: '',
  backupLevel: 'mysql',
  targetDb: '',
  targetMysqlDbName: ''
})

const loadingMysqlDatabases = ref(false)
const mysqlDatabaseOptions = ref([]) // 连接下的数据库列表

const scheduleForm = ref({
  name: '',
  backupLevel: 'mysql',
  targetDb: '',
  targetMysqlDbName: '',
  cron: 'daily',
  retainCount: 7
})

const scheduleMysqlDatabaseOptions = ref([])
const loadingScheduleMysqlDatabases = ref(false)

const pageItems = computed(() => {
  return backupList.value
})

const selectedIds = computed(() => new Set(selectedBackups.value.map(b => b.id)))

const isAllSelected = computed(() => {
  return pageItems.value.length > 0 && selectedBackups.value.length === pageItems.value.length
})

const isPartialSelected = computed(() => {
  return selectedBackups.value.length > 0 && selectedBackups.value.length < pageItems.value.length
})

const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)))

const visiblePageNumbers = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const pages = []
  pages.push(1)
  if (current > 3) pages.push('...')
  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  if (current < total - 2) pages.push('...')
  pages.push(total)
  return pages
})

const hasActiveFilters = computed(() => {
  return backupLevelFilter.value !== '' || (databaseFilter.value && databaseFilter.value !== 'all') || searchQuery.value !== ''
})

const allDbOptions = computed(() => {
  const seen = new Map()
  const result = []
  for (const item of [...instanceDatabases.value, ...availableDatabases.value]) {
    const isRemote = !!item.isRemote
    const uid = (isRemote ? 'r:' : 'l:') + item.id + ':' + item.name
    if (!seen.has(uid)) {
      seen.set(uid, true)
      result.push({ ...item, isRemote })
    }
  }
  return result
})

const filteredCreateDbs = computed(() => {
  const level = createForm.value.backupLevel
  return allDbOptions.value.filter(db => {
    if (level === 'mysql') return db.type !== 'redis'
    if (level === 'redis') return db.type === 'redis'
    return true
  })
})

const filteredScheduleDbs = computed(() => {
  const level = scheduleForm.value.backupLevel
  return allDbOptions.value.filter(db => {
    if (level === 'mysql') return db.type !== 'redis'
    if (level === 'redis') return db.type === 'redis'
    return true
  })
})

const selectedCreateDb = computed(() => {
  return filteredCreateDbs.value.find(db =>
    (db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name === createForm.value.targetDb
  )
})

const selectedScheduleDb = computed(() => {
  return filteredScheduleDbs.value.find(db =>
    (db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name === scheduleForm.value.targetDb
  )
})

const restoreMessage = computed(() => {
  if (!restoreRow.value) return ''
  const row = restoreRow.value
  const isSystem = row.backupLevel === 'system'
  const isRedis = row.backupLevel === 'redis'
  const target = row.database || '系统配置'
  return `确定要从备份 "${row.name}" 恢复${isRedis ? 'Redis数据' : isSystem ? '系统配置' : 'MySQL数据库 "' + target + '"'} 吗？恢复操作不可撤销！`
})

const parseDbUid = (uid) => {
  if (!uid) return null
  const parts = uid.split(':')
  if (parts.length < 2) return null
  const isRemote = parts[0] === 'r'
  const id = parseInt(parts[1], 10)
  const name = parts.length >= 3 ? parts.slice(2).join(':') : ''
  return { isRemote, id, name }
}

const findDbByUid = (uid) => {
  const parsed = parseDbUid(uid)
  if (!parsed) return null
  return allDbOptions.value.find(db => {
    return db.isRemote === parsed.isRemote && db.id === parsed.id && db.name === parsed.name
  })
}

const formatSize = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return size.toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

const saveFilterState = () => {
  try {
    const state = {
      backupLevelFilter: backupLevelFilter.value,
      databaseFilter: databaseFilter.value,
      sortField: sortField.value,
      sortOrder: sortOrder.value,
      pageSize: pageSize.value,
      searchQuery: searchQuery.value
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch (e) { /* ignore */ }
}

const loadFilterState = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const state = JSON.parse(saved)
      if (state.backupLevelFilter !== undefined) backupLevelFilter.value = state.backupLevelFilter
      if (state.databaseFilter) databaseFilter.value = state.databaseFilter
      if (state.sortField) sortField.value = state.sortField
      if (state.sortOrder) sortOrder.value = state.sortOrder
      if (state.pageSize) pageSize.value = state.pageSize
      if (state.searchQuery) searchQuery.value = state.searchQuery
    }
  } catch (e) { /* ignore */ }
}

const toggleSort = (field) => {
  if (sortField.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortOrder.value = 'asc'
  }
  currentPage.value = 1
  saveFilterState()
  loadBackups()
}

const resetSort = () => {
  sortField.value = 'createdAt'
  sortOrder.value = 'desc'
  currentPage.value = 1
  saveFilterState()
  loadBackups()
}

const onFilterChange = () => {
  currentPage.value = 1
  saveFilterState()
  loadBackups()
}

const onSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    currentPage.value = 1
    saveFilterState()
    loadBackups()
  }, 300)
}

const clearAllFilters = () => {
  backupLevelFilter.value = ''
  databaseFilter.value = 'all'
  searchQuery.value = ''
  currentPage.value = 1
  saveFilterState()
  loadBackups()
}

const goToPage = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  loadBackups()
}

const loadBackups = () => {
  loading.value = true
  const params = new URLSearchParams()
  params.set('page', String(currentPage.value))
  params.set('page_size', String(pageSize.value))
  params.set('sort_field', sortField.value)
  params.set('sort_order', sortOrder.value)

  if (databaseFilter.value && databaseFilter.value !== 'all') {
    const parsed = parseDbUid(databaseFilter.value)
    if (parsed) {
      params.set('server_id', String(parsed.id))
      if (parsed.isRemote) params.set('source', 'remote')
      if (parsed.name) params.set('database', parsed.name)
    }
  }
  if (backupLevelFilter.value) params.set('level', backupLevelFilter.value)
  if (searchQuery.value) params.set('search', searchQuery.value)

  const qs = params.toString()
  fetch(`/api/backups${qs ? '?' + qs : ''}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        backupList.value = (data.data.items || []).filter(b => b.id)
        totalItems.value = data.data.total || 0
      } else {
        backupList.value = []
        totalItems.value = 0
      }
    })
    .catch((e) => {
      console.error(e)
      backupList.value = []
      totalItems.value = 0
    })
    .finally(() => { loading.value = false; completeProgress?.() })
}

const loadSchedules = () => {
  fetch('/api/backups/scheduled')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        scheduleList.value = (data.data || []).map(s => ({
          ...s,
          label: cronLabel(s.cron),
          enabled: s.enabled !== false
        }))
      }
    })
    .catch((e) => { console.error(e) })
}

const loadAvailableDatabases = () => {
  return Promise.all([
    fetch('/api/databases/db/list/all').then(r => r.json()),
    fetch('/api/remote-servers').then(r => r.json())
  ]).then(([localRes, remoteRes]) => {
    const locals = (localRes.code === 0 ? localRes.data : []) || []
    const remotes = (remoteRes.code === 0 ? remoteRes.data : []) || []
    availableDatabases.value = [...locals.map(r => ({ ...r, isRemote: false })), ...remotes.map(r => ({ ...r, isRemote: true }))]
    if (databaseFilter.value && databaseFilter.value !== 'all') {
      const parsed = parseDbUid(databaseFilter.value)
      if (parsed) {
        const allOpts = [...locals.map(r => ({ ...r, isRemote: false })), ...remotes.map(r => ({ ...r, isRemote: true }))]
        const valid = allOpts.some(opt => opt.isRemote === parsed.isRemote && opt.id === parsed.id && (!parsed.name || opt.name === parsed.name))
        if (!valid) {
          databaseFilter.value = 'all'
          saveFilterState()
          loadBackups()
        }
      }
    }
  }).catch((e) => { console.error(e) })
}

const loadInstanceDatabases = () => {
  if (!database.value) return
  const db = database.value
  if (db.type === 'redis') return
  const source = db.isRemote ? 'remote' : 'local'
  fetch(`/api/mysql/databases?server_id=${db.id}&source=${source}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        const filtered = data.data.filter(n => !['information_schema', 'performance_schema', 'mysql', 'sys'].includes(n))
        instanceDatabases.value = filtered.map(name => ({
          id: db.id, name, type: 'mysql', host: db.host, port: db.port, isRemote: db.isRemote,
        }))
      }
    })
    .catch(() => {})
}

// 加载创建备份选择的连接的数据库列表
const loadCreateMysqlDatabases = (selectedDb) => {
  if (!selectedDb || selectedDb.type === 'redis') {
    mysqlDatabaseOptions.value = []
    createForm.value.targetMysqlDbName = ''
    return
  }
  loadingMysqlDatabases.value = true
  const source = selectedDb.isRemote ? 'remote' : 'local'
  fetch(`/api/mysql/databases?server_id=${selectedDb.id}&source=${source}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        const filtered = data.data.filter(n => !['information_schema', 'performance_schema', 'mysql', 'sys'].includes(n))
        const isRoot = selectedDb.username === 'root'
        // 只有 root 用户才显示「全部」选项
        mysqlDatabaseOptions.value = isRoot ? ['__ALL__', ...filtered] : filtered
        // 尝试自动选中：优先选择用户当前正在看的数据库（store.dbName）
        if (store.dbName && filtered.includes(store.dbName)) {
          createForm.value.targetMysqlDbName = store.dbName
        } else if (selectedDb.database && filtered.includes(selectedDb.database)) {
          createForm.value.targetMysqlDbName = selectedDb.database
        } else if (filtered.length > 0) {
          createForm.value.targetMysqlDbName = filtered[0]
        } else if (isRoot) {
          createForm.value.targetMysqlDbName = '__ALL__'
        } else {
          createForm.value.targetMysqlDbName = ''
        }
      }
    })
    .catch(() => {
      mysqlDatabaseOptions.value = []
      createForm.value.targetMysqlDbName = ''
    })
    .finally(() => { loadingMysqlDatabases.value = false })
}

// 监听 targetDb 变化
watch(() => createForm.value.targetDb, (newVal) => {
  const selectedDb = findDbByUid(newVal)
  loadCreateMysqlDatabases(selectedDb)
})

const onScheduleDbChange = (newVal) => {
  scheduleForm.value.targetMysqlDbName = ''
  scheduleMysqlDatabaseOptions.value = []
  if (!newVal) return
  const selectedDb = findDbByUid(newVal)
  if (!selectedDb || selectedDb.type === 'redis') return
  loadingScheduleMysqlDatabases.value = true
  const source = selectedDb.isRemote ? 'remote' : 'local'
  fetch(`/api/mysql/databases?server_id=${selectedDb.id}&source=${source}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        const filtered = data.data.filter(n => !['information_schema', 'performance_schema', 'mysql', 'sys'].includes(n))
        const isRoot = selectedDb.username === 'root'
        // 只有 root 用户才显示「全部」选项
        scheduleMysqlDatabaseOptions.value = isRoot ? ['__ALL__', ...filtered] : filtered
        if (store.dbName && filtered.includes(store.dbName)) {
          scheduleForm.value.targetMysqlDbName = store.dbName
        } else if (filtered.length > 0) {
          scheduleForm.value.targetMysqlDbName = filtered[0]
        } else if (isRoot) {
          scheduleForm.value.targetMysqlDbName = '__ALL__'
        } else {
          scheduleForm.value.targetMysqlDbName = ''
        }
      }
    })
    .catch(() => {
      scheduleMysqlDatabaseOptions.value = []
    })
    .finally(() => { loadingScheduleMysqlDatabases.value = false })
}

const cronLabel = (cron) => {
  const map = { daily: '每天', weekly: '每周', monthly: '每月' }
  if (map[cron]) return map[cron]
  return '每' + cron + '小时'
}

const processNavRequest = () => {
  if (!props.navRequest) return
  const nav = props.navRequest
  const dbFilter = (nav.isRemote ? 'r:' : 'l:') + String(nav.id) + (nav.name ? ':' + nav.name : '')
  databaseFilter.value = dbFilter
  saveFilterState()
  loadBackups()
  emit('navAccepted')
}

watch(connectionId, () => {
  activeMainTab.value = 'records'
  backupLevelFilter.value = ''
  databaseFilter.value = 'all'
  searchQuery.value = ''
  currentPage.value = 1
  sortField.value = 'createdAt'
  sortOrder.value = 'desc'
  loadBackups()
  loadSchedules()
  loadAvailableDatabases()
})

watch(() => props.navRequest, (val) => {
  if (val) processNavRequest()
})

watch(() => createForm.value.backupLevel, () => {
  createForm.value.targetDb = ''
})

watch(() => scheduleForm.value.backupLevel, () => {
  scheduleForm.value.targetDb = ''
})

const handleCreate = () => {
  const defaultLevel = database.value
    ? (database.value.type === 'redis' ? 'redis' : 'mysql')
    : 'system'
  
  // 构建 default target db uid
  let targetUid = ''
  if (database.value && store.name) {
    const foundDb = allDbOptions.value.find(db => db.name === store.name && db.id === database.value.id)
    if (foundDb) {
      targetUid = (foundDb.isRemote ? 'r:' : 'l:') + foundDb.id + ':' + foundDb.name
    }
  }
  
  // 获取当前 store 中的具体数据库名
  let targetMysqlDbName = ''
  if (database.value && database.value.type === 'mysql') {
    // store.dbName 就是用户当前选择的具体数据库名（如 test none）
    targetMysqlDbName = store.dbName
  }
  
  createForm.value = {
    name: '',
    description: '',
    backupLevel: defaultLevel,
    targetDb: targetUid,
    targetMysqlDbName: targetMysqlDbName
  }
  if (database.value && database.value.type === 'mysql' && targetUid) {
    // 如果已经选择了连接，直接加载该连接的数据库列表
    const selectedDb = findDbByUid(targetUid)
    loadCreateMysqlDatabases(selectedDb)
  } else if (database.value && database.value.type === 'mysql') {
    loadInstanceDatabases()
  }
  showCreateDialog.value = true
}

const handleImport = () => {
  showImportDialog.value = true
}

const toggleSelectAll = (e) => {
  if (e.target.checked) {
    selectedBackups.value = [...pageItems.value]
  } else {
    selectedBackups.value = []
  }
}

const toggleSelect = (row) => {
  const idx = selectedBackups.value.findIndex(b => b.id === row.id)
  if (idx >= 0) {
    selectedBackups.value.splice(idx, 1)
  } else {
    selectedBackups.value.push(row)
  }
}

const handleBatchDelete = () => {
  const ids = selectedBackups.value.map(b => b.id)
  if (ids.length === 0) { warning('请选择要删除的备份'); return }
  fetch('/api/backups', {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids })
  }).then(r => r.json()).then(result => {
    if (result.code === 0) {
      success(result.msg || `成功删除 ${ids.length} 个备份`)
    } else {
      error(result.msg || '删除失败')
    }
    selectedBackups.value = []
    loadBackups()
  }).catch(() => {
    error('删除失败')
  })
}

const submitCreate = () => {
  creating.value = true
  const isSystem = createForm.value.backupLevel === 'system'
  const isRedis = createForm.value.backupLevel === 'redis'
  
  // 根据选择的目标数据库 uid，找到对应的完整数据库对象
  let selectedDb = null
  if (!isSystem) {
    const targetUid = createForm.value.targetDb
    if (!targetUid) {
      warning('请选择目标数据库')
      creating.value = false
      return
    }
    selectedDb = findDbByUid(targetUid)
    if (!selectedDb) {
      warning('请选择目标数据库')
      creating.value = false
      return
    }
  }
  
  // 获取正确的数据库名
  let targetDb = ''
  if (isRedis) {
    targetDb = selectedDb?.name || ''
  } else if (!isSystem) {
    targetDb = createForm.value.targetMysqlDbName
    if (!targetDb) {
      warning('请选择要备份的MySQL数据库名')
      creating.value = false
      return
    }
  }
  
  const dbType = selectedDb?.type || (isRedis ? 'redis' : 'mysql')

  if (isRedis && selectedDb?.type === 'redis') {
    fetch(`/api/redis/backup?${sourceParam(selectedDb?.isRemote || false)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        server_id: selectedDb?.id || 0,
        database: 0,
        name: createForm.value.name || ''
      })
    })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success('Redis备份创建成功')
          showCreateDialog.value = false
          loadBackups()
        } else {
          error(data.msg || '创建失败')
        }
      })
      .catch(() => { error('创建失败') })
      .finally(() => { creating.value = false })
    return
  }

  fetch('/api/backups', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: createForm.value.name || '',
      description: createForm.value.description,
      backupLevel: createForm.value.backupLevel,
      database: isSystem ? '' : targetDb,
      serverId: selectedDb?.id || 0,
      source: selectedDb?.isRemote ? 'remote' : 'local',
      type: dbType,
      host: selectedDb?.host || '',
      port: selectedDb?.port || (isRedis ? 6379 : 3306)
    })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        const label = isSystem ? '系统' : isRedis ? 'Redis' : 'MySQL'
        success(label + '备份创建成功')
        showCreateDialog.value = false
        loadBackups()
      } else {
        error(data.msg || '创建失败')
      }
    })
    .catch(() => {
      error('创建失败')
    })
    .finally(() => {
      creating.value = false
    })
}

const handleDownload = (row) => {
  window.open(`/api/backups/${row.id}`, '_blank')
}

const handleDelete = (row) => {
  if (!row) return
  fetch(`/api/backups/${row.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success('删除成功')
        loadBackups()
      } else {
        error(data.msg || '删除失败')
      }
    })
    .catch(() => {
      error('删除失败')
    })
}

const handleRestore = (row) => {
  restoreRow.value = row
  showRestoreDialog.value = true
}

const doRestore = () => {
  const row = restoreRow.value
  if (!row) return
  restoring.value = true
  
  const isRedis = row.backupLevel === 'redis'
  
  let source = row.source || 'local'
  
  const endpoint = isRedis ? `/api/redis/restore?source=${source}` : '/api/backups/restore'
  fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ backup_id: row.id, source })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(data.msg || '恢复成功')
        showRestoreDialog.value = false
        loadBackups()
      } else {
        error(data.msg || '恢复失败')
      }
    })
    .catch(err => {
      error('恢复失败: ' + err.message)
    })
    .finally(() => {
      restoring.value = false
    })
}

const openScheduleDialog = (row) => {
  editScheduleData.value = row || null
  scheduleMysqlDatabaseOptions.value = []
  
  const getTargetDbUid = (dbName, level) => {
    if (!dbName || level === 'system') return ''
    const foundDb = allDbOptions.value.find(db => db.name === dbName)
    return foundDb ? (foundDb.isRemote ? 'r:' : 'l:') + foundDb.id + ':' + foundDb.name : ''
  }
  
  if (row) {
    const targetDbUid = getTargetDbUid(row.database, row.backupLevel)
    scheduleForm.value = {
      name: row.name,
      backupLevel: row.backupLevel || 'mysql',
      targetDb: targetDbUid,
      targetMysqlDbName: row.backupLevel === 'mysql' ? (row.database || '__ALL__') : '',
      cron: row.cron || 'daily',
      retainCount: row.retainCount || 7
    }
    if (targetDbUid && row.backupLevel === 'mysql') {
      onScheduleDbChange(targetDbUid)
    }
  } else {
    const dbName = database.value ? (store.dbName || '') : ''
    const targetDbUid = getTargetDbUid(dbName, 'mysql')
    scheduleForm.value = {
      name: '',
      backupLevel: 'mysql',
      targetDb: targetDbUid,
      targetMysqlDbName: '',
      cron: 'daily',
      retainCount: 7
    }
    if (targetDbUid) {
      onScheduleDbChange(targetDbUid)
    }
  }
  showScheduleDialog.value = true
}

const submitSchedule = () => {
  scheduling.value = true
  const isSystem = scheduleForm.value.backupLevel === 'system'
  
  // 根据选择的目标数据库 uid，找到对应的完整数据库对象
  let selectedDb = null
  if (!isSystem) {
    const targetUid = scheduleForm.value.targetDb
    if (!targetUid) {
      warning('请选择目标连接')
      scheduling.value = false
      return
    }
    selectedDb = findDbByUid(targetUid)
    if (!selectedDb) {
      warning('请选择目标连接')
      scheduling.value = false
      return
    }
    if (selectedDb.type !== 'redis' && !scheduleForm.value.targetMysqlDbName) {
      warning('请选择目标数据库')
      scheduling.value = false
      return
    }
  }
  
  const targetDb = isSystem ? '' : (selectedDb?.type === 'redis' ? selectedDb?.name : scheduleForm.value.targetMysqlDbName)

  const body = {
    name: scheduleForm.value.name,
    backupLevel: scheduleForm.value.backupLevel,
    database: targetDb,
    serverId: selectedDb?.id || 0,
    source: selectedDb?.isRemote ? 'remote' : 'local',
    cron: scheduleForm.value.cron,
    label: cronLabel(scheduleForm.value.cron),
    retainCount: scheduleForm.value.retainCount
  }
  if (editScheduleData.value) {
    body.id = editScheduleData.value.id
  }

  fetch('/api/backups/scheduled', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(editScheduleData.value ? '更新成功' : '创建成功')
        showScheduleDialog.value = false
        editScheduleData.value = null
        loadSchedules()
      } else {
        error(data.msg || '操作失败')
      }
    })
    .catch(() => { error('操作失败') })
    .finally(() => { scheduling.value = false })
}

const toggleSchedule = (row) => {
  const newEnabled = row.enabled
  fetch('/api/backups/scheduled', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: row.id, enabled: newEnabled })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(newEnabled ? '已启用定时备份' : '已禁用定时备份')
        loadSchedules()
      } else {
        row.enabled = !newEnabled
        error(data.msg || '操作失败')
      }
    })
    .catch(() => {
      row.enabled = !newEnabled
      error('操作失败')
    })
}

const deleteSchedule = (row) => {
  if (!row) return
  fetch(`/api/backups/scheduled/${row.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success('删除成功')
        loadSchedules()
      } else {
        error(data.msg || '删除失败')
      }
    })
    .catch(() => {
      error('删除失败')
    })
}

onMounted(() => {
  loadFilterState()
  loadBackups()
  loadSchedules()
  loadAvailableDatabases()
})

onActivated(() => {
  loadBackups()
  loadSchedules()
})
</script>
