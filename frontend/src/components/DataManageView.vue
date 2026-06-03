<template>
  <div class="page-padding overflow-y-auto h-full">
    <!-- ═══════════════════════════════════════════════════════════
         No Instance Selected — Instance List
         ═══════════════════════════════════════════════════════════ -->
    <div v-if="!connectionId" class="fade-up">
      <div class="content-card">
        <div class="content-header">
          <div class="flex items-start justify-between">
            <div>
              <h2 class="text-[15px] font-semibold" style="color: var(--text-primary)">数据管理</h2>
              <p class="text-[13px] mt-0.5" style="color: var(--text-tertiary)">选择一个实例开始浏览和管理数据</p>
            </div>
            <button class="btn-primary" @click="addInstance">
              <Plus class="h-3.5 w-3.5" />
              添加实例
            </button>
          </div>
        </div>

        <div class="content-body">
          <div
            v-for="inst in deduplicatedInstances"
            :key="(inst.isRemote ? 'r' : 'l') + inst.id"
            class="instance-card group"
            :class="{ 'instance-card--active': connectionId === instanceUid(inst) }"
            @click="selectInstance(inst)"
          >
            <div class="flex items-center gap-3">
              <div class="instance-icon" :style="{ background: getInstanceColor(inst) }">
                <Database class="h-[18px] w-[18px] text-white" />
              </div>
              <div class="flex-1 flex flex-col gap-0.5 min-w-0">
                <div class="flex items-center gap-1.5">
                  <span class="text-[13px] font-medium truncate" style="color: var(--text-primary)">{{ inst.name }}</span>
                  <span v-if="inst.isRemote" class="badge-status badge-status-warning">远程</span>
                </div>
                <div class="flex items-center gap-1.5 text-[11px]" style="color: var(--text-tertiary)">
                  <StatusDot :status="connectionId === instanceUid(inst) ? 'selected' : (onlineStatus[instanceUid(inst)] !== false ? 'online' : 'offline')" size="xs" />
                  {{ inst.type === 'mysql' ? 'MySQL' : inst.type }}
                  <span class="ml-auto font-mono-data">{{ inst.host || 'localhost' }}:{{ inst.port || 3306 }}</span>
                </div>
              </div>
            </div>
            <div class="instance-actions">
              <span class="text-[11px]" style="color: var(--text-tertiary)">{{ inst.version || '—' }}</span>
              <div class="flex gap-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100" @click.stop>
                <button class="btn-ghost" @click="editInstance(inst)">编辑</button>
                <button class="btn-ghost-danger" @click="confirmDeleteInstance(inst)">删除</button>
              </div>
            </div>
          </div>
          <div v-if="deduplicatedInstances.length === 0" class="empty-state py-10">
            <Inbox class="h-12 w-12 empty-state-icon" />
            <span class="empty-state-text">暂无实例</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════
         Instance Selected — Main Content
         ═══════════════════════════════════════════════════════════ -->
    <div v-else class="flex flex-col gap-4">
      <!-- Header Card with Breadcrumb -->
      <div class="content-card fade-up">
        <div class="content-header">
          <div class="flex items-start justify-between">
            <div>
              <!-- Breadcrumb Navigation -->
              <nav class="breadcrumb-nav">
                <span
                  class="breadcrumb-item breadcrumb-item--active"
                  @click="backToDatabases"
                >{{ currentInst?.name }}</span>

                <template v-if="isRedis && selectedRedisKey">
                  <ChevronRight class="breadcrumb-sep" />
                  <span class="breadcrumb-item breadcrumb-item--current">{{ selectedRedisKey }}</span>
                </template>

                <template v-if="!isRedis && selectedDatabase">
                  <ChevronRight class="breadcrumb-sep" />
                  <span
                    class="breadcrumb-item breadcrumb-item--active"
                    @click="backToTables"
                  >{{ selectedDatabase }}</span>
                </template>

                <template v-if="!isRedis && selectedTable">
                  <ChevronRight class="breadcrumb-sep" />
                  <span class="breadcrumb-item breadcrumb-item--current">{{ selectedTable }}</span>
                </template>
              </nav>
              <p class="text-[13px] mt-0.5" style="color: var(--text-tertiary)">
                {{ isRedis ? 'Redis' : 'MySQL' }} · {{ currentInst?.host }}:{{ currentInst?.port }}
              </p>
            </div>
            <div class="flex items-center gap-2 shrink-0 flex-wrap">
              <button v-if="!isRedis && !selectedDatabase" class="btn-primary" @click="openCreateDbDialog">
                <Plus class="h-3.5 w-3.5" />
                创建数据库
              </button>
              <button class="btn-secondary" @click="refreshData">
                <RefreshCw class="h-3.5 w-3.5" />
                刷新
              </button>
              <button v-if="selectedTable && !isRedis" class="btn-primary" @click="showInsertDialog = true">
                <Plus class="h-3.5 w-3.5" />
                插入
              </button>
              <button v-if="selectedTable && !isRedis" class="btn-secondary" @click="exportData(false)">
                <Download class="h-3.5 w-3.5" />
                导出
              </button>
              <button v-if="selectedTable && !isRedis" class="btn-secondary" @click="exportData(true)">
                <Download class="h-3.5 w-3.5" />
                全量导出
              </button>
            </div>
          </div>
        </div>

        <!-- ── Redis: Key List View ── -->
        <div v-if="isRedis && !selectedRedisKey" class="content-body fade-up">
          <!-- Redis Stats -->
          <div class="redis-stats">
            <div class="stat-card">
              <div class="stat-icon" style="background: var(--accent-soft); color: var(--accent)">
                <Database class="h-4 w-4" />
              </div>
              <div>
                <div class="stat-value">{{ redisInfo.redis_version || '—' }}</div>
                <div class="stat-label">版本</div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-icon" style="background: var(--success-soft); color: var(--success)">
                <RefreshCw class="h-4 w-4" />
              </div>
              <div>
                <div class="stat-value">{{ redisInfo.uptime_in_days ? redisInfo.uptime_in_days + ' 天' : '—' }}</div>
                <div class="stat-label">运行时间</div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-icon" style="background: var(--warning-soft); color: var(--warning)">
                <Database class="h-4 w-4" />
              </div>
              <div>
                <div class="stat-value">{{ redisInfo.used_memory_human || '—' }}</div>
                <div class="stat-label">已用内存</div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-icon" style="background: var(--accent-soft); color: var(--accent)">
                <Database class="h-4 w-4" />
              </div>
              <div>
                <div class="stat-value">{{ redisInfo._totalKeys || '—' }}</div>
                <div class="stat-label">Key数量</div>
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-icon" style="background: var(--success-soft); color: var(--success)">
                <Database class="h-4 w-4" />
              </div>
              <div>
                <div class="stat-value">
                  {{ redisInfo.keyspace_hits && redisInfo.keyspace_misses ?
                    (parseInt(redisInfo.keyspace_hits) * 100 / (parseInt(redisInfo.keyspace_hits) + parseInt(redisInfo.keyspace_misses))).toFixed(1) + '%'
                    : '—' }}
                </div>
                <div class="stat-label">命中率</div>
              </div>
            </div>
          </div>

          <!-- Redis Search Bar -->
          <div class="redis-search-bar">
            <div class="redis-search-input">
              <span class="redis-search-prefix font-mono-data">SCAN</span>
              <Input v-model="redisPattern" placeholder="Key 匹配模式 (如 user:* )" class="redis-search-field" @keyup.enter="searchRedisKeys" />
            </div>
            <button class="btn-primary" @click="searchRedisKeys" :disabled="loadingRedisKeys">
              <Search class="h-4 w-4" />
              搜索
            </button>
          </div>

          <!-- Redis Key List -->
          <div class="redis-key-list">
            <div v-if="loadingRedisKeys" class="redis-key-loading">
              <Loader2 class="h-6 w-6 animate-spin" style="color: var(--accent)" />
            </div>
            <div v-if="redisKeys.length === 0 && !loadingRedisKeys" class="empty-state py-10">
              <Inbox class="h-10 w-10 empty-state-icon" />
              <span class="empty-state-text">暂无Key</span>
            </div>
            <div
              v-for="rk in redisKeys"
              :key="rk.key"
              class="redis-key-item"
              @click="selectRedisKey(rk.key)"
            >
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <span
                  class="badge-status"
                  :class="{
                    'badge-status-info': rk.type === 'string',
                    'badge-status-success': rk.type === 'list' || rk.type === 'set',
                    'badge-status-warning': rk.type === 'hash',
                    'badge-status-error': rk.type === 'zset',
                    'badge-status-neutral': !['string','list','set','hash','zset'].includes(rk.type)
                  }"
                >{{ rk.type }}</span>
                <span class="font-mono-data text-[13px] truncate" style="color: var(--text-primary)">{{ rk.key }}</span>
              </div>
              <div class="flex items-center gap-3 shrink-0">
                <span class="text-[11px]" style="color: var(--text-tertiary)">{{ formatKeySize(rk.type, rk.size) }}</span>
                <span v-if="rk.ttl > 0" class="badge-status badge-status-warning">TTL: {{ rk.ttl }}s</span>
                <span v-else-if="rk.ttl === -1" class="badge-status badge-status-success">永不过期</span>
              </div>
            </div>
            <div v-if="redisCursor !== '0'" class="redis-key-more">
              <button class="btn-ghost" @click="loadMoreRedisKeys" :disabled="loadingRedisKeys">加载更多</button>
            </div>
          </div>

          <!-- Redis Command Console -->
          <div class="redis-console">
            <div class="redis-console-header">
              <span class="text-[13px] font-medium" style="color: var(--text-secondary)">Redis 命令</span>
            </div>
            <div class="flex gap-2">
              <Input v-model="redisCmd" placeholder="输入 Redis 命令，如 KEYS *" class="flex-1 h-8 text-xs" style="border-color: var(--border)" @keyup.enter="executeRedisCmd" />
              <button class="btn-primary" @click="executeRedisCmd">执行</button>
            </div>
            <div v-if="redisCmdResult" class="redis-console-result code-editor">
              <pre class="m-0 font-mono-data text-xs whitespace-pre-wrap break-all" style="color: var(--text-secondary)">{{ redisCmdResult }}</pre>
            </div>
          </div>
        </div>

        <!-- ── Redis: Key Detail View ── -->
        <div v-else-if="isRedis && selectedRedisKey" class="content-body fade-up">
          <div class="flex justify-between items-center mb-4">
            <div class="flex items-center gap-2">
              <span
                class="badge-status"
                :class="{
                  'badge-status-info': selectedRedisKeyData?.type === 'string',
                  'badge-status-success': selectedRedisKeyData?.type === 'list' || selectedRedisKeyData?.type === 'set',
                  'badge-status-warning': selectedRedisKeyData?.type === 'hash',
                  'badge-status-error': selectedRedisKeyData?.type === 'zset',
                  'badge-status-neutral': !['string','list','set','hash','zset'].includes(selectedRedisKeyData?.type)
                }"
              >{{ selectedRedisKeyData?.type }}</span>
              <span class="font-mono-data text-[15px] font-semibold" style="color: var(--text-primary)">{{ selectedRedisKey }}</span>
            </div>
            <div class="text-xs" style="color: var(--text-tertiary)">
              <span v-if="selectedRedisKeyData?.ttl > 0" class="badge-status badge-status-warning">TTL: {{ selectedRedisKeyData.ttl }}s</span>
              <span v-else-if="selectedRedisKeyData?.ttl === -1" class="badge-status badge-status-success">永不过期</span>
            </div>
          </div>
          <div class="redis-value-panel">
            <div v-if="selectedRedisKeyData?.type === 'string'">
              <Textarea :model-value="selectedRedisKeyData?.value" readonly rows="6" class="font-mono text-sm code-editor" />
            </div>
            <div v-else-if="selectedRedisKeyData?.type === 'list'">
              <div v-for="(item, idx) in selectedRedisKeyData?.value" :key="idx" class="redis-value-row">
                <span class="redis-value-idx">{{ idx }}</span>
                <span style="color: var(--text-primary)">{{ item }}</span>
              </div>
            </div>
            <div v-else-if="selectedRedisKeyData?.type === 'set'">
              <div v-for="(item, idx) in selectedRedisKeyData?.value" :key="idx" class="redis-value-row">
                <span style="color: var(--text-primary)">{{ item }}</span>
              </div>
            </div>
            <div v-else-if="selectedRedisKeyData?.type === 'hash'">
              <Table>
                <TableHeader>
                  <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableHead class="min-w-[200px] text-[12px] font-normal" style="color: var(--text-tertiary)">Field</TableHead>
                    <TableHead class="min-w-[300px] text-[12px] font-normal" style="color: var(--text-tertiary)">Value</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="entry in hashEntries" :key="entry.field" class="hover:bg-muted" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableCell class="font-mono-data text-sm truncate max-w-[200px]" style="color: var(--text-primary)">{{ entry.field }}</TableCell>
                    <TableCell class="font-mono-data text-sm truncate max-w-[300px]" style="color: var(--text-primary)">{{ entry.value }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
            <div v-else-if="selectedRedisKeyData?.type === 'zset'">
              <Table>
                <TableHeader>
                  <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableHead class="min-w-[200px] text-[12px] font-normal" style="color: var(--text-tertiary)">Member</TableHead>
                    <TableHead class="w-[120px] text-[12px] font-normal" style="color: var(--text-tertiary)">Score</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="(item, idx) in selectedRedisKeyData?.value" :key="idx" class="hover:bg-muted" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableCell class="font-mono-data text-sm truncate max-w-[200px]" style="color: var(--text-primary)">{{ item.member }}</TableCell>
                    <TableCell style="color: var(--text-primary)">{{ item.score }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
            <div v-else>
              <pre class="m-0 font-mono-data text-[13px] whitespace-pre-wrap break-all" style="color: var(--text-primary)">{{ JSON.stringify(selectedRedisKeyData?.value, null, 2) }}</pre>
            </div>
          </div>
        </div>

        <!-- ── MySQL: Database Grid ── -->
        <div v-if="!isRedis && !selectedDatabase" class="content-body fade-up">
          <div class="section-label" style="color: var(--text-secondary)">选择数据库</div>
          <div v-if="loadingDatabases" class="flex items-center gap-2 py-6 text-[13px]" style="color: var(--text-tertiary)">
            <Loader2 class="h-4 w-4 animate-spin" />
            正在加载数据库列表...
          </div>
          <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] grid-gap">
            <div
              v-for="db in databases"
              :key="db"
              class="grid-card group"
              @click="selectDatabase(db)"
            >
              <div class="grid-card-icon" style="background: var(--accent-soft); color: var(--accent)">
                <Database class="h-4 w-4" />
              </div>
              <div class="grid-card-info">
                <span class="grid-card-name">{{ db }}</span>
              </div>
              <button
                class="grid-card-delete"
                title="删除数据库"
                @click.stop="openDeleteDbDialog(db)"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <div v-if="!loadingDatabases && databases.length === 0" class="empty-state py-10">
            <Inbox class="h-10 w-10 empty-state-icon" />
            <span class="empty-state-text">暂无数据库</span>
          </div>
        </div>

        <!-- ── MySQL: Table Grid ── -->
        <div v-else-if="!isRedis && !selectedTable" class="content-body fade-up">
          <div class="flex items-center justify-between mb-3">
            <div class="section-label" style="color: var(--text-secondary)">选择数据表</div>
            <div class="flex items-center gap-2">
              <button class="btn-secondary" @click="openCreateTableDialog">
                <Plus class="h-3 w-3" />
                新建表
              </button>
              <button class="btn-secondary" @click="loadTables">
                <RefreshCw class="h-3 w-3" />
              </button>
            </div>
          </div>
          <div v-if="loadingTables" class="flex items-center gap-2 py-6 text-[13px]" style="color: var(--text-tertiary)">
            <Loader2 class="h-4 w-4 animate-spin" />
            正在加载表列表...
          </div>
          <div v-else class="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] grid-gap">
            <div
              v-for="tbl in tables"
              :key="tbl"
              class="grid-card group"
              style="--card-accent: var(--success); --card-accent-soft: var(--success-soft)"
              @click="selectTable(tbl)"
            >
              <div class="grid-card-icon" style="background: var(--success-soft); color: var(--success)">
                <Table2 class="h-4 w-4" />
              </div>
              <div class="grid-card-info">
                <span class="grid-card-name">{{ tbl }}</span>
              </div>
              <button
                class="grid-card-delete"
                title="删除表"
                @click.stop="confirmDeleteTable(tbl)"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
          <div v-if="!loadingTables && tables.length === 0 && tableLoadError" class="flex items-center gap-2 py-6 text-[13px]" style="color: var(--danger)">
            <AlertTriangle class="h-4 w-4" />
            <span>加载失败</span>
            <button class="btn-ghost" @click="loadTables()">重试</button>
          </div>
          <div v-else-if="!loadingTables && tables.length === 0" class="empty-state py-10">
            <Inbox class="h-10 w-10 empty-state-icon" />
            <span class="empty-state-text">暂无数据表</span>
          </div>
        </div>
      </div>

      <!-- ── MySQL: Data Table Card ── -->
      <div v-if="!isRedis && selectedTable" class="content-card fade-up">
        <!-- Pagination Bar -->
        <div class="data-table-pagination">
          <span class="text-[13px]" style="color: var(--text-tertiary)">共 {{ totalRows }} 行数据</span>
          <div class="flex items-center gap-2">
            <button
              class="pagination-btn"
              :disabled="page <= 1"
              @click="page--; onPageChange()"
            >
              <ChevronLeft class="h-3.5 w-3.5" />
            </button>
            <span class="text-xs font-mono-data min-w-[60px] text-center" style="color: var(--text-tertiary)">{{ page }} / {{ totalPages }}</span>
            <button
              class="pagination-btn"
              :disabled="page >= totalPages"
              @click="page++; onPageChange()"
            >
              <ChevronRight class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        <!-- Table Content -->
        <div class="relative overflow-hidden" :style="{ height: tableHeight }">
          <div v-if="loadingData" class="data-table-loading">
            <Loader2 class="h-6 w-6 animate-spin" style="color: var(--accent)" />
          </div>
          <Table class="h-full">
            <TableHeader>
              <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                <TableHead v-for="col in columns" :key="col.name" class="min-w-[120px] text-[12px] font-normal" style="color: var(--text-tertiary)">
                  {{ col.name }}
                </TableHead>
                <TableHead class="data-table-action-col text-[12px] font-normal" style="color: var(--text-tertiary)">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow
                v-for="(row, idx) in tableData"
                :key="idx"
                class="data-table-row"
              >
                <TableCell v-for="col in columns" :key="col.name" class="max-w-[300px]">
                  <span
                    :class="row[col.name] === null ? 'data-table-null' : 'font-mono-data text-[13px]'"
                    :style="row[col.name] !== null ? 'color: var(--text-primary)' : ''"
                  >
                    {{ row[col.name] !== null ? row[col.name] : 'NULL' }}
                  </span>
                </TableCell>
                <TableCell class="data-table-action-col">
                  <div class="data-table-row-actions">
                    <button class="btn-ghost" @click="editRow(row)" :disabled="!columns.some(c => c.key === 'PRI')">编辑</button>
                    <button class="btn-ghost-danger" @click="confirmDeleteRow(row)" :disabled="!columns.some(c => c.key === 'PRI')">删除</button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>

        <!-- Bottom Panel: Structure & SQL -->
        <div class="data-table-bottom-panel">
          <Tabs v-model="activeTab">
            <TabsList style="background: var(--muted)">
              <TabsTrigger value="structure" class="text-[12px]" :class="activeTab === 'structure' ? 'tab-active' : 'tab-inactive'">表结构</TabsTrigger>
              <TabsTrigger value="sql" class="text-[12px]" :class="activeTab === 'sql' ? 'tab-active' : 'tab-inactive'">SQL</TabsTrigger>
            </TabsList>
            <TabsContent value="structure">
              <Table>
                <TableHeader>
                  <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableHead class="min-w-[120px] text-[12px] font-normal" style="color: var(--text-tertiary)">字段名</TableHead>
                    <TableHead class="min-w-[140px] text-[12px] font-normal" style="color: var(--text-tertiary)">类型</TableHead>
                    <TableHead class="w-[80px] text-[12px] font-normal" style="color: var(--text-tertiary)">键</TableHead>
                    <TableHead class="min-w-[120px] text-[12px] font-normal" style="color: var(--text-tertiary)">默认值</TableHead>
                    <TableHead class="min-w-[120px] text-[12px] font-normal" style="color: var(--text-tertiary)">额外</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="col in columns" :key="col.name" class="hover:bg-muted" style="border-bottom: 1px solid var(--border-subtle)">
                    <TableCell class="font-mono-data" style="color: var(--text-primary)">{{ col.name }}</TableCell>
                    <TableCell style="color: var(--text-primary)">{{ col.type }}</TableCell>
                    <TableCell style="color: var(--text-primary)">{{ col.key }}</TableCell>
                    <TableCell style="color: var(--text-primary)">{{ col.default }}</TableCell>
                    <TableCell style="color: var(--text-primary)">{{ col.extra }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TabsContent>
            <TabsContent value="sql">
              <div class="sql-console">
                <Textarea v-model="sqlQuery" :rows="5" placeholder="输入SQL查询语句..." class="font-mono text-sm mb-3 code-editor" style="border-color: var(--border)" />
                <div class="flex justify-end mb-3">
                  <button class="btn-primary" @click="executeSql" :disabled="sqlResultLoading">
                    {{ sqlResultLoading ? '执行中...' : '执行' }}
                  </button>
                </div>
                <div v-if="sqlResult" class="mt-2">
                  <div v-if="sqlResult.type === 'rows'" class="sql-result-table">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead v-for="col in sqlResult.columns" :key="col" class="text-xs">{{ col }}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRow v-for="(row, idx) in sqlResult.data" :key="idx">
                          <TableCell v-for="col in sqlResult.columns" :key="col" class="text-xs font-mono-data">{{ row[col] ?? 'NULL' }}</TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                    <div class="sql-result-footer" style="color: var(--text-tertiary)">共 {{ sqlResult.data.length }} 行</div>
                  </div>
                  <div v-else-if="sqlResult.type === 'message'" class="sql-result-msg sql-result-msg--success">{{ sqlResult.data }}</div>
                  <div v-else-if="sqlResult.type === 'error'" class="sql-result-msg sql-result-msg--error">{{ sqlResult.data }}</div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>

    <!-- ═══════════════════════════════════════════════════════════
         Dialogs (unchanged logic, updated styling)
         ═══════════════════════════════════════════════════════════ -->
    <Dialog v-model:open="showInsertDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="max-w-[600px] max-h-[80vh] overflow-y-auto">
          <DialogTitle>插入数据</DialogTitle>
          <DialogDescription />
          <div class="grid gap-4 py-4">
            <div v-for="col in columns" :key="col.name" class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">{{ col.name }}</label>
              <Input v-if="isNumericType(col.type)" type="number" v-model="insertForm[col.name]" :placeholder="`请输入 ${col.name}`" class="w-full" />
              <Switch v-else-if="isBooleanType(col.type)" v-model:checked="insertForm[col.name]" />
              <Input v-else-if="isDateType(col.type)" :type="getDateInputType(col.type)" v-model="insertForm[col.name]" :placeholder="`请选择 ${col.name}`" class="w-full" />
              <Textarea v-else-if="isTextType(col.type)" v-model="insertForm[col.name]" :rows="3" :placeholder="`请输入 ${col.name}`" />
              <Input v-else v-model="insertForm[col.name]" :placeholder="`请输入 ${col.name}`" />
            </div>
          </div>
          <div class="flex justify-end gap-2 flex-wrap">
            <button class="btn-secondary" @click="showInsertDialog = false">取消</button>
            <button class="btn-primary" @click="insertData">确认插入</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="showEditDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="max-w-[600px] max-h-[80vh] overflow-y-auto">
          <DialogTitle>编辑数据</DialogTitle>
          <DialogDescription />
          <div class="grid gap-4 py-4">
            <div v-for="col in columns" :key="col.name" class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">{{ col.name }}</label>
              <Input v-if="isNumericType(col.type)" type="number" v-model="editForm[col.name]" :placeholder="`请输入 ${col.name}`" class="w-full" />
              <Switch v-else-if="isBooleanType(col.type)" v-model:checked="editForm[col.name]" />
              <Input v-else-if="isDateType(col.type)" :type="getDateInputType(col.type)" v-model="editForm[col.name]" :placeholder="`请选择 ${col.name}`" class="w-full" />
              <Textarea v-else-if="isTextType(col.type)" v-model="editForm[col.name]" :rows="3" :placeholder="`请输入 ${col.name}`" />
              <Input v-else v-model="editForm[col.name]" :placeholder="`请输入 ${col.name}`" />
            </div>
          </div>
          <div class="flex justify-end gap-2 flex-wrap">
            <button class="btn-secondary" @click="showEditDialog = false">取消</button>
            <button class="btn-primary" @click="updateData">保存修改</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="confirmState.open">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="max-w-[400px]">
          <DialogTitle>{{ confirmState.title }}</DialogTitle>
          <DialogDescription>{{ confirmState.description }}</DialogDescription>
          <div class="flex justify-end gap-2 mt-4 flex-wrap">
            <button class="btn-secondary" @click="confirmState.open = false">取消</button>
            <button
              :class="confirmState.variant === 'destructive' ? 'btn-danger' : 'btn-primary'"
              @click="handleConfirm"
            >确认</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="showCreateDbDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="max-w-[440px]">
          <DialogTitle>创建数据库</DialogTitle>
          <DialogDescription />
          <div class="grid gap-4 py-4">
            <div class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">数据库名称 <span style="color: var(--danger)">*</span></label>
              <Input v-model="createDbForm.name" placeholder="请输入数据库名称" class="w-full" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">用户密码 <span class="text-xs font-normal" style="color: var(--text-tertiary)">(可选，将创建同名用户并授权)</span></label>
              <Input v-model="createDbForm.password" type="password" placeholder="请输入用户密码" class="w-full" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">确认密码 <span class="text-xs font-normal" style="color: var(--text-tertiary)">(如填写密码则需再次输入)</span></label>
              <Input v-model="createDbForm.confirmPassword" type="password" placeholder="请再次输入密码" class="w-full" />
            </div>
            <div class="grid gap-1.5">
              <label class="text-sm font-medium leading-none" style="color: var(--text-primary)">字符集</label>
              <Select v-model="createDbForm.charset">
                <SelectTrigger class="w-full">
                  <SelectValue placeholder="选择字符集" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="utf8mb4">utf8mb4</SelectItem>
                  <SelectItem value="utf8mb3">utf8mb3</SelectItem>
                  <SelectItem value="utf8">utf8</SelectItem>
                  <SelectItem value="latin1">latin1</SelectItem>
                  <SelectItem value="ascii">ascii</SelectItem>
                  <SelectItem value="gbk">gbk</SelectItem>
                  <SelectItem value="gb2312">gb2312</SelectItem>
                  <SelectItem value="big5">big5</SelectItem>
                  <SelectItem value="binary">binary</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div v-if="createDbError" class="text-[13px] rounded-lg px-3 py-2" style="color: var(--danger); background: var(--danger-soft)">{{ createDbError }}</div>
          <div class="flex justify-end gap-2 flex-wrap">
            <button class="btn-secondary" @click="showCreateDbDialog = false">取消</button>
            <button class="btn-primary" :disabled="!createDbForm.name.trim() || (createDbForm.password && createDbForm.password !== createDbForm.confirmPassword)" @click="createDatabase">创建</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="showCreateTableDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
          <div class="flex flex-col gap-y-1.5">
            <DialogTitle>创建数据表</DialogTitle>
            <DialogDescription>设置表名和字段结构</DialogDescription>
          </div>
          <div class="flex flex-col gap-4 py-2">
            <div class="flex flex-col gap-1.5">
              <label class="text-sm font-medium" style="color: var(--text-primary)">表名 <span style="color: var(--danger)">*</span></label>
              <Input v-model="createTableForm.name" placeholder="例如: users" style="border-color: var(--border); box-shadow: none" />
            </div>
            <div class="flex flex-col gap-1.5">
              <div class="flex items-center justify-between">
                <label class="text-sm font-medium" style="color: var(--text-primary)">字段列表</label>
                <button class="btn-secondary" @click="addColumn">
                  <Plus class="h-3 w-3" />
                  添加字段
                </button>
              </div>
              <div v-if="createTableForm.columns.length === 0" class="text-[13px] py-4 text-center border-2 border-dashed rounded-lg" style="color: var(--text-tertiary); border-color: var(--border)">
                请添加至少一个字段
              </div>
              <div v-for="(col, idx) in createTableForm.columns" :key="idx" class="flex items-center gap-2 p-3 rounded-lg" style="border: 1px solid var(--border); background: var(--muted)">
                <div class="flex flex-col gap-0.5 flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <Input v-model="col.name" placeholder="字段名" class="flex-1 h-[32px] text-[13px]" style="border-color: var(--border); box-shadow: none" />
                    <select v-model="col.type" class="h-[32px] px-2 rounded-md text-[13px] outline-none" style="border: 1px solid var(--border); background: var(--surface); color: var(--text-primary)">
                      <option v-for="t in columnTypes" :key="t" :value="t">{{ t }}</option>
                    </select>
                    <Input v-model="col.length" placeholder="长度" class="w-[70px] h-[32px] text-[13px]" style="border-color: var(--border); box-shadow: none" />
                  </div>
                  <div class="flex items-center gap-4 mt-1">
                    <label class="flex items-center gap-1.5 text-[12px] cursor-pointer select-none">
                      <input type="checkbox" v-model="col.notNull" style="accent-color: var(--accent)" />
                      <span style="color: var(--text-tertiary)">NOT NULL</span>
                    </label>
                    <label class="flex items-center gap-1.5 text-[12px] cursor-pointer select-none">
                      <input type="checkbox" v-model="col.autoIncrement" style="accent-color: var(--accent)" />
                      <span style="color: var(--text-tertiary)">AUTO_INCREMENT</span>
                    </label>
                    <label class="flex items-center gap-1.5 text-[12px] cursor-pointer select-none">
                      <span style="color: var(--text-tertiary)">默认值:</span>
                      <Input v-model="col.defaultValue" placeholder="NULL" class="w-[100px] h-[24px] text-[12px]" style="border-color: var(--border); box-shadow: none" />
                    </label>
                    <select v-model="col.keyType" class="h-[24px] px-1.5 rounded text-[12px] outline-none" style="border: 1px solid var(--border); background: var(--surface); color: var(--text-primary)">
                      <option value="">无</option>
                      <option value="PRI">主键</option>
                      <option value="UNI">唯一键</option>
                      <option value="MUL">索引</option>
                    </select>
                  </div>
                </div>
                <button class="w-7 h-7 rounded-md flex items-center justify-center shrink-0 transition-colors" style="color: var(--danger)" @click="removeColumn(idx)">
                  <X class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>
          <div v-if="createTableError" class="text-[13px] rounded-lg px-3 py-2" style="color: var(--danger); background: var(--danger-soft)">{{ createTableError }}</div>
          <div class="flex justify-end gap-2 mt-1 flex-wrap">
            <button class="btn-secondary" @click="showCreateTableDialog = false">取消</button>
            <button class="btn-primary" :disabled="!createTableForm.name.trim() || createTableForm.columns.length === 0" @click="createTable">创建表</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="showDeleteTableDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="sm:max-w-[400px]">
          <div class="flex flex-col gap-y-1.5">
            <DialogTitle>删除数据表</DialogTitle>
            <DialogDescription>确定要删除表 <strong style="color: var(--danger)">{{ deleteTableName }}</strong> 吗？此操作不可撤销，所有数据将永久丢失。</DialogDescription>
          </div>
          <div class="flex justify-end gap-2 flex-wrap">
            <button class="btn-secondary" @click="showDeleteTableDialog = false">取消</button>
            <button class="btn-danger" @click="deleteTable">确认删除</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <Dialog v-model:open="showDeleteDbDialog">
      <DialogPortal>
        <DialogOverlay />
        <DialogContent class="sm:max-w-[440px]">
          <div class="flex flex-col gap-y-1.5">
            <DialogTitle>删除数据库</DialogTitle>
            <DialogDescription>
              此操作将永久删除数据库 <strong style="color: var(--danger)">{{ deleteDbName }}</strong> 及其所有数据，且不可恢复。
            </DialogDescription>
          </div>
          <div class="flex flex-col gap-1.5 mt-3">
            <label class="text-sm" style="color: var(--text-primary)">请输入数据库名称 <strong style="color: var(--danger)">{{ deleteDbName }}</strong> 以确认删除</label>
            <Input v-model="deleteDbConfirmInput" :placeholder="deleteDbName" style="border-color: var(--border); box-shadow: none" />
          </div>
          <div class="flex justify-end gap-2 mt-3 flex-wrap">
            <button class="btn-secondary" @click="showDeleteDbDialog = false">取消</button>
            <button class="btn-danger" :disabled="deleteDbConfirmInput !== deleteDbName" @click="doDeleteDatabase">确认删除</button>
          </div>
        </DialogContent>
      </DialogPortal>
    </Dialog>

    <InstanceDialog
      v-model="showInstanceDialog"
      :type="instanceDialogType"
      :data="instanceEditData"
      :existing-instances="instances"
      @success="loadInstances"
      @navigate-to-instance="navigateToInstance"
    />
  </div>
</template>

<script setup>
defineOptions({ name: 'DataManageView' })
import { ref, computed, onMounted, onActivated, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useHealthStore } from '../stores/health'
import { toast } from 'vue-sonner'
import { useAppContext } from '../stores/context'
import { sourceParam, sourceValue, instanceUid } from '@/lib/instance'
import { databaseApi } from '../api/database'
import InstanceDialog from './InstanceDialog.vue'
import StatusDot from './StatusDot.vue'

const completeProgress = inject('completeProgress')

import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Input } from '@/components/ui/Input.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Switch } from '@/components/ui/Switch.vue'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'

import {
  Database, ChevronRight, ChevronLeft, RefreshCw, Plus, Download, Search,
  Table2, Loader2, AlertTriangle, Inbox, X, Trash2
} from 'lucide-vue-next'

const API_BASE = '/api/mysql'
const REDIS_API = '/api/redis'

const store = useAppContext()
const { connectionId, isRedis, serverId } = storeToRefs(store)

const props = defineProps({
  navRequest: { type: Object, default: null }
})

const emit = defineEmits(['navAccepted'])

const instances = ref([])
const previousInstanceIds = ref(new Set())
const selectedDatabase = ref(null)
const selectedTable = ref(null)
const loadingDatabases = ref(false)
const loadingTables = ref(false)
const loadingData = ref(false)
const tableLoadError = ref('')
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)

