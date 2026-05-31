<template>
  <div class="page-padding h-full flex flex-col overflow-hidden">
    <div class="content-card flex flex-col flex-1 min-h-0">
      <div class="content-header shrink-0">
        <div class="flex items-start justify-between">
          <div>
            <h2 class="text-[15px] font-semibold text-foreground">备份管理</h2>
            <p class="text-[13px] text-muted-foreground mt-0.5">管理数据库备份与定时计划</p>
          </div>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" class="h-[32px] text-[13px] max-sm:hidden" @click="handleImport">
              <Upload class="h-3.5 w-3.5 mr-1.5" />导入备份
            </Button>
            <Button variant="outline" size="sm" class="h-[32px] text-[13px] max-sm:hidden" @click="openScheduleDialog()">
              <Clock class="h-3.5 w-3.5 mr-1.5" />新建计划
            </Button>
            <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="handleCreate">
              <Plus class="h-3.5 w-3.5 mr-1.5" />创建备份
            </Button>
          </div>
        </div>
      </div>

      <div class="border-t border-border section-padding shrink-0">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <button
              :class="[
                'inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-[13px] font-semibold transition-all duration-200 relative overflow-hidden',
                activeMainTab === 'records'
                  ? 'bg-gradient-to-r from-[#1A1A1A] to-[#333] text-white shadow-lg shadow-[#1A1A1A]/20'
                  : 'bg-muted text-muted-foreground hover:bg-[#EAEAEA] hover:text-secondary-foreground'
              ]"
              @click="activeMainTab = 'records'"
            >
              <FileText class="h-4 w-4" />备份记录
              <span v-if="activeMainTab === 'records'" class="absolute inset-0 bg-white/10 animate-pulse pointer-events-none"></span>
            </button>
            <button
              :class="[
                'inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-[13px] font-semibold transition-all duration-200 relative overflow-hidden',
                activeMainTab === 'scheduled'
                  ? 'bg-gradient-to-r from-[#1A1A1A] to-[#333] text-white shadow-lg shadow-[#1A1A1A]/20'
                  : 'bg-muted text-muted-foreground hover:bg-[#EAEAEA] hover:text-secondary-foreground'
              ]"
              @click="activeMainTab = 'scheduled'"
            >
              <Clock class="h-4 w-4" />定时备份
              <span v-if="activeMainTab === 'scheduled'" class="absolute inset-0 bg-white/10 animate-pulse pointer-events-none"></span>
            </button>
          </div>
          <div v-if="activeMainTab === 'records'" class="flex items-center gap-2 flex-wrap">
            <div class="flex h-[32px] items-center rounded-lg border border-border bg-white px-2 gap-1">
              <Search class="h-3.5 w-3.5 text-muted-foreground" />
              <Input v-model="searchQuery" placeholder="搜索备份" class="border-0 shadow-none h-[28px] text-[13px] w-[140px] bg-transparent" @input="onSearchInput" />
            </div>
            <Select v-model="databaseFilter" @update:model-value="onFilterChange">
              <SelectTrigger class="h-[32px] w-[150px] text-[13px] border-border">
                <Database class="h-3.5 w-3.5 mr-1" />
                <SelectValue placeholder="全部数据库" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部数据库</SelectItem>
                <SelectItem v-for="db in allDbOptions" :key="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name" :value="(db.isRemote ? 'r:' : 'l:') + db.id + ':' + db.name">
                  <div class="flex items-center gap-2">
                    <Badge :class="db.type === 'redis' ? 'bg-amber-50 text-amber-600 border-amber-200' : 'bg-blue-50 text-blue-600 border-blue-200'" class="text-[10px] px-1.5 py-0 rounded-full">
                      {{ db.type === 'redis' ? 'R' : 'M' }}
                    </Badge>
                    {{ db.name }}
                    <span class="text-muted-foreground text-xs ml-auto">{{ db.host || '本地' }}</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="sm" class="h-[32px] text-[13px]" @click="loadBackups">
              <RefreshCw class="h-3.5 w-3.5 mr-1.5" />刷新
            </Button>
            <Button v-if="hasActiveFilters" variant="ghost" size="sm" class="h-[32px] text-[13px] text-muted-foreground" @click="clearAllFilters">
              <X class="h-3.5 w-3.5 mr-1.5" />清除筛选
            </Button>
          </div>
        </div>
      </div>

      <div v-if="activeMainTab === 'records'" class="border-t border-border section-padding shrink-0">
        <div class="flex items-center gap-2">
          <span class="text-[12px] text-muted-foreground">备份级别：</span>
          <Button v-for="opt in levelOptions" :key="opt.value" variant="ghost" size="sm"
            :class="[backupLevelFilter === opt.value ? 'bg-[#1A1A1A] text-white hover:bg-[#333]' : 'text-secondary-foreground hover:bg-muted', 'h-[28px] text-[12px] gap-1.5']"
            @click="backupLevelFilter = opt.value; onFilterChange()">
            <Database v-if="opt.value === 'mysql'" class="h-3 w-3 shrink-0 text-primary" />
            <HardDrive v-else-if="opt.value === 'redis'" class="h-3 w-3 shrink-0 text-[#e6a23c]" />
            <Settings v-else-if="opt.value === 'system'" class="h-3 w-3 shrink-0 text-muted-foreground" />
            {{ opt.label }}
          </Button>
        </div>
      </div>

      <div class="flex-1 min-h-0" style="padding: 0 var(--section-padding-x) var(--section-padding-y)">
        <div v-if="activeMainTab === 'records'" class="h-full flex flex-col">
          <div class="flex-1 min-h-0 overflow-y-auto">
            <div v-if="loading" class="flex items-center justify-center py-16">
              <Loader2 class="h-5 w-5 text-primary animate-spin mr-2" />
              <span class="text-[13px] text-primary">加载中...</span>
            </div>
            <div v-else-if="pageItems.length === 0" class="flex items-center justify-center py-16">
              <Inbox class="h-8 w-8 text-muted-foreground/40 mr-2" />
              <span class="text-[13px] text-muted-foreground">暂无备份记录</span>
            </div>
            <Table v-else :class="{ 'opacity-40 pointer-events-none': loading }">
              <TableHeader>
                <TableRow class="hover:bg-transparent border-b border-[#F0F0F0]">
                  <TableHead class="w-10 text-[12px] font-normal text-muted-foreground h-10">
                    <input type="checkbox" class="h-[15px] w-[15px] rounded border-border accent-[#4facfe] cursor-pointer" :checked="isAllSelected" :indeterminate="isPartialSelected" @change="toggleSelectAll" />
                  </TableHead>
                  <TableHead class="min-w-[220px] text-[12px] font-normal text-muted-foreground h-10 cursor-pointer select-none hover:text-foreground transition-colors" @click="toggleSort('name')">
                    <span class="flex items-center gap-1">
                      备份名称
                      <ArrowUpDown v-if="sortField !== 'name'" class="h-3 w-3 opacity-40" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-foreground" />
                      <ArrowDown v-else class="h-3 w-3 text-foreground" />
                    </span>
                  </TableHead>
                  <TableHead class="w-[56px] text-[12px] font-normal text-muted-foreground h-10">状态</TableHead>
                  <TableHead class="min-w-[100px] text-[12px] font-normal text-muted-foreground h-10">目标数据库</TableHead>
                  <TableHead class="w-[80px] text-[12px] font-normal text-muted-foreground h-10 cursor-pointer select-none hover:text-foreground transition-colors" @click="toggleSort('fileSize')">
                    <span class="flex items-center gap-1">
                      大小
                      <ArrowUpDown v-if="sortField !== 'fileSize'" class="h-3 w-3 opacity-40" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-foreground" />
                      <ArrowDown v-else class="h-3 w-3 text-foreground" />
                    </span>
                  </TableHead>
                  <TableHead class="min-w-[145px] text-[12px] font-normal text-muted-foreground h-10 cursor-pointer select-none hover:text-foreground transition-colors" @click="toggleSort('createdAt')">
                    <span class="flex items-center gap-1">
                      创建时间
                      <ArrowUpDown v-if="sortField !== 'createdAt'" class="h-3 w-3 opacity-40" />
                      <ArrowUp v-else-if="sortOrder === 'asc'" class="h-3 w-3 text-foreground" />
                      <ArrowDown v-else class="h-3 w-3 text-foreground" />
                    </span>
                  </TableHead>
                  <TableHead class="text-center min-w-[200px] text-[12px] font-normal text-muted-foreground h-10">
                    <span class="inline-flex items-center gap-1">
                      操作
                      <button
                        v-if="sortField"
                        class="ml-1 inline-flex items-center gap-0.5 text-[11px] text-muted-foreground hover:text-primary transition-colors"
                        @click.stop="resetSort"
                      >
                        <RotateCcw class="h-2.5 w-2.5" />重置
                      </button>
                    </span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="row in pageItems" :key="row.id" :class="{ 'bg-muted': selectedIds.has(row.id) }" class="hover:bg-muted transition-colors duration-150 border-b border-[#F0F0F0]">
                  <TableCell>
                    <input type="checkbox" class="h-[15px] w-[15px] rounded border-border accent-[#4facfe] cursor-pointer" :checked="selectedIds.has(row.id)" @change="toggleSelect(row)" />
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center gap-1.5">
                      <Settings v-if="row.backupLevel === 'system'" class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                      <HardDrive v-else-if="row.backupLevel === 'redis'" class="h-3.5 w-3.5 text-[#e6a23c] shrink-0" />
                      <Database v-else class="h-3.5 w-3.5 text-primary shrink-0" />
                      <span class="text-[13px] truncate text-foreground max-w-[180px] inline-block" :title="row.name">{{ row.name }}</span>
                      <Badge v-if="row.backupType === 'import'" variant="outline" class="text-[10px] py-0 bg-[#16a34a]/10 text-[#16a34a] border-[#16a34a]/20 shrink-0">导入</Badge>
                      <Badge v-else-if="row.backupType === 'scheduled'" variant="secondary" class="text-[10px] py-0 bg-muted text-foreground shrink-0">定时</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div class="inline-flex items-center gap-1">
                      <span v-if="row.status === 'success'" class="inline-flex items-center gap-1 text-[11px] text-[#16a34a]">
                        <span class="h-1.5 w-1.5 rounded-full bg-[#16a34a]"></span>
                        成功
                      </span>
                      <span v-else class="inline-flex items-center gap-1 text-[11px] text-red-500">
                        <span class="h-1.5 w-1.5 rounded-full bg-red-500"></span>
                        失败
                      </span>
                    </div>
                  </TableCell>
                  <TableCell><span class="text-[13px] text-foreground" v-if="row.database">{{ row.database }}</span><span class="text-[13px] text-muted-foreground" v-else>-</span></TableCell>
                  <TableCell><span class="text-[13px] font-mono-data text-foreground">{{ formatSize(row.fileSize) }}</span></TableCell>
                  <TableCell><span class="text-[13px] text-muted-foreground">{{ formatLogTime(row.createdAt) }}</span></TableCell>
                  <TableCell>
                    <div class="flex items-center justify-center h-full">
                      <div class="inline-flex items-center rounded-lg border border-border bg-muted/50 p-0.5">
                        <button class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] text-foreground/70 hover:text-foreground hover:bg-white transition-all whitespace-nowrap leading-none" @click="handleRestore(row)">
                          <RotateCcw class="h-3.5 w-3.5 shrink-0" />恢复
                        </button>
                        <button class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] text-foreground/70 hover:text-foreground hover:bg-white transition-all whitespace-nowrap leading-none" @click="handleDownload(row)">
                          <Download class="h-3.5 w-3.5 shrink-0" />下载
                        </button>
                        <div class="w-px h-4 bg-border mx-0.5 shrink-0"></div>
                        <button class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] text-red-400 hover:text-red-500 hover:bg-white transition-all whitespace-nowrap leading-none" @click="confirmDeleteTarget = row; showDeleteDialog = true">
                          <Trash2 class="h-3.5 w-3.5 shrink-0" />删除
                        </button>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
          <div v-if="pageItems.length > 0" class="flex items-center justify-between mt-2 shrink-0">
            <span class="text-[12px] text-muted-foreground">共 {{ totalItems }} 条，第 {{ currentPage }}/{{ totalPages }} 页</span>
            <div class="flex items-center gap-1">
              <Button variant="outline" size="sm" class="h-7 px-2 text-[12px]" :disabled="currentPage <= 1" @click="goToPage(currentPage - 1)">
                <ChevronLeft class="h-3.5 w-3.5" />
              </Button>
              <template v-for="p in visiblePageNumbers" :key="p">
                <span v-if="p === '...'" class="px-1 text-[12px] text-muted-foreground">...</span>
                <Button
                  v-else
                  variant="outline"
                  size="sm"
                  class="h-7 min-w-[28px] px-1 text-[12px]"
                  :class="p === currentPage ? 'bg-foreground text-white border-foreground hover:bg-[#333]' : ''"
                  @click="goToPage(p)"
                >{{ p }}</Button>
              </template>
              <Button variant="outline" size="sm" class="h-7 px-2 text-[12px]" :disabled="currentPage >= totalPages" @click="goToPage(currentPage + 1)">
                <ChevronRight class="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
          <div v-if="selectedBackups.length > 0" class="flex items-center gap-2 mt-2 shrink-0">
            <span class="text-[13px] text-muted-foreground">已选 {{ selectedBackups.length }} 项</span>
            <Button variant="destructive" size="sm" class="h-7 text-[13px]" @click="showBatchDeleteDialog = true">批量删除</Button>
            <Button variant="outline" size="sm" class="h-7 text-[13px]" @click="selectedBackups = []">取消选择</Button>
          </div>
        </div>

        <div v-if="activeMainTab === 'scheduled'" class="h-full flex flex-col">
          <div class="flex-1 min-h-0 overflow-y-auto">
            <div v-if="scheduleList.length === 0" class="flex items-center justify-center py-16">
              <Inbox class="h-8 w-8 text-muted-foreground/40 mr-2" />
              <span class="text-[13px] text-muted-foreground">暂无定时备份计划</span>
            </div>
            <Table v-else>
              <TableHeader>
                <TableRow class="hover:bg-transparent border-b border-[#F0F0F0]">
                  <TableHead class="min-w-[160px] text-[12px] font-normal text-muted-foreground">计划名称</TableHead>
                  <TableHead class="w-[75px] text-[12px] font-normal text-muted-foreground">级别</TableHead>
                  <TableHead class="min-w-[120px] text-[12px] font-normal text-muted-foreground">目标数据库</TableHead>
                  <TableHead class="w-[90px] text-[12px] font-normal text-muted-foreground">执行周期</TableHead>
                  <TableHead class="w-[70px] text-[12px] font-normal text-muted-foreground">状态</TableHead>
                  <TableHead class="w-[70px] text-[12px] font-normal text-muted-foreground">保留</TableHead>
                  <TableHead class="min-w-[160px] text-[12px] font-normal text-muted-foreground">上次执行</TableHead>
                  <TableHead class="text-center min-w-[140px] text-[12px] font-normal text-muted-foreground">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="row in scheduleList" :key="row.id" class="hover:bg-muted border-b border-[#F0F0F0]">
                  <TableCell>
                    <span class="truncate text-[13px] text-foreground">{{ row.name }}</span>
                  </TableCell>
                  <TableCell>
                    <Badge v-if="row.backupLevel === 'system'" variant="outline" class="bg-amber-50 text-amber-600 border-amber-200">系统</Badge>
                    <Badge v-else-if="row.backupLevel === 'redis'" variant="outline" class="bg-amber-50 text-amber-600 border-amber-200">Redis</Badge>
                    <Badge v-else variant="outline" class="bg-blue-50 text-blue-600 border-blue-200">MySQL</Badge>
                  </TableCell>
                  <TableCell>
                    <span v-if="row.database" class="text-foreground">{{ row.database }}</span>
                    <span v-else class="text-muted-foreground">-</span>
                  </TableCell>
                  <TableCell class="text-foreground">{{ row.label }}</TableCell>
                  <TableCell>
                    <Switch
                      :model-value="row.enabled"
                      @update:model-value="(val) => { row.enabled = val; toggleSchedule(row) }"
                    />
                  </TableCell>
                  <TableCell class="text-foreground">{{ row.retainCount || 7 }} 份</TableCell>
                  <TableCell>
                    <span v-if="row.lastRun" class="text-foreground">{{ row.lastRun }}</span>
                    <span v-else class="text-muted-foreground">尚未执行</span>
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center justify-center h-full">
                      <div class="inline-flex items-center rounded-lg border border-border bg-muted/50 p-0.5">
                        <button class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] text-foreground/70 hover:text-foreground hover:bg-white transition-all whitespace-nowrap leading-none" @click="openScheduleDialog(row)">
                          <FileText class="h-3.5 w-3.5 shrink-0" />编辑
                        </button>
                        <div class="w-px h-4 bg-border mx-0.5 shrink-0"></div>
                        <button class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] text-red-400 hover:text-red-500 hover:bg-white transition-all whitespace-nowrap leading-none" @click="confirmDeleteTarget = row; showDeleteScheduleDialog = true">
                          <Trash2 class="h-3.5 w-3.5 shrink-0" />删除
                        </button>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>

    <Dialog v-model:open="showCreateDialog">
      <DialogContent class="sm:max-w-[520px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle class="text-[15px] text-foreground">创建备份</DialogTitle>
        </div>
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">备份类型 <span class="text-red-500">*</span></label>
            <div class="inline-flex h-9 items-center rounded-lg bg-muted p-1 text-muted-foreground">
              <button
                v-for="opt in createLevelOptions"
                :key="opt.value"
                :class="[
                  'inline-flex items-center justify-center rounded-md px-3 py-1 text-sm font-medium transition-all',
                  createForm.backupLevel === opt.value
                    ? 'bg-white text-foreground shadow'
                    : 'hover:text-foreground'
                ]"
                @click="createForm.backupLevel = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
            <p v-if="createForm.backupLevel === 'system'" class="text-xs text-muted-foreground mt-1">
              备份当前所有配置（数据库列表、远程服务器、备份记录、定时计划）
            </p>
          </div>
          <div v-if="createForm.backupLevel !== 'system'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">目标数据库 <span class="text-red-500">*</span></label>
            <Select v-model="createForm.targetDb">
              <SelectTrigger class="border-border shadow-none">
                <template v-if="selectedCreateDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedCreateDb.name }}
                    <Badge
                      variant="outline"
                      :class="selectedCreateDb.type === 'redis' ? 'bg-amber-50 text-amber-600 border-amber-200' : 'bg-blue-50 text-blue-600 border-blue-200'"
                      class="text-[10px] px-1.5 py-0 shrink-0"
                    >
                      {{ selectedCreateDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </Badge>
                    <span class="text-muted-foreground text-xs shrink-0">{{ selectedCreateDb.host || '本地' }}</span>
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
                        :class="db.type === 'redis' ? 'bg-amber-50 text-amber-600 border-amber-200' : 'bg-blue-50 text-blue-600 border-blue-200'"
                        class="text-[10px] px-1.5 py-0"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </Badge>
                      <span class="text-muted-foreground text-xs">{{ db.host || '本地' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <!-- MySQL 备份：选择具体要备份的数据库名 -->
          <div v-if="createForm.backupLevel === 'mysql'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">选择要备份的数据库 <span class="text-red-500">*</span></label>
            <Select v-model="createForm.targetMysqlDbName">
              <SelectTrigger class="border-border shadow-none">
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
            <p v-if="loadingMysqlDatabases" class="text-xs text-muted-foreground flex items-center gap-1 mt-1">
              <Loader2 class="h-3 w-3 animate-spin" /> 正在加载数据库列表...
            </p>
            <p v-else-if="createForm.targetDb && mysqlDatabaseOptions.length === 0" class="text-xs text-amber-500 mt-1">
              该连接下没有可用的数据库
            </p>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">备份名称</label>
            <Input v-model="createForm.name" placeholder="留空自动生成" class="border-border shadow-none" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">描述信息</label>
            <Textarea v-model="createForm.description" placeholder="可选" :rows="2" class="border-border shadow-none" />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <Button variant="outline" @click="showCreateDialog = false">取消</Button>
          <Button variant="primary" @click="submitCreate" :disabled="creating">
            <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
            确认创建
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showScheduleDialog">
      <DialogContent class="sm:max-w-[520px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle class="text-[15px] text-foreground">{{ editScheduleData ? '编辑定时计划' : '新建定时备份' }}</DialogTitle>
        </div>
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">计划名称 <span class="text-red-500">*</span></label>
            <Input v-model="scheduleForm.name" placeholder="如：每日数据库备份" class="border-border shadow-none" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">备份类型 <span class="text-red-500">*</span></label>
            <div class="inline-flex h-9 items-center rounded-lg bg-muted p-1 text-muted-foreground">
              <button
                v-for="opt in createLevelOptions"
                :key="opt.value"
                :class="[
                  'inline-flex items-center justify-center rounded-md px-3 py-1 text-sm font-medium transition-all',
                  scheduleForm.backupLevel === opt.value
                    ? 'bg-white text-foreground shadow'
                    : 'hover:text-foreground'
                ]"
                @click="scheduleForm.backupLevel = opt.value"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
          <div v-if="scheduleForm.backupLevel !== 'system'" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">目标连接 <span class="text-red-500">*</span></label>
            <Select v-model="scheduleForm.targetDb" @update:model-value="onScheduleDbChange">
              <SelectTrigger class="border-border shadow-none">
                <template v-if="selectedScheduleDb">
                  <span class="flex items-center gap-1.5 truncate">
                    {{ selectedScheduleDb.name }}
                    <Badge
                      variant="outline"
                      :class="selectedScheduleDb.type === 'redis' ? 'bg-amber-50 text-amber-600 border-amber-200' : 'bg-blue-50 text-blue-600 border-blue-200'"
                      class="text-[10px] px-1.5 py-0 shrink-0"
                    >
                      {{ selectedScheduleDb.type === 'redis' ? 'Redis' : 'MySQL' }}
                    </Badge>
                    <span class="text-muted-foreground text-xs shrink-0">{{ selectedScheduleDb.host || '本地' }}</span>
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
                        :class="db.type === 'redis' ? 'bg-amber-50 text-amber-600 border-amber-200' : 'bg-blue-50 text-blue-600 border-blue-200'"
                        class="text-[10px] px-1.5 py-0"
                      >
                        {{ db.type === 'redis' ? 'Redis' : 'MySQL' }}
                      </Badge>
                      <span class="text-muted-foreground text-xs">{{ db.host || '本地' }}</span>
                    </div>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="scheduleForm.backupLevel === 'mysql' && scheduleForm.targetDb" class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">目标数据库 <span class="text-red-500">*</span></label>
            <Select v-model="scheduleForm.targetMysqlDbName">
              <SelectTrigger class="border-border shadow-none">
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
            <p v-if="loadingScheduleMysqlDatabases" class="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              <Loader2 class="h-3 w-3 animate-spin" /> 正在加载数据库列表...
            </p>
            <p v-else-if="scheduleForm.targetDb && scheduleMysqlDatabaseOptions.length === 0" class="text-xs text-amber-500 mt-1">
              该连接下没有可用的数据库
            </p>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">执行周期 <span class="text-red-500">*</span></label>
            <Select v-model="scheduleForm.cron">
              <SelectTrigger class="border-border shadow-none">
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
            <label class="text-sm font-medium text-foreground">保留份数</label>
            <Input v-model="scheduleForm.retainCount" type="number" :min="1" :max="30" class="w-32 border-border shadow-none" />
            <p class="text-xs text-muted-foreground">超过保留数量后自动删除最早的备份</p>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <Button variant="outline" @click="showScheduleDialog = false">取消</Button>
          <Button variant="primary" @click="submitSchedule" :disabled="scheduling">
            <Loader2 v-if="scheduling" class="h-4 w-4 animate-spin" />
            确认
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-foreground">删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-muted-foreground">
            确定要删除备份 "{{ confirmDeleteTarget?.name }}" 吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" @click="showDeleteDialog = false">取消</Button>
          <Button variant="destructive" @click="handleDelete(confirmDeleteTarget); showDeleteDialog = false">确定删除</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showBatchDeleteDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-foreground">批量删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-muted-foreground">
            确定要删除选中的 {{ selectedBackups.length }} 个备份吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" @click="showBatchDeleteDialog = false">取消</Button>
          <Button variant="destructive" @click="handleBatchDelete(); showBatchDeleteDialog = false">确定删除</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showDeleteScheduleDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-foreground">删除确认</DialogTitle>
          <DialogDescription class="text-[13px] text-muted-foreground">
            确定要删除计划 "{{ confirmDeleteTarget?.name }}" 吗？此操作不可撤销！
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" @click="showDeleteScheduleDialog = false">取消</Button>
          <Button variant="destructive" @click="deleteSchedule(confirmDeleteTarget); showDeleteScheduleDialog = false">确定删除</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showRestoreDialog">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px] text-foreground">恢复确认</DialogTitle>
          <DialogDescription class="text-[13px] text-muted-foreground">
            {{ restoreMessage }}
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" @click="showRestoreDialog = false">取消</Button>
          <Button variant="destructive" @click="doRestore" :disabled="restoring">
            <RefreshCw v-if="restoring" class="h-4 w-4 animate-spin" />
            确定恢复
          </Button>
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
import { toast } from 'vue-sonner'
import { sourceParam } from '@/lib/instance'
import { formatLogTime } from '@/lib/utils'
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
        // 在最前面增加「全部」选项
        mysqlDatabaseOptions.value = ['__ALL__', ...filtered]
        // 尝试自动选中：优先选择用户当前正在看的数据库（store.dbName）
        if (store.dbName && filtered.includes(store.dbName)) {
          createForm.value.targetMysqlDbName = store.dbName
        } else if (selectedDb.database && filtered.includes(selectedDb.database)) {
          createForm.value.targetMysqlDbName = selectedDb.database
        } else if (filtered.length > 0) {
          createForm.value.targetMysqlDbName = filtered[0]
        } else {
          createForm.value.targetMysqlDbName = '__ALL__'
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
        scheduleMysqlDatabaseOptions.value = ['__ALL__', ...filtered]
        if (store.dbName && filtered.includes(store.dbName)) {
          scheduleForm.value.targetMysqlDbName = store.dbName
        } else if (filtered.length > 0) {
          scheduleForm.value.targetMysqlDbName = filtered[0]
        } else {
          scheduleForm.value.targetMysqlDbName = '__ALL__'
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
  if (ids.length === 0) { toast.warning('请选择要删除的备份'); return }
  fetch('/api/backups', {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids })
  }).then(r => r.json()).then(result => {
    if (result.code === 0) {
      toast.success(result.msg || `成功删除 ${ids.length} 个备份`)
    } else {
      toast.error(result.msg || '删除失败')
    }
    selectedBackups.value = []
    loadBackups()
  }).catch(() => {
    toast.error('删除失败')
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
      toast.warning('请选择目标数据库')
      creating.value = false
      return
    }
    selectedDb = findDbByUid(targetUid)
    if (!selectedDb) {
      toast.warning('请选择目标数据库')
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
      toast.warning('请选择要备份的MySQL数据库名')
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
          toast.success('Redis备份创建成功')
          showCreateDialog.value = false
          loadBackups()
        } else {
          toast.error(data.msg || '创建失败')
        }
      })
      .catch(() => { toast.error('创建失败') })
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
        toast.success(label + '备份创建成功')
        showCreateDialog.value = false
        loadBackups()
      } else {
        toast.error(data.msg || '创建失败')
      }
    })
    .catch(() => {
      toast.error('创建失败')
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
        toast.success('删除成功')
        loadBackups()
      } else {
        toast.error(data.msg || '删除失败')
      }
    })
    .catch(() => {
      toast.error('删除失败')
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
        toast.success(data.msg || '恢复成功')
        showRestoreDialog.value = false
        loadBackups()
      } else {
        toast.error(data.msg || '恢复失败')
      }
    })
    .catch(err => {
      toast.error('恢复失败: ' + err.message)
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
      toast.warning('请选择目标连接')
      scheduling.value = false
      return
    }
    selectedDb = findDbByUid(targetUid)
    if (!selectedDb) {
      toast.warning('请选择目标连接')
      scheduling.value = false
      return
    }
    if (selectedDb.type !== 'redis' && !scheduleForm.value.targetMysqlDbName) {
      toast.warning('请选择目标数据库')
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
        toast.success(editScheduleData.value ? '更新成功' : '创建成功')
        showScheduleDialog.value = false
        editScheduleData.value = null
        loadSchedules()
      } else {
        toast.error(data.msg || '操作失败')
      }
    })
    .catch(() => { toast.error('操作失败') })
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
        toast.success(newEnabled ? '已启用定时备份' : '已禁用定时备份')
        loadSchedules()
      } else {
        row.enabled = !newEnabled
        toast.error(data.msg || '操作失败')
      }
    })
    .catch(() => {
      row.enabled = !newEnabled
      toast.error('操作失败')
    })
}

const deleteSchedule = (row) => {
  if (!row) return
  fetch(`/api/backups/scheduled/${row.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success('删除成功')
        loadSchedules()
      } else {
        toast.error(data.msg || '删除失败')
      }
    })
    .catch(() => {
      toast.error('删除失败')
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