const deduplicatedInstances = computed(() => {
  const seen = new Set()
  const result = []
  
  for (const inst of instances.value) {
    const host = String(inst.host || 'localhost').trim()
    const username = String(inst.username || '').trim()
    const port = Number(inst.port || 3306)
    const key = `${host}:${username}:${port}`
    
    if (!seen.has(key)) {
      seen.add(key)
      result.push(inst)
    }
  }
  
  return result
})

const navigateToInstance = (inst) => {
  selectInstance(inst)
}

const databases = ref([])
const tables = ref([])
const columns = ref([])
const tableData = ref([])
const totalRows = ref(0)
const page = ref(1)
const pageSize = ref(50)

const activeTab = ref('structure')
const sqlQuery = ref('')
const sqlResult = ref(null)
const sqlResultLoading = ref(false)
const showInsertDialog = ref(false)
const showEditDialog = ref(false)
const insertForm = ref({})
const editForm = ref({})

const showInstanceDialog = ref(false)
const instanceDialogType = ref('create')
const instanceEditData = ref(null)

const showCreateDbDialog = ref(false)
const createDbForm = ref({ name: '', password: '', confirmPassword: '', charset: 'utf8mb4' })
const createDbError = ref('')

const showDeleteDbDialog = ref(false)
const deleteDbName = ref('')
const deleteDbConfirmInput = ref('')

const openDeleteDbDialog = (dbName) => {
  deleteDbName.value = dbName
  deleteDbConfirmInput.value = ''
  showDeleteDbDialog.value = true
}

const doDeleteDatabase = () => {
  const url = `${API_BASE}/databases/delete?${sourceParam(currentInst.value?.isRemote || false)}`
  const body = { server_id: serverId.value, name: deleteDbName.value }
  console.log('[删除数据库] 请求URL:', url)
  console.log('[删除数据库] 请求体:', JSON.stringify(body))
  fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
    .then(res => {
      console.log('[删除数据库] 响应状态:', res.status)
      return res.json()
    })
    .then(data => {
      console.log('[删除数据库] 响应数据:', data)
      if (data.code === 0) {
        toast.success(`数据库 ${deleteDbName.value} 已删除`)
        showDeleteDbDialog.value = false
        if (selectedDatabase.value === deleteDbName.value) {
          selectedDatabase.value = null
          store.setDatabase('')
          tables.value = []; columns.value = []; tableData.value = []; totalRows.value = 0
        }
        loadDatabases()
      } else {
        toast.error(data.msg || '删除失败')
      }
    })
    .catch((err) => {
      console.error('[删除数据库] 请求异常:', err)
      toast.error('删除失败: ' + err.message)
    })
}

const openCreateDbDialog = () => {
  createDbForm.value = { name: '', password: '', confirmPassword: '', charset: 'utf8mb4' }
  showCreateDbDialog.value = true
}

const confirmState = ref({
  open: false, title: '', description: '', variant: 'default', onConfirm: null,
})

const tableHeight = computed(() => selectedTable.value ? 'calc(100vh - 480px)' : '0')
const totalPages = computed(() => Math.max(1, Math.ceil(totalRows.value / pageSize.value)))

const showCreateTableDialog = ref(false)
const showDeleteTableDialog = ref(false)
const deleteTableName = ref('')
const createTableError = ref('')
const columnTypes = ['INT', 'BIGINT', 'VARCHAR', 'TEXT', 'BOOLEAN', 'DATE', 'DATETIME', 'TIMESTAMP', 'FLOAT', 'DOUBLE', 'DECIMAL', 'CHAR', 'BLOB', 'JSON', 'ENUM']
const createTableForm = ref({ name: '', columns: [] })

const addColumn = () => {
  createTableForm.value.columns.push({
    name: '', type: 'INT', length: '', notNull: false, autoIncrement: false,
    defaultValue: '', keyType: '',
  })
}

const removeColumn = (idx) => {
  createTableForm.value.columns.splice(idx, 1)
}

const openCreateTableDialog = () => {
  createTableForm.value = { name: '', columns: [] }
  createTableError.value = ''
  addColumn()
  showCreateTableDialog.value = true
}

const createTable = () => {
  const form = createTableForm.value
  if (!form.name.trim()) { createTableError.value = '请输入表名'; return }
  if (form.columns.length === 0) { createTableError.value = '请添加至少一个字段'; return }
  const validColumns = form.columns.filter(c => c.name.trim())
  if (validColumns.length === 0) { createTableError.value = '请填写字段名'; return }
  createTableError.value = ''

  const pkColumns = validColumns.filter(c => c.keyType === 'PRI')
  const nonPkColumns = validColumns.filter(c => c.keyType !== 'PRI')

  const cols = validColumns.map(col => {
    let def = `\`${col.name.trim().replace(/`/g, '``')}\` ${col.type}`
    if (col.length.trim()) def += `(${col.length.trim()})`
    if (col.notNull) def += ' NOT NULL'
    if (col.autoIncrement) def += ' AUTO_INCREMENT'
    
    if (col.defaultValue && col.keyType !== 'PRI') {
      const defaultVal = col.defaultValue.trim()
      if (defaultVal && defaultVal.toUpperCase() !== 'NULL') {
        // 判断是否是数值类型
        const isNumeric = /^(int|bigint|float|double|decimal|tinyint|smallint|mediumint|numeric)/i.test(col.type)
        if (isNumeric) {
          def += ` DEFAULT ${col.defaultValue}`
        } else {
          def += ` DEFAULT '${col.defaultValue.replace(/'/g, "\\'")}'`
        }
      }
    }
    if (col.keyType === 'UNI') def += ' UNIQUE'
    else if (col.keyType === 'MUL') def += ' INDEX'
    return def
  })

  if (pkColumns.length > 1) {
    cols.push(`PRIMARY KEY (${pkColumns.map(c => `\`${c.name.trim().replace(/`/g, '``')}\``).join(', ')})`)
  } else if (pkColumns.length === 1) {
    const idx = validColumns.indexOf(pkColumns[0])
    cols[idx] += ' PRIMARY KEY'
  }

  const sql = `CREATE TABLE \`${form.name.trim().replace(/`/g, '``')}\` (${cols.join(', ')}) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success(`表 ${form.name.trim()} 创建成功`)
        showCreateTableDialog.value = false
        loadTables()
      } else {
        createTableError.value = data.msg || '创建失败'
      }
    })
    .catch(() => { createTableError.value = '请求失败' })
}

const confirmDeleteTable = (tbl) => {
  deleteTableName.value = tbl
  showDeleteTableDialog.value = true
}

const deleteTable = () => {
  const tbl = deleteTableName.value
  const sql = `DROP TABLE IF EXISTS \`${tbl}\``
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        toast.success(`表 ${tbl} 已删除`)
        showDeleteTableDialog.value = false
        loadTables()
      } else {
        toast.error(data.msg || '删除失败')
        showDeleteTableDialog.value = false
      }
    })
    .catch(() => { toast.error('删除失败'); showDeleteTableDialog.value = false })
}

const currentInst = computed(() => {
  if (!connectionId.value) return null
  return instances.value.find(i => instanceUid(i) === connectionId.value) || null
})

const hashEntries = computed(() => {
  const data = selectedRedisKeyData.value?.value
  if (!data || typeof data !== 'object') return []
  return Object.entries(data).map(([field, value]) => ({ field, value }))
})

const getKeyTypeBadge = (type) => {
  const map = { string: 'default', list: 'secondary', set: 'outline', hash: 'secondary', zset: 'destructive' }
  return map[type] || 'outline'
}

const formatKeySize = (type, size) => {
  if (type === 'string') return size + ' 字符'
  if (type === 'list') return size + ' 个元素'
  if (type === 'set') return size + ' 个成员'
  if (type === 'hash') return size + ' 个字段'
  if (type === 'zset') return size + ' 个成员'
  return ''
}

const getInstanceColor = (inst) => {
  if (inst.type === 'redis') return '#e6a23c'
  if (inst.isRemote) return '#67c23a'
  return '#3b82f6'
}

const isNumericType = (t) => /^(int|tinyint|smallint|mediumint|bigint|float|double|decimal|numeric|number)/i.test(t || '')
const isBooleanType = (t) => /^(bit|bool|boolean)/i.test(t || '')
const isDateType = (t) => /^(date|datetime|timestamp|time)/i.test(t || '')
const isTextType = (t) => /^(text|mediumtext|longtext|blob|mediumblob|longblob)/i.test(t || '')
const getDateInputType = (t) => /^date$/i.test(t) ? 'date' : /^time$/i.test(t) ? 'time' : 'datetime-local'

const escapeSqlName = (n) => n.replace(/`/g, '``')
const escapeSqlValue = (v) => {
  if (v === undefined || v === null) return 'NULL'
  if (typeof v === 'boolean') return v ? '1' : '0'
  if (typeof v === 'string') {
    return `'${v.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/\x00/g, '\\0').replace(/\n/g, '\\n').replace(/\r/g, '\\r').replace(/\x1a/g, '\\Z')}'`
  }
  if (typeof v === 'number' && isFinite(v)) return String(v)
  return 'NULL'
}

const showConfirm = (title, description, onConfirm, variant = 'destructive') => {
  confirmState.value = { open: true, title, description, variant, onConfirm }
}

const handleConfirm = () => {
  confirmState.value.open = false
  confirmState.value.onConfirm?.()
}

const loadInstances = () => {
  store.loadInstances().then(result => {
    const newIds = new Set(result.map(i => (i.isRemote ? 'r:' : 'l:') + i.id))
    const newInstanceIds = [...newIds].filter(uid => !previousInstanceIds.value.has(uid))
    instances.value = result
    previousInstanceIds.value = newIds
    healthStore.cleanup([...newIds])
    newInstanceIds.forEach(uid => {
      healthStore.forceCheckOne(uid)
    })
    checkOnlineStatus(result)
    if (!connectionId.value && instances.value.length > 0) {
      const inst = instances.value[0]
      store.setContext({
        connectionId: instanceUid(inst), userName: inst.username, dbName: '',
        type: inst.type, host: inst.host, port: inst.port,
        isRemote: inst.isRemote || false, name: inst.name,
      })
    }
    autoOpenFromStore()
    completeProgress?.()
  })
}

const checkOnlineStatus = async (items) => {
  await healthStore.refreshAll()
}

const autoOpenFromStore = () => {
  if (!connectionId.value) return
  if (isRedis.value) {
    loadRedisInfo(); loadRedisKeys()
    return
  }
  loadDatabases()
  const savedDb = store.dbName
  if (savedDb) {
    selectedDatabase.value = savedDb
    loadTables()
  }
}

const selectInstance = (inst) => {
  store.setContext({
    connectionId: instanceUid(inst), userName: inst.username, dbName: '',
    type: inst.type, host: inst.host, port: inst.port,
    isRemote: inst.isRemote || false, name: inst.name,
  })
}

const clearDataState = () => {
  selectedDatabase.value = null; selectedTable.value = null
  databases.value = []; tables.value = []; columns.value = []; tableData.value = []
  totalRows.value = 0; page.value = 1
  redisKeys.value = []; selectedRedisKey.value = null; selectedRedisKeyData.value = null
  redisCursor.value = '0'
  loadingTables.value = false; loadingDatabases.value = false; loadingData.value = false
  tableLoadError.value = ''
}

const backToDatabases = () => {
  if (isRedis.value) { selectedRedisKey.value = null; selectedRedisKeyData.value = null; return }
  selectedDatabase.value = null; clearTablesState()
}

const backToTables = () => { clearTablesState() }

const clearTablesState = () => {
  selectedTable.value = null; columns.value = []; tableData.value = []; totalRows.value = 0; page.value = 1
}

const selectDatabase = (db) => {
  if (selectedDatabase.value === db) return
  selectedDatabase.value = db
  store.setDatabase(db)
  clearTablesState(); tables.value = []; loadTables()
}

const selectTable = (tbl) => {
  if (selectedTable.value === tbl) return
  selectedTable.value = tbl; page.value = 1; columns.value = []; tableData.value = []
  loadColumns(); fetchTableData()
}

const loadDatabases = () => {
  if (!connectionId.value) return
  loadingDatabases.value = true
  fetch(`${API_BASE}/databases?server_id=${serverId.value}&${sourceParam(currentInst.value?.isRemote || false)}`)
    .then(res => res.json())
    .then(data => { if (data.code === 0) databases.value = data.data || []; else toast.error(data.msg || '获取数据库列表失败') })
    .catch(err => toast.error('连接MySQL失败: ' + err.message))
    .finally(() => { loadingDatabases.value = false })
}

const loadTables = () => {
  if (!connectionId.value || !selectedDatabase.value) return
  loadingTables.value = true; tableLoadError.value = ''
  fetch(`${API_BASE}/tables?server_id=${serverId.value}&${sourceParam(currentInst.value?.isRemote || false)}&database=${encodeURIComponent(selectedDatabase.value)}`)
    .then(res => res.json())
    .then(data => { if (data.code === 0) tables.value = data.data || []; else { tableLoadError.value = data.msg || '获取表列表失败'; toast.error(data.msg || '获取表列表失败') } })
    .catch(err => { tableLoadError.value = err.message; toast.error('连接MySQL失败: ' + err.message) })
    .finally(() => { loadingTables.value = false })
}

const loadColumns = () => {
  if (!connectionId.value) return
  fetch(`${API_BASE}/columns?server_id=${serverId.value}&${sourceParam(currentInst.value?.isRemote || false)}&database=${encodeURIComponent(selectedDatabase.value)}&table=${encodeURIComponent(selectedTable.value)}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) columns.value = (data.data || []).map(col => ({ name: col.Field, type: col.Type, key: col.Key, default: col.Default, extra: col.Extra }))
      else toast.error(data.msg || '获取列信息失败')
    })
    .catch(err => toast.error('连接失败: ' + err.message))
}

const fetchTableData = () => {
  if (!connectionId.value) return
  loadingData.value = true
  fetch(`${API_BASE}/data?${new URLSearchParams({ server_id: serverId.value, source: sourceValue(currentInst.value?.isRemote || false), database: selectedDatabase.value, table: selectedTable.value, page: page.value, pageSize: pageSize.value })}`)
    .then(res => res.json())
    .then(data => { if (data.code === 0) { tableData.value = data.data.rows || []; totalRows.value = data.data.total || 0 } else toast.error(data.msg || '获取数据失败') })
    .catch(err => toast.error('连接失败: ' + err.message))
    .finally(() => { loadingData.value = false })
}

const processNavRequest = () => {
  if (!props.navRequest) return
  const targetUid = props.navRequest.instanceUid || instanceUid({ id: props.navRequest.id, host: '', port: 0, username: '', isRemote: props.navRequest.isRemote })
  let inst = instances.value.find(i => instanceUid(i) === targetUid)
  if (!inst) {
    inst = instances.value.find(i => i.id === props.navRequest.id)
    if (!inst) return
  }
  selectInstance(inst); emit('navAccepted')
}

const refreshData = () => {
  if (isRedis.value) { loadRedisKeys(); loadRedisInfo(); return }
  if (selectedTable.value) { fetchTableData(); toast.success('刷新成功'); return }
  if (selectedDatabase.value) { loadTables(); toast.success('刷新成功'); return }
  loadDatabases(); toast.success('刷新成功')
}

const createDatabase = () => {
  if (!connectionId.value) return
  const name = createDbForm.value.name.trim()
  if (!name) { toast.warning('请输入数据库名称'); return }
  if (createDbForm.value.password && createDbForm.value.password !== createDbForm.value.confirmPassword) {
    createDbError.value = '两次输入的密码不一致'
    return
  }
  createDbError.value = ''
  const url = `${API_BASE}/databases/create?${sourceParam(currentInst.value?.isRemote || false)}`
  const body = {
    server_id: serverId.value,
    name,
    password: createDbForm.value.password,
    charset: createDbForm.value.charset,
  }
  console.log('[创建数据库] 请求URL:', url)
  console.log('[创建数据库] 请求体:', JSON.stringify(body))
  fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
    .then(res => {
      console.log('[创建数据库] 响应状态:', res.status)
      return res.json()
    })
    .then(data => {
      console.log('[创建数据库] 响应数据:', data)
      if (data.code === 0) {
        toast.success('数据库创建成功')
        showCreateDbDialog.value = false
        createDbForm.value = { name: '', password: '', confirmPassword: '', charset: 'utf8mb4' }
        createDbError.value = ''
        loadDatabases()
      } else {
        createDbError.value = data.msg || '创建数据库失败'
        toast.error(data.msg || '创建数据库失败')
      }
    })
    .catch(err => {
      console.error('[创建数据库] 请求异常:', err)
      createDbError.value = '创建数据库失败: ' + err.message
      toast.error('创建数据库失败: ' + err.message)
    })
}
const onPageChange = () => { fetchTableData() }

const addInstance = () => { instanceEditData.value = null; instanceDialogType.value = 'create'; showInstanceDialog.value = true }
const editInstance = (inst) => { instanceEditData.value = { ...inst }; instanceDialogType.value = 'edit'; showInstanceDialog.value = true }

const confirmDeleteInstance = (inst) => { showConfirm('删除确认', '确定删除该实例？', () => deleteInstance(inst)) }

const deleteInstance = (inst) => {
  const deletePromise = inst.isRemote
    ? fetch(`/api/remote-servers/${inst.id}`, { method: 'DELETE' }).then(r => r.json())
    : databaseApi.delete({ id: inst.id }).then(res => res.data)
  deletePromise.then(data => {
    if (data.code === 0) {
      toast.success('删除成功')
      if (connectionId.value === instanceUid(inst)) { store.clearContext(); clearDataState() }
      loadInstances()
    } else toast.error(data.msg || '删除失败')
  }).catch(() => toast.error('删除失败'))
}

const editRow = (row) => { editForm.value = { ...row }; showEditDialog.value = true }
const confirmDeleteRow = (row) => { showConfirm('删除确认', '确定删除该行数据？', () => doDeleteRow(row)) }

const doDeleteRow = (row) => {
  if (!connectionId.value) return
  const pkCol = columns.value.find(c => c.key === 'PRI')
  if (!pkCol) { toast.warning('该表没有主键，无法删除'); return }
  const sql = `DELETE FROM \`${escapeSqlName(selectedDatabase.value)}\`.\`${escapeSqlName(selectedTable.value)}\` WHERE \`${escapeSqlName(pkCol.name)}\` = ${escapeSqlValue(row[pkCol.name])}`
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql }) })
    .then(res => res.json()).then(data => { if (data.code === 0) { toast.success('删除成功'); fetchTableData() } else toast.error(data.msg) })
    .catch(err => toast.error('操作失败: ' + err.message))
}

const insertData = () => {
  if (!connectionId.value) return
  const fields = columns.value.filter(c => c.extra !== 'auto_increment')
  const names = fields.map(c => `\`${escapeSqlName(c.name)}\``).join(', ')
  const vals = fields.map(c => escapeSqlValue(insertForm.value[c.name])).join(', ')
  const sql = `INSERT INTO \`${escapeSqlName(selectedDatabase.value)}\`.\`${escapeSqlName(selectedTable.value)}\` (${names}) VALUES (${vals})`
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql }) })
    .then(res => res.json()).then(data => { if (data.code === 0) { toast.success('插入成功'); showInsertDialog.value = false; insertForm.value = {}; fetchTableData() } else toast.error(data.msg) })
    .catch(err => toast.error('操作失败: ' + err.message))
}

const updateData = () => {
  if (!connectionId.value) return
  const pkCol = columns.value.find(c => c.key === 'PRI')
  if (!pkCol) { toast.warning('该表无主键，无法安全更新，请使用SQL控制台手动执行'); return }
  const sets = columns.value.filter(c => c.key !== 'PRI' && c.extra !== 'auto_increment')
    .map(c => `\`${escapeSqlName(c.name)}\` = ${escapeSqlValue(editForm.value[c.name])}`).join(', ')
  if (!sets) { toast.warning('没有可更新的字段'); return }
  const sql = `UPDATE \`${escapeSqlName(selectedDatabase.value)}\`.\`${escapeSqlName(selectedTable.value)}\` SET ${sets} WHERE \`${escapeSqlName(pkCol.name)}\` = ${escapeSqlValue(editForm.value[pkCol.name])}`
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql }) })
    .then(res => res.json()).then(data => { if (data.code === 0) { toast.success('保存成功'); showEditDialog.value = false; fetchTableData() } else toast.error(data.msg) })
    .catch(err => toast.error('操作失败: ' + err.message))
}

const escapeCsvValue = (v) => {
  if (v === null || v === undefined) return 'NULL'
  const s = String(v)
  if (s.includes(',') || s.includes('"') || s.includes('\n') || s.includes('\r')) {
    return '"' + s.replace(/"/g, '""') + '"'
  }
  return s
}

const exportData = async (exportAll = false) => {
  if (!tableData.value.length && !exportAll) { toast.warning('没有数据可导出'); return }

  let dataToExport = tableData.value
  if (exportAll) {
    try {
      const res = await fetch(`${API_BASE}/data?${new URLSearchParams({ server_id: serverId.value, source: sourceValue(currentInst.value?.isRemote || false), database: selectedDatabase.value, table: selectedTable.value, page: 1, pageSize: totalRows.value })}`)
      const data = await res.json()
      if (data.code === 0) {
        dataToExport = data.data.rows || []
      } else {
        toast.error(data.msg || '获取全量数据失败')
        return
      }
    } catch (err) {
      toast.error('获取全量数据失败: ' + err.message)
      return
    }
  }

  const cols = columns.value.map(c => c.name)
  let csv = cols.join(',') + '\n'
  dataToExport.forEach(row => {
    csv += cols.map(c => escapeCsvValue(row[c])).join(',') + '\n'
  })
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href = url; a.download = `${selectedTable.value}_${new Date().toISOString().slice(0,10)}.csv`
  a.click(); URL.revokeObjectURL(url); toast.success(exportAll ? `已导出全部 ${dataToExport.length} 条数据` : '导出成功')
}

const DANGEROUS_SQL_PATTERNS = [
  /^\s*DROP\s+(DATABASE|SCHEMA|TABLE)/i,
  /^\s*TRUNCATE\s+/i,
  /^\s*DELETE\s+FROM\s+\S+\s*$/i,
  /^\s*DELETE\s+FROM\s+\S+\s*;\s*$/i,
]

const executeSql = () => {
  if (!sqlQuery.value.trim()) { toast.warning('请输入SQL语句'); return }
  if (!connectionId.value) { toast.warning('请先选择一个实例'); return }
  const sqlTrimmed = sqlQuery.value.trim()
  const isDangerous = DANGEROUS_SQL_PATTERNS.some(p => p.test(sqlTrimmed))
  if (isDangerous && !confirm('该SQL语句可能造成数据不可逆的修改，确认执行？')) return
  sqlResultLoading.value = true
  sqlResult.value = null
  fetch(`${API_BASE}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ server_id: serverId.value, database: selectedDatabase.value, sql: sqlQuery.value }) })
    .then(res => res.json()).then(data => {
      if (data.code === 0) {
        if (data.data.rows && data.data.rows.length > 0) {
          sqlResult.value = { type: 'rows', data: data.data.rows, columns: Object.keys(data.data.rows[0]) }
          toast.success(`查询成功，返回 ${data.data.rows.length} 行`)
        } else {
          sqlResult.value = { type: 'message', data: data.data.msg || '执行成功' }
          toast.success(data.data.msg || '执行成功')
        }
      } else {
        sqlResult.value = { type: 'error', data: data.msg }
        toast.error(data.msg)
      }
    })
    .catch(err => {
      sqlResult.value = { type: 'error', data: err.message }
      toast.error('执行失败: ' + err.message)
    })
    .finally(() => { sqlResultLoading.value = false })
}

const redisKeys = ref([])
const redisInfo = ref({})
const redisPattern = ref('*')
const redisCursor = ref('0')
const selectedRedisKey = ref(null)
const selectedRedisKeyData = ref(null)
const loadingRedisKeys = ref(false)

const loadRedisInfo = () => {
  if (!connectionId.value) return
  fetch(`${REDIS_API}/info?server_id=${serverId.value}&${sourceParam(currentInst.value?.isRemote || false)}`)
    .then(res => res.json()).then(data => {
      if (data.code === 0) {
        const info = data.data || {}; let totalKeys = 0
        for (const key of Object.keys(info)) { if (key.startsWith('db')) { const match = String(info[key]).match(/keys=(\d+)/); if (match) totalKeys += parseInt(match[1]) } }
        info._totalKeys = totalKeys; redisInfo.value = info
      }
    }).catch((e) => { console.error(e) })
}

const loadRedisKeys = (append = false) => {
  if (!connectionId.value) return
  loadingRedisKeys.value = true
  fetch(`${REDIS_API}/keys?${new URLSearchParams({ server_id: serverId.value, source: sourceValue(currentInst.value?.isRemote || false), pattern: redisPattern.value || '*', cursor: redisCursor.value === '0' ? '0' : redisCursor.value, count: '50' })}`)
    .then(res => res.json()).then(data => {
      if (data.code === 0) { const newKeys = data.data.keys || []; redisKeys.value = append ? [...redisKeys.value, ...newKeys] : newKeys; redisCursor.value = String(data.data.nextCursor || '0') }
      else toast.error(data.msg || '获取keys失败')
    }).catch(err => toast.error('连接Redis失败: ' + err.message))
    .finally(() => { loadingRedisKeys.value = false })
}

const searchRedisKeys = () => { if (!redisPattern.value.trim()) redisPattern.value = '*'; redisCursor.value = '0'; loadRedisKeys() }
const loadMoreRedisKeys = () => { if (redisCursor.value === '0') return; loadRedisKeys(true) }

const selectRedisKey = async (key) => {
  selectedRedisKey.value = key; selectedRedisKeyData.value = null
  if (!connectionId.value) return
  try {
    const res = await fetch(`${REDIS_API}/key?server_id=${serverId.value}&${sourceParam(currentInst.value?.isRemote || false)}&key=${encodeURIComponent(key)}`)
    const data = await res.json()
    if (data.code === 0) selectedRedisKeyData.value = data.data; else toast.error(data.msg || '获取key值失败')
  } catch (err) { toast.error('获取key值失败: ' + err.message) }
}

const DANGEROUS_REDIS_COMMANDS = ['FLUSHALL', 'FLUSHDB', 'SHUTDOWN', 'DEBUG', 'SCRIPT', 'SLAVEOF', 'REPLICAOF', 'BGSAVE', 'BGREWRITEAOF', 'CLUSTER']

const executeRedisCmd = () => {
  if (!redisCmd.value.trim()) { toast.warning('请输入Redis命令'); return }
  if (!connectionId.value) return
  const parts = redisCmd.value.trim().split(/\s+/)
  const cmd = parts[0].toUpperCase()
  if (DANGEROUS_REDIS_COMMANDS.includes(cmd)) {
    toast.error(`禁止执行危险命令: ${cmd}，请使用设置页面进行相关操作`)
    return
  }
  fetch(`${REDIS_API}/execute?${sourceParam(currentInst.value?.isRemote || false)}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ server_id: serverId.value, command: cmd, args: parts.slice(1) }) })
    .then(res => res.json()).then(data => {
      if (data.code === 0) {
        const result = data.data.result
        if (typeof result === 'string') redisCmdResult.value = result
        else if (typeof result === 'number') redisCmdResult.value = String(result)
        else redisCmdResult.value = JSON.stringify(result, null, 2)
        toast.success('命令执行成功')
      } else { redisCmdResult.value = '错误: ' + data.msg; toast.error(data.msg) }
    })
    .catch(err => { redisCmdResult.value = '错误: ' + err.message; toast.error('执行失败: ' + err.message) })
}

const redisCmd = ref('')
const redisCmdResult = ref('')

watch(connectionId, (newId) => {
  if (newId) { clearDataState(); autoOpenFromStore() }
})

watch(() => props.navRequest, (val) => { if (val) processNavRequest() })

onMounted(() => { loadInstances() })
onActivated(() => { loadInstances() })
</script>

<style scoped>
/* ── Instance Cards ── */
.instance-card {
  padding: 0.875rem;
  border-radius: var(--card-radius);
  border: var(--card-border);
  background: var(--surface);
  box-shadow: var(--card-shadow);
  cursor: pointer;
  transition: all var(--transition-normal);
  margin-bottom: 0.5rem;
}
.instance-card:hover {
  box-shadow: var(--card-shadow-hover);
  transform: translateY(-1px);
  border-color: color-mix(in srgb, var(--accent) 20%, var(--border));
}
.instance-card:active {
  transform: translateY(0);
  box-shadow: var(--card-shadow);
}
.instance-card--active {
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
  background: var(--accent-soft);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--accent) 30%, transparent);
}
.instance-icon {
  width: 2rem;
  height: 2rem;
  border-radius: 0.625rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.instance-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 0.5rem;
  padding-left: 2.75rem;
}

/* ── Breadcrumb Navigation ── */
.breadcrumb-nav {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.9375rem;
  flex-wrap: wrap;
}
.breadcrumb-item {
  padding: 0.125rem 0.375rem;
  border-radius: 0.375rem;
  transition: all var(--transition-fast);
  cursor: default;
}
.breadcrumb-item--active {
  color: var(--text-primary);
  font-weight: 500;
  cursor: pointer;
}
.breadcrumb-item--active:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
.breadcrumb-item--current {
  color: var(--text-primary);
  font-weight: 600;
}
.breadcrumb-sep {
  width: 1rem;
  height: 1rem;
  color: var(--text-tertiary);
  flex-shrink: 0;
}

/* ── Section Labels ── */
.section-label {
  font-size: 0.8125rem;
  font-weight: 500;
  margin-bottom: 0.75rem;
}

/* ── Grid Cards (Database / Table) ── */
.grid-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.875rem 1rem;
  border-radius: var(--card-radius);
  border: var(--card-border);
  background: var(--surface);
  box-shadow: var(--card-shadow);
  cursor: pointer;
  transition: all var(--transition-normal);
}
.grid-card:hover {
  box-shadow: var(--card-shadow-hover);
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--accent) 25%, var(--border));
}
.grid-card:active {
  transform: translateY(0);
  box-shadow: var(--card-shadow);
}
.grid-card-icon {
  width: 2rem;
  height: 2rem;
  border-radius: 0.625rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.grid-card-info {
  flex: 1;
  min-width: 0;
}
.grid-card-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-primary);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.grid-card-delete {
  position: absolute;
  top: 0.375rem;
  right: 0.375rem;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 0.375rem;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: all var(--transition-fast);
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--danger);
}
.grid-card:hover .grid-card-delete {
  opacity: 1;
}
.grid-card-delete:hover {
  background: var(--danger-soft);
}

/* ── Redis Stats Grid ── */
.redis-stats {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-bottom: 0.75rem;
}

/* ── Redis Search ── */
.redis-search-bar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.75rem;
}
.redis-search-input {
  display: flex;
  flex: 1;
  max-width: 400px;
}
.redis-search-prefix {
  display: inline-flex;
  align-items: center;
  border-radius: 0.5rem 0 0 0.5rem;
  border: 1px solid var(--border);
  border-right: none;
  padding: 0 0.75rem;
  font-size: 0.75rem;
  color: var(--text-tertiary);
  background: var(--muted);
}
.redis-search-field {
  border-radius: 0 0.5rem 0.5rem 0 !important;
  height: 2rem;
  font-size: 0.75rem;
}

/* ── Redis Key List ── */
.redis-key-list {
  border-radius: var(--card-radius);
  border: 1px solid var(--border);
  overflow: hidden;
  position: relative;
}
.redis-key-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--background) 60%, transparent);
  z-index: 10;
}
.redis-key-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.625rem 1rem;
  cursor: pointer;
  transition: background var(--transition-fast);
  border-bottom: 1px solid var(--border-subtle);
}
.redis-key-item:last-child {
  border-bottom: none;
}
.redis-key-item:hover {
  background: var(--accent-soft);
}
.redis-key-more {
  text-align: center;
  padding: 0.5rem;
  border-top: 1px solid var(--border);
}

/* ── Redis Console ── */
.redis-console {
  margin-top: 0.75rem;
  border-radius: var(--card-radius);
  border: 1px solid var(--border);
  padding: 0.75rem 1rem;
}
.redis-console-header {
  margin-bottom: 0.5rem;
}
.redis-console-result {
  margin-top: 0.5rem;
  padding: 0.5rem;
  max-height: 120px;
  overflow: auto;
}

/* ── Redis Value Panel ── */
.redis-value-panel {
  overflow: auto;
  border-radius: var(--card-radius);
  border: 1px solid var(--border);
  padding: 0.75rem;
}
.redis-value-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 0.8125rem;
}
.redis-value-row:last-child {
  border-bottom: none;
}
.redis-value-idx {
  color: var(--text-tertiary);
  min-width: 30px;
  font-size: 0.6875rem;
}

/* ── Data Table ── */
.data-table-pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--section-padding-y) var(--section-padding-x);
  border-bottom: 1px solid var(--border-subtle);
}
.pagination-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 1.75rem;
  width: 1.75rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-primary);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.pagination-btn:hover:not(:disabled) {
  background: var(--muted);
  border-color: var(--border-strong);
}
.pagination-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.data-table-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--background) 60%, transparent);
  z-index: 10;
}
.data-table-row {
  transition: background var(--transition-fast);
  border-bottom: 1px solid var(--border-subtle);
}
.data-table-row:hover {
  background: var(--muted);
}
.data-table-action-col {
  text-align: center;
  width: 100px;
  position: sticky;
  right: 0;
  background: var(--surface);
}
.data-table-row-actions {
  display: flex;
  gap: 0.25rem;
  opacity: 0;
  transition: opacity var(--transition-fast);
}
.data-table-row:hover .data-table-row-actions {
  opacity: 1;
}
.data-table-null {
  color: var(--text-tertiary);
  font-style: italic;
}

/* ── Bottom Panel ── */
.data-table-bottom-panel {
  padding: var(--section-padding-y) var(--section-padding-x);
  border-top: 1px solid var(--border-subtle);
}

/* ── SQL Console ── */
.sql-console {
  /* wrapper for SQL editor area */
}
.sql-result-table {
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  overflow: auto;
  max-height: 400px;
}
.sql-result-footer {
  font-size: 0.75rem;
  padding: 0.5rem 0.75rem;
  border-top: 1px solid var(--border);
}
.sql-result-msg {
  padding: 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
}
.sql-result-msg--success {
  background: var(--success-soft);
  border: 1px solid color-mix(in srgb, var(--success) 20%, transparent);
  color: var(--success);
}
.sql-result-msg--error {
  background: var(--danger-soft);
  border: 1px solid color-mix(in srgb, var(--danger) 20%, transparent);
  color: var(--danger);
}
</style>
