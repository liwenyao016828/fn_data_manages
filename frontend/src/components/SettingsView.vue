<template>
  <div class="page-padding h-full overflow-y-auto" ref="pageContainer">
    <!-- Page Header -->
    <div class="mb-4">
      <h2 class="text-[17px] font-semibold" style="color: var(--text-primary)">设置</h2>
      <p class="text-[13px] mt-0.5" style="color: var(--text-tertiary)">管理系统偏好与数据库实例配置</p>
    </div>

    <!-- Tab Bars -->
    <div class="flex flex-col">
      <!-- Main Tab Bar -->
      <div class="inline-flex gap-1 p-1 rounded-xl bg-[var(--muted)] self-start">
        <button
          :class="activeTab === 'instance' ? 'tab-active' : 'tab-inactive'"
          class="px-4 py-1.5 text-[13px]"
          @click="activeTab = 'instance'"
        >实例管理</button>
        <button
          :class="activeTab === 'system' ? 'tab-active' : 'tab-inactive'"
          class="px-4 py-1.5 text-[13px]"
          @click="activeTab = 'system'"
        >系统设置</button>
      </div>

      <!-- Sub Tab Bar: Instance -->
      <div v-if="activeTab === 'instance'" class="inline-flex gap-1 p-1 rounded-xl bg-[var(--muted)] mt-2 self-start">
        <button
          :class="instanceTab === 'local' ? 'tab-active' : 'tab-inactive'"
          class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[13px]"
          @click="instanceTab = 'local'"
        >
          本地实例
          <span class="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[11px] font-semibold"
            :class="instanceTab === 'local' ? 'pill-active' : 'pill-default'">
            {{ localInstances.length }}
          </span>
        </button>
        <button
          :class="instanceTab === 'remote' ? 'tab-active' : 'tab-inactive'"
          class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[13px]"
          @click="instanceTab = 'remote'"
        >
          远程服务器
          <span class="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[11px] font-semibold"
            :class="instanceTab === 'remote' ? 'pill-active' : 'pill-default'">
            {{ remoteInstances.length }}
          </span>
        </button>
      </div>

      <!-- Sub Tab Bar: System -->
      <div v-if="activeTab === 'system'" class="inline-flex gap-1 p-1 rounded-xl bg-[var(--muted)] mt-2 self-start">
        <button
          v-for="item in [
            { key: 'logs', label: '日志中心' },
            { key: 'interface', label: '界面设置' },
            { key: 'data', label: '数据设置' },
            { key: 'health', label: '测活配置' },
            { key: 'about', label: '关于系统' },
          ]"
          :key="item.key"
          :class="systemSubTab === item.key ? 'tab-active' : 'tab-inactive'"
          class="px-4 py-1.5 text-[13px]"
          @click="systemSubTab = item.key"
        >{{ item.label }}</button>
      </div>
    </div>

    <!-- Content Area -->
    <div>

        <!-- 系统设置面板 -->
        <div v-if="activeTab === 'system'" class="flex flex-col gap-4 mt-4">

          <!-- 日志中心设置 -->
          <div v-if="systemSubTab === 'logs'" class="fade-up">
            <div class="content-card flex flex-col">
              <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">启用日志中心</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">关闭后侧边栏隐藏日志中心入口</span>
                </div>
                <Switch :model-value="logEnabled" @update:model-value="toggleLogEnabled" />
              </div>
              <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">日志存储路径</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">日志文件保存的目录路径</span>
                </div>
                <div class="flex items-center gap-2">
                  <Input v-model="logStoragePath" placeholder="存储路径..." class="h-8 text-[13px] w-[180px]" />
                  <Button size="sm" variant="outline" class="h-8 px-2.5" @click="openFolderBrowser">
                    <FolderOpen class="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
              <div class="flex items-center justify-between px-5 py-4">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">日志保留天数</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">超过此天数的旧日志将被自动清理</span>
                </div>
                <Select v-model="logRetentionDays" class="w-[140px]" @update:model-value="saveLogConfig">
                  <SelectTrigger class="h-8 text-[13px]"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">7 天</SelectItem>
                    <SelectItem value="15">15 天</SelectItem>
                    <SelectItem value="30">30 天</SelectItem>
                    <SelectItem value="60">60 天</SelectItem>
                    <SelectItem value="90">90 天</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <!-- 界面设置 -->
          <div v-if="systemSubTab === 'interface'" class="fade-up">
            <div class="content-card flex flex-col">
              <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">侧边栏状态</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">{{ sidebarCollapsed ? '已收起' : '已展开' }}</span>
                </div>
                <Button size="sm" variant="outline" class="h-8 text-[13px]" @click="toggleSidebar">
                  {{ sidebarCollapsed ? '展开侧栏' : '收起侧栏' }}
                </Button>
              </div>
              <div class="flex items-center justify-between px-5 py-4">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">紧凑模式</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">减少卡片间距，显示更多内容</span>
                </div>
                <Switch v-model="compactMode" />
              </div>
            </div>
          </div>

          <!-- 数据设置 -->
          <div v-if="systemSubTab === 'data'" class="fade-up">
            <div class="content-card flex flex-col">
              <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">自动刷新间隔</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">控制台页面数据自动轮询间隔</span>
                </div>
                <Select v-model="refreshInterval" class="w-[140px]">
                  <SelectTrigger class="h-8 text-[13px]"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="off">关闭</SelectItem>
                    <SelectItem value="5000">5 秒</SelectItem>
                    <SelectItem value="10000">10 秒</SelectItem>
                    <SelectItem value="30000">30 秒</SelectItem>
                    <SelectItem value="60000">60 秒</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="flex items-center justify-between px-5 py-4">
                <div class="flex flex-col gap-0.5">
                  <span class="text-[13px] font-medium" style="color: var(--text-primary)">备份文件保留</span>
                  <span class="text-[11px]" style="color: var(--text-tertiary)">超过此天数的旧备份将被清理</span>
                </div>
                <Select v-model="backupRetentionDays" class="w-[120px]">
                  <SelectTrigger class="h-8 text-[13px]"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">7 天</SelectItem>
                    <SelectItem value="15">15 天</SelectItem>
                    <SelectItem value="30">30 天</SelectItem>
                    <SelectItem value="90">90 天</SelectItem>
                    <SelectItem value="never">永不</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          <!-- 测活配置 -->
          <div v-if="systemSubTab === 'health'" class="fade-up">
            <div class="content-card">
              <div class="grid grid-cols-1 md:grid-cols-2">
                <div class="flex items-center justify-between px-5 py-4 border-b border-r" style="border-color: var(--border-subtle)">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[13px] font-medium" style="color: var(--text-primary)">启用定时测活</span>
                    <span class="text-[11px]" style="color: var(--text-tertiary)">自动检测连接可用性</span>
                  </div>
                  <Switch v-model="healthConfig.enabled" @update:model-value="saveHealthConfig" />
                </div>
                <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[13px] font-medium" style="color: var(--text-primary)">异常告警</span>
                    <span class="text-[11px]" style="color: var(--text-tertiary)">状态变更时记录日志</span>
                  </div>
                  <Switch v-model="healthConfig.alertEnabled" @update:model-value="saveHealthConfig" />
                </div>
                <div class="flex items-center justify-between px-5 py-4 border-b border-r" style="border-color: var(--border-subtle)">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[13px] font-medium" style="color: var(--text-primary)">测活频率</span>
                    <span class="text-[11px]" style="color: var(--text-tertiary)">检测间隔 10-300 秒</span>
                  </div>
                  <Select v-model="healthIntervalSec" class="w-[120px]" @update:model-value="updateHealthInterval">
                    <SelectTrigger class="h-8 text-[13px]"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="10">10 秒</SelectItem>
                      <SelectItem value="15">15 秒</SelectItem>
                      <SelectItem value="30">30 秒</SelectItem>
                      <SelectItem value="60">60 秒</SelectItem>
                      <SelectItem value="120">120 秒</SelectItem>
                      <SelectItem value="300">300 秒</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="flex items-center justify-between px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[13px] font-medium" style="color: var(--text-primary)">连接超时</span>
                    <span class="text-[11px]" style="color: var(--text-tertiary)">超时时间 1-30 秒</span>
                  </div>
                  <Select v-model="healthTimeoutSec" class="w-[120px]" @update:model-value="updateHealthTimeout">
                    <SelectTrigger class="h-8 text-[13px]"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">1 秒</SelectItem>
                      <SelectItem value="3">3 秒</SelectItem>
                      <SelectItem value="5">5 秒</SelectItem>
                      <SelectItem value="10">10 秒</SelectItem>
                      <SelectItem value="15">15 秒</SelectItem>
                      <SelectItem value="30">30 秒</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="flex items-center justify-between px-5 py-4 md:col-span-2 border-b" style="border-color: var(--border-subtle)">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[13px] font-medium" style="color: var(--text-primary)">立即检测</span>
                    <span class="text-[11px]" style="color: var(--text-tertiary)">立即对所有数据库连接执行一次测活</span>
                  </div>
                  <Button size="sm" variant="outline" class="h-8 text-[13px]" @click="forceHealthCheck" :disabled="forceChecking">
                    <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': forceChecking }" />
                    {{ forceChecking ? '检测中...' : '立即检测' }}
                  </Button>
                </div>
              </div>
            </div>

            <!-- Health Status -->
            <div v-if="healthStore.totalCount > 0" class="content-card mt-4">
              <div class="px-5 py-4 border-b" style="border-color: var(--border-subtle)">
                <div class="flex items-center gap-3">
                  <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg" style="background: var(--accent-soft)">
                    <span class="h-2 w-2 rounded-full" style="background: var(--success)" />
                    <span class="text-[12px] font-medium" style="color: var(--success)">在线 {{ healthStore.onlineCount }}</span>
                  </div>
                  <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg" style="background: color-mix(in srgb, var(--danger) 8%, transparent)">
                    <span class="h-2 w-2 rounded-full" style="background: var(--danger)" />
                    <span class="text-[12px] font-medium" style="color: var(--danger)">离线 {{ healthStore.offlineCount }}</span>
                  </div>
                  <span class="text-[12px]" style="color: var(--text-tertiary)">共 {{ healthStore.totalCount }} 个实例</span>
                </div>
              </div>
              <div class="max-h-[280px] overflow-y-auto">
                <div
                  v-for="item in healthSortedDetails"
                  :key="item.uid"
                  class="flex items-center gap-3 px-5 py-3 border-b transition-colors"
                  :style="{ 'border-color': 'var(--border-subtle)', 'background': item.online ? 'transparent' : 'color-mix(in srgb, var(--danger) 4%, transparent)' }"
                >
                  <StatusDot :status="item.online ? 'online' : 'offline'" size="sm" />
                  <div class="flex-1 min-w-0">
                    <div class="text-[12px] font-medium truncate" style="color: var(--text-primary)">{{ item.name }}</div>
                    <div class="text-[10px] truncate font-mono-data" style="color: var(--text-tertiary)">{{ item.host }}:{{ item.port }}</div>
                  </div>
                  <span class="pill pill-default text-[10px]">{{ item.type === 'redis' ? 'Redis' : 'MySQL' }}</span>
                  <span v-if="item.latencyMs >= 0" class="text-[11px] font-mono-data w-[50px] text-right" style="color: var(--text-tertiary)">{{ item.latencyMs }}ms</span>
                  <span v-if="!item.online && item.error" class="text-xs max-w-[140px] truncate" style="color: var(--danger)" :title="item.error">{{ item.error }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 关于系统 -->
          <div v-if="systemSubTab === 'about'" class="fade-up">
            <div class="content-card">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-0">
                <div v-for="(item, idx) in [
                  { label: '系统版本', value: appVersion || '加载中...', mono: true },
                  { label: '运行端口', value: serverPort || '-', mono: true },
                  { label: '当前用户', value: systemUsername || userName || '-', mono: false },
                  { label: '主机名', value: systemHostname || '-', mono: true },
                  { label: '操作系统', value: systemOsName || '-', mono: true },
                  { label: '本地连接数', value: localInstances.length + ' 个', mono: false },
                ]" :key="item.label" class="flex items-center justify-between px-5 py-3.5 border-b" :style="{ 'border-color': 'var(--border-subtle)' }">
                  <span class="text-[12px]" style="color: var(--text-tertiary)">{{ item.label }}</span>
                  <span class="text-[13px]" :class="item.mono ? 'font-mono-data' : ''" style="color: var(--text-primary)">{{ item.value }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 数据库实例管理面板 -->
        <div v-if="activeTab === 'instance'" class="flex flex-col gap-4 mt-4">
          <!-- Instance Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 grid-gap min-h-[60px]">
            <div
              v-for="db in (instanceTab === 'local' ? localInstances : remoteInstances)"
              :key="db.id"
              class="content-card-interactive hover-lift px-4 py-3 cursor-pointer min-h-[68px]"
              :class="selectedDbId === db.id ? 'ring-2' : ''"
              :style="selectedDbId === db.id ? 'ring-color: var(--accent)' : ''"
              @click="selectInstance(db.id)"
            >
              <div class="text-[13px] font-semibold truncate" style="color: var(--text-primary)">{{ db.name }}</div>
              <div class="flex items-center gap-1.5 text-xs mt-1.5">
                <StatusDot :status="selectedDbId === db.id ? 'selected' : (onlineStatus[instanceUid(db)] !== false ? 'online' : 'offline')" size="xs" />
                <span style="color: var(--text-secondary)">{{ db.type === 'redis' ? 'Redis' : 'MySQL' }}</span>
                <span class="ml-auto font-mono-data text-[11px]" style="color: var(--text-tertiary)">{{ db.host }}:{{ db.port }}</span>
              </div>
            </div>
            <div v-if="(instanceTab === 'local' ? localInstances : remoteInstances).length === 0"
              class="col-span-full">
              <div class="empty-state">
                <div class="empty-state-icon">
                  <Settings class="h-8 w-8" />
                </div>
                <div class="empty-state-text">暂无{{ instanceTab === 'local' ? '本地' : '远程' }}实例</div>
              </div>
            </div>
          </div>

          <!-- Instance Detail -->
          <div v-if="currentDb" ref="detailCard">
            <!-- Detail Sub-tabs -->
            <div class="flex items-center gap-2 mb-4">
              <button
                :class="detailTab === 'info' ? 'tab-active' : 'tab-inactive'"
                class="px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200"
                @click="detailTab = 'info'"
              >基本信息</button>
              <button
                v-if="currentDb.type === 'mysql' || currentDb.type === 'redis'"
                :class="detailTab === 'config' ? 'tab-active' : 'tab-inactive'"
                class="px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200"
                @click="detailTab = 'config'"
              >配置修改</button>
              <button
                v-if="currentDb.type === 'mysql' && currentDb.username === 'root'"
                :class="detailTab === 'users' ? 'tab-active' : 'tab-inactive'"
                class="px-4 py-2 rounded-lg text-[13px] font-medium transition-all duration-200"
                @click="detailTab = 'users'"
              >用户管理</button>
              <span v-if="currentDb.type === 'mysql' && currentDb.username !== 'root'" class="text-[12px] px-2 self-center" style="color: var(--text-tertiary)">用户管理<span style="color: var(--danger)" class="text-[11px] ml-0.5">（仅限 root 账户）</span></span>
            </div>

            <!-- 基本信息 -->
            <div v-if="detailTab === 'info'" class="content-card fade-up">
              <div class="grid grid-cols-1 md:grid-cols-2">
                <div v-for="(item, idx) in [
                  { label: '实例名称', value: currentDb.name, mono: false },
                  { label: '数据库类型', value: currentDb.type === 'mysql' ? 'MySQL' : 'Redis', badge: true },
                  { label: '主机地址', value: currentDb.host, mono: true },
                  { label: '端口', value: currentDb.port + (currentDb.container ? ' (Docker)' : ''), mono: true },
                  { label: '版本', value: currentDb.version || '-', mono: false },
                  { label: '用户名', value: currentDb.username || '-', mono: false },
                ]" :key="item.label" class="flex items-center justify-between px-5 py-3.5 border-b" :style="{ 'border-color': 'var(--border-subtle)' }">
                  <span class="text-[12px]" style="color: var(--text-tertiary)">{{ item.label }}</span>
                  <span v-if="item.badge" class="pill pill-active text-[11px]">{{ item.value }}</span>
                  <span v-else class="text-[13px]" :class="item.mono ? 'font-mono-data' : ''" style="color: var(--text-primary)">{{ item.value }}</span>
                </div>
              </div>
              <div class="flex flex-wrap gap-3 px-5 py-4">
                <Button size="sm" variant="secondary" class="h-8 text-[13px]" @click="confirmAction('restart')">
                  <RefreshCw class="h-4 w-4" />
                  重启实例
                </Button>
                <Button size="sm" variant="destructive" class="h-8 text-[13px]" @click="confirmAction('stop')">
                  <CircleX class="h-4 w-4" />
                  停止实例
                </Button>
              </div>
            </div>

            <!-- 配置修改 -->
            <div v-if="detailTab === 'config' && (currentDb.type === 'mysql' || currentDb.type === 'redis')" class="content-card fade-up">
              <!-- Code Editor -->
              <div class="code-editor rounded-t-lg overflow-hidden border-b" style="border-color: var(--border-subtle)">
                <div class="flex flex-wrap items-center justify-between px-4 py-2.5" style="background: var(--surface)">
                  <span class="text-[12px]" style="color: var(--text-tertiary)">
                    {{ configSource === 'file' ? '配置文件: ' + configFilePath : '运行时变量' }}
                  </span>
                  <div class="flex items-center gap-2">
                    <Button size="sm" variant="outline" class="h-7 text-[12px]" @click="loadConfig(requestId)">
                      <RefreshCw class="h-3.5 w-3.5" />
                      重新加载
                    </Button>
                    <Button variant="primary" size="sm" class="h-7 text-[12px]" @click="saveConfig" :disabled="savingConfig" :loading="savingConfig">
                      <FileText class="h-3.5 w-3.5" />
                      {{ savingConfig ? '保存中...' : '保存配置' }}
                    </Button>
                  </div>
                </div>
                <Textarea
                  v-model="configContent"
                  class="font-mono text-sm leading-relaxed border-0 rounded-none min-h-[360px] focus-visible:ring-0"
                />
              </div>

              <!-- MySQL Common Config -->
              <div class="px-5 py-4" v-if="currentDb.type === 'mysql'">
                <h4 class="text-[13px] font-semibold mb-3" style="color: var(--text-primary)">常用配置项</h4>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-3">
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">最大连接数</label>
                    <Input type="number" v-model="commonConfig.maxConnections" :min="10" :max="10000" @change="updateConfig('max_connections')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">InnoDB缓冲池大小</label>
                    <Input v-model="commonConfig.innodbBufferSize" placeholder="如: 1G" @change="updateConfig('innodb_buffer_pool_size')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">服务端口</label>
                    <div class="flex items-center gap-2">
                      <Input type="number" v-model="commonConfig.port" :min="1024" :max="65535" @change="updateConfig('port')" class="h-8 text-[13px] flex-1" />
                      <Button variant="primary" size="sm" class="h-8 text-[12px] shrink-0" @click="confirmAction('updatePort')" :disabled="updatingPort" :loading="updatingPort">
                        {{ updatingPort ? '修改中...' : '应用端口' }}
                      </Button>
                    </div>
                    <div v-if="currentDb?.container" class="mt-1.5 p-2.5 rounded-lg" style="background: var(--accent-soft); border: 1px solid color-mix(in srgb, var(--accent) 20%, transparent)">
                      <div class="text-[11px] font-medium mb-1.5" style="color: var(--accent)">Docker 端口映射</div>
                      <div class="flex items-center gap-3 text-[11px]">
                        <span style="color: var(--text-tertiary)">容器内: <span class="font-mono-data" style="color: var(--text-primary)">{{ dockerInternalPort }}</span></span>
                        <span style="color: var(--text-tertiary)">→</span>
                        <span style="color: var(--text-tertiary)">宿主机: <span class="font-mono-data" style="color: var(--text-primary)">{{ currentDb.port }}</span></span>
                      </div>
                    </div>
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">数据目录</label>
                    <Input v-model="commonConfig.dataDir" placeholder="/var/lib/mysql" disabled class="h-8 text-[13px]" style="background: var(--surface)" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">慢查询阈值(秒)</label>
                    <Input type="number" v-model="commonConfig.slowQueryTime" :min="0" :max="60" step="0.1" @change="updateConfig('long_query_time')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">绑定地址</label>
                    <Input v-model="commonConfig.bindAddress" placeholder="0.0.0.0" @change="updateConfig('bind_address')" class="h-8 text-[13px]" />
                  </div>
                </div>
              </div>

              <!-- Redis Common Config -->
              <div class="px-5 py-4" v-if="currentDb.type === 'redis'">
                <h4 class="text-[13px] font-semibold mb-3" style="color: var(--text-primary)">常用配置项</h4>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-3">
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">最大内存</label>
                    <Input v-model="redisCommonConfig.maxmemory" placeholder="如: 1gb" @change="updateRedisConfig('maxmemory')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">最大连接数</label>
                    <Input type="number" v-model="redisCommonConfig.maxclients" :min="10" :max="100000" @change="updateRedisConfig('maxclients')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">服务端口</label>
                    <Input type="number" v-model="redisCommonConfig.port" :min="1024" :max="65535" @change="updateRedisConfig('port')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">超时时间(秒)</label>
                    <Input type="number" v-model="redisCommonConfig.timeout" :min="0" :max="86400" @change="updateRedisConfig('timeout')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">数据库数量</label>
                    <Input type="number" v-model="redisCommonConfig.databases" :min="1" :max="256" @change="updateRedisConfig('databases')" class="h-8 text-[13px]" />
                  </div>
                  <div class="flex flex-col gap-1.5">
                    <label class="text-[13px]" style="color: var(--text-primary)">绑定地址</label>
                    <Input v-model="redisCommonConfig.bind" placeholder="0.0.0.0" @change="updateRedisConfig('bind')" class="h-8 text-[13px]" />
                  </div>
                </div>
              </div>
            </div>

            <!-- 用户管理 -->
            <div v-if="detailTab === 'users'" class="fade-up">
              <div v-if="currentDb.type === 'mysql' && currentDb.username === 'root'" class="content-card">
                <div class="flex flex-wrap items-center justify-between px-5 py-3.5 border-b" style="border-color: var(--border-subtle)">
                  <span class="text-[13px]" style="color: var(--text-tertiary)">管理数据库用户及其权限</span>
                  <Button variant="primary" size="sm" class="h-8 text-[13px]" @click="showCreateUserDialog = true">
                    <Plus class="h-4 w-4" />
                    创建用户
                  </Button>
                </div>

                <Table>
                  <TableHeader>
                    <TableRow class="hover:bg-transparent border-b" style="border-color: var(--border-subtle)">
                      <TableHead class="w-[150px] text-[12px] font-normal" style="color: var(--text-tertiary)">用户名</TableHead>
                      <TableHead class="w-[150px] text-[12px] font-normal" style="color: var(--text-tertiary)">主机</TableHead>
                      <TableHead class="min-w-[250px] text-[12px] font-normal" style="color: var(--text-tertiary)">全局权限</TableHead>
                      <TableHead class="text-center text-[12px] font-normal" style="color: var(--text-tertiary)">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="row in users" :key="row.user + row.host" class="hover:bg-muted border-b" style="border-color: var(--border-subtle)">
                      <TableCell class="font-medium" style="color: var(--text-primary)">
                        <div class="flex items-center gap-1.5">
                          {{ row.user }}
                          <Lock v-if="row.account_locked" class="h-3.5 w-3.5" style="color: var(--danger)" />
                        </div>
                      </TableCell>
                      <TableCell class="font-mono text-sm" style="color: var(--text-primary)">{{ row.host }}</TableCell>
                      <TableCell>
                        <div class="flex flex-wrap gap-1">
                          <Badge v-for="priv in row.privileges" :key="priv" variant="secondary" class="text-[11px] rounded-full"
                            :class="priv === 'ALL' ? 'pill-active' : priv === 'DELETE' || priv === 'DROP' ? 'badge-status badge-status-danger' : 'pill-default'">
                            {{ privLabelMap[priv] || priv }}
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell class="align-middle">
                        <div class="flex items-center justify-center h-full">
                          <div class="inline-flex items-center rounded-lg p-0.5" style="border: 1px solid var(--border-subtle); background: var(--surface)">
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none action-btn"
                              style="color: var(--text-secondary)"
                              @click="openEditPermDialog(row)"
                            >
                              <Shield class="h-3.5 w-3.5 shrink-0" />
                              授权
                            </button>
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none action-btn"
                              style="color: var(--text-secondary)"
                              @click="openDbGrantDialog(row)"
                            >
                              <DatabaseIcon class="h-3.5 w-3.5 shrink-0" />
                              库权限
                            </button>
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none action-btn"
                              style="color: var(--text-secondary)"
                              @click="openChangePwdDialog(row)"
                            >
                              <KeyRound class="h-3.5 w-3.5 shrink-0" />
                              改密
                            </button>
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none action-btn"
                              style="color: var(--text-secondary)"
                              @click="openChangeHostDialog(row)"
                            >
                              <Globe class="h-3.5 w-3.5 shrink-0" />
                              改主机
                            </button>
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none action-btn"
                              style="color: var(--text-secondary)"
                              @click="toggleUserLock(row)"
                            >
                              <component :is="row.account_locked ? Unlock : LockIcon" class="h-3.5 w-3.5 shrink-0" />
                              {{ row.account_locked ? '解锁' : '锁定' }}
                            </button>
                            <div class="w-px h-4 mx-0.5 shrink-0" style="background: var(--border)"></div>
                            <button
                              class="inline-flex items-center justify-center gap-1.5 px-2.5 py-1.5 rounded-md text-[11px] transition-all whitespace-nowrap leading-none btn-ghost-danger"
                              @click="confirmAction('deleteUser', row)"
                            >
                              <Trash2 class="h-3.5 w-3.5 shrink-0" />
                              删除
                            </button>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                    <TableRow v-if="users.length === 0">
                      <TableCell colspan="4" class="h-32 text-center">
                        <div class="empty-state">
                          <div class="empty-state-icon">
                            <Users class="h-10 w-10" />
                          </div>
                          <div class="empty-state-text">暂无用户</div>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
              <div v-else class="content-card p-8">
                <div class="empty-state">
                  <div class="empty-state-icon">
                    <Users class="h-10 w-10" />
                  </div>
                  <div class="empty-state-text">用户管理功能仅限 root 账户使用</div>
                  <div class="text-[13px] mt-1" style="color: var(--text-tertiary)">当前账号为 {{ currentDb.username }}，请使用 root 账号登录后重试</div>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="flex items-center justify-center py-12">
            <div class="empty-state">
              <div class="empty-state-icon">
                <Settings class="h-12 w-12" />
              </div>
              <div class="empty-state-text">请选择一个数据库实例进行管理</div>
            </div>
          </div>
        </div>

    </div>

    <Dialog v-model:open="showCreateUserDialog">
      <DialogContent class="sm:max-w-[520px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>创建数据库用户</DialogTitle>
        </div>
        <div class="space-y-4">
          <div v-if="createUserResult.show" :class="createUserResult.success ? 'bg-emerald-500/5 border border-emerald-500/20 text-emerald-400' : 'bg-red-500/5 border border-red-500/20 text-red-400'" class="rounded-md p-3 text-sm">
            {{ createUserResult.message }}
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">用户名 <span class="text-destructive">*</span></label>
            <Input v-model="userForm.username" placeholder="输入用户名" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">密码 <span class="text-destructive">*</span></label>
            <Input type="password" v-model="userForm.password" placeholder="输入密码" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">允许主机</label>
            <Input v-model="userForm.host" placeholder="默认: %(任意主机)" />
            <span class="text-xs text-muted-foreground">常用值: %(任意), localhost(本地), 192.168.1.%(指定网段)</span>
          </div>
          <div class="flex flex-col gap-2">
            <label class="text-sm font-medium text-foreground">权限选择</label>
            <div class="flex items-center gap-2 pb-2 border-b border-border">
              <label
                class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium cursor-pointer transition-colors border"
                :class="userForm.privileges.includes('ALL')
                  ? 'bg-primary text-white border-primary'
                  : 'bg-card text-foreground border-border hover:border-primary/50'"
                @click="toggleCreateUserPriv('ALL')"
              >
                <span>全部权限</span>
              </label>
            </div>
            <div class="space-y-3">
              <div>
                <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">数据操作</div>
                <div class="flex flex-wrap gap-2">
                  <label
                    v-for="priv in [{value:'SELECT',label:'查询'},{value:'INSERT',label:'插入'},{value:'UPDATE',label:'更新'},{value:'DELETE',label:'删除'}]"
                    :key="priv.value"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                    :class="userForm.privileges.includes(priv.value)
                      ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                      : 'bg-card text-foreground border-border hover:border-primary/30'"
                    @click="toggleCreateUserPriv(priv.value)"
                  >
                    <span class="w-1.5 h-1.5 rounded-full" :class="userForm.privileges.includes(priv.value) ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                    {{ priv.label }}
                  </label>
                </div>
              </div>
              <div>
                <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">结构管理</div>
                <div class="flex flex-wrap gap-2">
                  <label
                    v-for="priv in [{value:'CREATE',label:'创建'},{value:'DROP',label:'删除结构'}]"
                    :key="priv.value"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                    :class="userForm.privileges.includes(priv.value)
                      ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                      : 'bg-card text-foreground border-border hover:border-primary/30'"
                    @click="toggleCreateUserPriv(priv.value)"
                  >
                    <span class="w-1.5 h-1.5 rounded-full" :class="userForm.privileges.includes(priv.value) ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                    {{ priv.label }}
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 flex-wrap">
          <Button variant="outline" @click="showCreateUserDialog = false; createUserResult.show = false">取消</Button>
          <Button @click="createUser" :disabled="creatingUser" :loading="creatingUser">
            {{ creatingUser ? '创建中...' : '确认创建' }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="confirmState.open">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>{{ confirmState.title }}</DialogTitle>
          <DialogDescription>{{ confirmState.description }}</DialogDescription>
        </div>
        <div class="flex justify-end gap-2 flex-wrap">
          <Button variant="outline" @click="confirmState.open = false">取消</Button>
          <Button :variant="confirmState.variant" @click="handleConfirm">
            {{ confirmState.confirmText }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="editPermState.open">
      <DialogContent class="sm:max-w-[600px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>修改权限 - {{ editPermState.row?.user }}@{{ editPermState.row?.host }}</DialogTitle>
          <DialogDescription>点击小卡片选择权限</DialogDescription>
        </div>
        <div v-if="editPermState.result?.show" :class="editPermState.result.success ? 'bg-emerald-500/5 border border-emerald-500/20 text-emerald-400' : 'bg-red-500/5 border border-red-500/20 text-red-400'" class="rounded-md p-3 text-sm">
          {{ editPermState.result.message }}
        </div>

        <div class="flex flex-col gap-4 py-2">
          <div class="flex items-center gap-2 pb-2 border-b border-border">
            <label
              class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium cursor-pointer transition-colors border"
              :class="editPermState.privileges.includes('ALL')
                ? 'bg-primary text-white border-primary'
                : 'bg-card text-foreground border-border hover:border-primary/50'"
              @click="togglePrivilege('ALL')"
            >
              <span>全部权限</span>
            </label>
            <span class="text-xs text-amber-500">授予所有操作权限</span>
          </div>

          <div class="space-y-4">
            <div>
              <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">数据操作</div>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="priv in privilegeGroups[0].items"
                  :key="priv.value"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                  :class="editPermState.privileges.includes(priv.value)
                    ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                    : 'bg-card text-foreground border-border hover:border-primary/30'"
                  @click="togglePrivilege(priv.value)"
                >
                  <span class="w-1.5 h-1.5 rounded-full" :class="editPermState.privileges.includes(priv.value) ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                  {{ priv.label }}
                </label>
              </div>
            </div>

            <div>
              <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">结构管理</div>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="priv in privilegeGroups[1].items"
                  :key="priv.value"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                  :class="editPermState.privileges.includes(priv.value)
                    ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                    : 'bg-card text-foreground border-border hover:border-primary/30'"
                  @click="togglePrivilege(priv.value)"
                >
                  <span class="w-1.5 h-1.5 rounded-full" :class="editPermState.privileges.includes(priv.value) ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                  {{ priv.label }}
                </label>
              </div>
            </div>

            <div>
              <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">视图与存储过程</div>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="priv in privilegeGroups[2].items"
                  :key="priv.value"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                  :class="editPermState.privileges.includes(priv.value)
                    ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                    : 'bg-card text-foreground border-border hover:border-primary/30'"
                  @click="togglePrivilege(priv.value)"
                >
                  <span class="w-1.5 h-1.5 rounded-full" :class="editPermState.privileges.includes(priv.value) ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                  {{ priv.label }}
                </label>
              </div>
            </div>

            <div>
              <div class="text-xs font-semibold text-muted-foreground uppercase mb-2">其他</div>
              <div class="flex flex-wrap gap-2">
                <label
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] cursor-pointer transition-all border"
                  :class="editPermState.privileges.includes('GRANT OPTION')
                    ? 'bg-primary/10 text-primary border-primary/30 font-medium'
                    : 'bg-card text-foreground border-border hover:border-primary/30'"
                  @click="togglePrivilege('GRANT OPTION')"
                >
                  <span class="w-1.5 h-1.5 rounded-full" :class="editPermState.privileges.includes('GRANT OPTION') ? 'bg-primary' : 'bg-muted-foreground/30'"></span>
                  授权权限
                </label>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 flex-wrap">
          <Button variant="outline" @click="editPermState.open = false">取消</Button>
          <Button variant="primary" @click="submitEditPerm">确认修改</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="dbGrantState.open">
      <DialogContent class="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>数据库权限 - {{ dbGrantState.row?.user }}@{{ dbGrantState.row?.host }}</DialogTitle>
          <DialogDescription>管理该用户在各数据库上的权限</DialogDescription>
        </div>
        <div v-if="dbGrantState.result.show" :class="dbGrantState.result.success ? 'bg-emerald-500/5 border border-emerald-500/20 text-emerald-400' : 'bg-red-500/5 border border-red-500/20 text-red-400'" class="rounded-md p-3 text-sm">
          {{ dbGrantState.result.message }}
        </div>

        <div class="space-y-3">
          <div class="rounded-lg border border-border bg-muted p-3 space-y-3">
            <div class="flex items-center gap-2">
              <div class="flex-1">
                <label class="text-xs text-muted-foreground mb-1 block">数据库</label>
                <Select v-model="dbGrantState.selectedDb" @update:model-value="onDbSelectChange">
                  <SelectTrigger class="h-8 text-[13px]"><SelectValue placeholder="选择数据库" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="dbName in allDbNames" :key="dbName" :value="dbName">
                      <span class="flex items-center gap-1.5">
                        <span>{{ dbName }}</span>
                        <Badge v-if="isDbGranted(dbName)" variant="outline" class="text-[10px] h-[16px] px-1 bg-primary/10 text-primary border-primary/20">已授权</Badge>
                      </span>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div class="flex-1">
                <label class="text-xs text-muted-foreground mb-1 block">权限</label>
                <Select v-model="dbGrantState.selectedPrivs" multiple>
                  <SelectTrigger class="h-8 text-[13px]"><SelectValue placeholder="选择权限" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="priv in privilegeOptions" :key="priv.value" :value="priv.value">{{ priv.label }}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div class="flex justify-end">
              <Button size="sm" variant="primary" class="h-8 text-[13px]" @click="submitDbGrant" :disabled="!dbGrantState.selectedDb || dbGrantState.selectedPrivs.length === 0">
                {{ isDbGranted(dbGrantState.selectedDb) ? '修改权限' : '添加权限' }}
              </Button>
            </div>
          </div>

          <div v-if="dbGrantState.dbGrants.length > 0" class="space-y-2">
            <div class="text-[12px] font-semibold text-muted-foreground uppercase tracking-wider">已有权限</div>
            <div class="space-y-2 max-h-[280px] overflow-y-auto">
              <div v-for="grant in dbGrantState.dbGrants" :key="grant.database" class="flex items-center justify-between rounded-lg border border-border bg-card p-3">
                <div class="flex-1 min-w-0">
                  <div class="text-[13px] font-medium text-foreground">{{ grant.database }}</div>
                  <div class="flex flex-wrap gap-1 mt-1">
                    <Badge v-for="priv in grant.privileges" :key="priv" variant="secondary" class="text-[11px] rounded-full"
                      :class="priv === 'ALL' ? 'pill-active' : priv === 'DELETE' || priv === 'DROP' ? 'badge-status badge-status-danger' : 'pill-default'">
                      {{ privLabelMap[priv] || priv }}
                    </Badge>
                  </div>
                </div>
                <Button variant="ghost" size="xs" class="text-destructive hover:bg-destructive/10 shrink-0 ml-2" @click="removeDbGrant(grant.database)">
                  删除
                </Button>
              </div>
            </div>
          </div>
          <div v-else class="text-center py-6 text-muted-foreground text-sm">
            暂无数据库级别权限
          </div>
        </div>

        <div class="flex justify-end gap-2 mt-1 flex-wrap">
          <Button variant="outline" @click="dbGrantState.open = false">关闭</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="changePwdState.open">
      <DialogContent class="sm:max-w-[420px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>修改密码 - {{ changePwdState.row?.user }}@{{ changePwdState.row?.host }}</DialogTitle>
        </div>
        <div class="space-y-4">
          <div v-if="changePwdState.result.show" :class="changePwdState.result.success ? 'bg-emerald-500/5 border border-emerald-500/20 text-emerald-400' : 'bg-red-500/5 border border-red-500/20 text-red-400'" class="rounded-md p-3 text-sm">
            {{ changePwdState.result.message }}
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">新密码 <span class="text-destructive">*</span></label>
            <Input type="password" v-model="changePwdState.password" placeholder="输入新密码" />
          </div>
        </div>
        <div class="flex justify-end gap-2 flex-wrap">
          <Button variant="outline" @click="changePwdState.open = false">取消</Button>
          <Button @click="submitChangePwd" :disabled="!changePwdState.password">确认修改</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="changeHostState.open">
      <DialogContent class="sm:max-w-[420px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>修改主机 - {{ changeHostState.row?.user }}@{{ changeHostState.row?.host }}</DialogTitle>
        </div>
        <div class="space-y-4">
          <div v-if="changeHostState.result.show" :class="changeHostState.result.success ? 'bg-emerald-500/5 border border-emerald-500/20 text-emerald-400' : 'bg-red-500/5 border border-red-500/20 text-red-400'" class="rounded-md p-3 text-sm">
            {{ changeHostState.result.message }}
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">当前主机</label>
            <Input :model-value="changeHostState.row?.host || ''" disabled class="bg-muted" />
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">新主机 <span class="text-destructive">*</span></label>
            <Input v-model="changeHostState.newHost" placeholder="默认: %(任意主机)" />
            <span class="text-xs text-muted-foreground">常用值: %(任意), localhost(本地), 192.168.1.%(指定网段)</span>
          </div>
          <div class="flex flex-col gap-1.5">
            <label class="text-sm font-medium text-foreground">密码</label>
            <Input type="password" v-model="changeHostState.password" placeholder="不填则生成随机密码" />
          </div>
        </div>
        <div class="flex justify-end gap-2 flex-wrap">
          <Button variant="outline" @click="changeHostState.open = false">取消</Button>
          <Button @click="submitChangeHost" :disabled="!changeHostState.newHost">确认修改</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="saveAndRestartState.open">
      <DialogContent class="sm:max-w-[425px]">
        <div class="flex flex-col gap-y-1.5">
          <DialogTitle>确认操作</DialogTitle>
          <DialogDescription>
            修改配置后需要重启实例才能生效。您要选择哪种操作？
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 flex-wrap mt-2">
          <Button variant="outline" @click="saveAndRestartState.open = false">
            取消
          </Button>
          <Button variant="secondary" @click="handleSaveOnly">
            仅保存
          </Button>
          <Button variant="primary" @click="handleSaveAndRestart">
            保存并重启
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showFolderBrowser">
      <DialogContent class="sm:max-w-[480px]">
        <DialogTitle class="text-[15px] font-semibold">选择目录</DialogTitle>
        <DialogDescription class="text-[13px] text-muted-foreground truncate">{{ browsePath || '/' }}</DialogDescription>
        <div class="border border-border rounded-lg overflow-hidden">
          <div class="max-h-[320px] overflow-y-auto">
            <div
              v-if="browseParent !== '' && browseParent !== browsePath && !browseIsRoot"
              class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-muted border-b border-border"
              @click="navigateFolder(browseParent)"
            >
              <ArrowUp class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <span class="text-[13px] text-muted-foreground">上级目录</span>
            </div>
            <div
              v-for="d in browseDirs"
              :key="d.path"
              class="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-muted border-b border-border last:border-b-0"
              @click="navigateFolder(d.path)"
              @dblclick="selectFolder(d.path)"
            >
              <HardDrive v-if="d.drive" class="h-3.5 w-3.5 text-primary shrink-0" />
              <FolderOpen v-else class="h-3.5 w-3.5 icon-special-color shrink-0" />
              <span class="text-[13px] text-foreground flex-1 truncate">{{ d.name }}</span>
              <ChevronRight class="h-3.5 w-3.5 text-muted-foreground shrink-0" />
            </div>
            <div v-if="browseDirs.length === 0" class="flex items-center justify-center py-8 text-[13px] text-muted-foreground">
              此目录没有子目录
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-2">
          <Button variant="outline" class="h-[32px] text-[13px]" @click="showFolderBrowser = false">取消</Button>
          <Button variant="primary" class="h-[32px] text-[13px]" @click="selectFolder(browsePath)">选择此目录</Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
defineOptions({ name: 'SettingsView' })
import { ref, computed, onMounted, onActivated, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { sourceParam, instanceUid } from '@/lib/instance'
import { writeLog } from '../api/log'
import { useMessage } from '../composables/useMessage'
import StatusDot from './StatusDot.vue'
import {
  RefreshCw, CircleX, FileText,
  Plus, Settings, Users, Database, HardDrive, Info, Palette, Activity,
  FolderOpen, ChevronRight, ArrowUp, Lock,
  KeyRound, Shield, Database as DatabaseIcon, Globe, Lock as LockIcon, Unlock, Trash2
} from 'lucide-vue-next'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/Tabs.vue'
import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { Input } from '@/components/ui/Input.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select.vue'
import { Textarea } from '@/components/ui/Textarea.vue'
import { Switch } from '@/components/ui/Switch.vue'

const completeProgress = inject('completeProgress')
const sidebarCollapsed = inject('sidebarCollapsed', ref(false))

const pageContainer = ref(null)
const detailCard = ref(null)

const savedActiveTab = localStorage.getItem('settingsMainTab')
const activeTab = ref(savedActiveTab || 'instance')
const systemSubTab = ref(localStorage.getItem('settingsSystemSubTab') || 'logs')
const allDatabases = ref([])
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const selectedDbId = ref(null)

const savedInstanceTab = localStorage.getItem('settingsInstanceTab')
const instanceTab = ref(savedInstanceTab || 'local')
let requestId = 0

const store = useAppContext()
const { connectionId, logEnabled, userName, host, port, name } = storeToRefs(store)

const { success, error, warning, info } = useMessage()

const deduplicateInstances = (instances) => {
  const seen = new Set()
  const result = []
  
  for (const inst of instances) {
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
}

const localInstances = computed(() => deduplicateInstances(allDatabases.value.filter(db => !db.isRemote)))
const remoteInstances = computed(() => deduplicateInstances(allDatabases.value.filter(db => db.isRemote)))
const savingConfig = ref(false)
const updatingPort = ref(false)
const creatingUser = ref(false)
const showCreateUserDialog = ref(false)

const saveAndRestartState = ref({
  open: false,
  action: '',
})

const configContent = ref('')
const configFilePath = ref('')
const configSource = ref('sql')
const users = ref([])

const commonConfig = ref({
  maxConnections: 151,
  innodbBufferSize: '128M',
  port: 3306,
  dataDir: '/var/lib/mysql',
  slowQueryTime: 10,
  bindAddress: '0.0.0.0'
})

const redisCommonConfig = ref({
  maxmemory: '',
  maxclients: 10000,
  port: 6379,
  timeout: 0,
  databases: 16,
  bind: '0.0.0.0'
})

const userForm = ref({
  username: '',
  password: '',
  host: '%',
  privileges: ['SELECT']
})

const createUserResult = ref({ show: false, success: false, message: '' })

const compactMode = ref(localStorage.getItem('compactMode') === 'true')
const refreshInterval = ref(localStorage.getItem('refreshInterval') || '10000')
const backupRetentionDays = ref(localStorage.getItem('backupRetentionDays') || '30')
const logStoragePath = ref('./data/logs')
const logRetentionDays = ref('30')
const showFolderBrowser = ref(false)
const browsePath = ref('')
const browseParent = ref('')
const browseDirs = ref([])
const browseIsRoot = ref(false)
const appVersion = ref('')
const serverPort = ref(window.location.port || '80')
const systemHostname = ref('')
const systemOsName = ref('')
const systemUsername = ref('')

const healthConfig = ref({ intervalSec: 30, timeoutSec: 5, alertEnabled: true, enabled: true })
const healthIntervalSec = ref('30')
const healthTimeoutSec = ref('5')
const forceChecking = ref(false)

const toggleLogEnabled = (val) => {
  store.setLogEnabled(val)
  writeLog(`${val ? '启用' : '禁用'}日志中心`)
}

const healthSortedDetails = computed(() => {
  const details = healthStore.allDetails
  return [...details].sort((a, b) => {
    if (a.online !== b.online) return a.online ? -1 : 1
    return (a.name || '').localeCompare(b.name || '')
  })
})

const loadHealthConfig = () => {
  fetch('/api/health/config')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        healthConfig.value = { ...healthConfig.value, ...data.data }
        healthIntervalSec.value = String(data.data.intervalSec || 30)
        healthTimeoutSec.value = String(data.data.timeoutSec || 5)
      }
    })
    .catch(() => {})
}

const saveHealthConfig = () => {
  fetch('/api/health/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(healthConfig.value)
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success('测活配置已保存')
        writeLog(`测活配置已更新: 启用=${healthConfig.value.enabled}, 间隔=${healthConfig.value.intervalSec}s, 超时=${healthConfig.value.timeoutSec}s, 告警=${healthConfig.value.alertEnabled}`)
      } else {
        error(data.msg || '保存失败')
      }
    })
    .catch(() => error('保存失败'))
}

const updateHealthInterval = (val) => {
  healthConfig.value.intervalSec = parseInt(val) || 30
  saveHealthConfig()
}

const updateHealthTimeout = (val) => {
  healthConfig.value.timeoutSec = parseInt(val) || 5
  saveHealthConfig()
}

const forceHealthCheck = () => {
  forceChecking.value = true
  healthStore.forceCheckAll()
    .then(() => {
      success('测活检测完成')
      writeLog('执行手动测活检测')
    })
    .catch(() => error('检测失败'))
    .finally(() => { forceChecking.value = false })
}

const loadSystemInfo = () => {
  fetch('/api/system/info')
    .then(res => res.json())
    .then(data => {
      if (data.hostname) systemHostname.value = data.hostname
      if (data.os) systemOsName.value = data.os
      if (data.version) appVersion.value = 'v' + data.version
      if (data.username) systemUsername.value = data.username
    })
    .catch(() => {})
}
loadSystemInfo()

const loadBackupRetention = () => {
  fetch('/api/backups/retention')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        const days = data.data.days
        if (days !== undefined && days !== null) {
          backupRetentionDays.value = String(days)
        }
      }
    })
    .catch(() => {})
}

const loadLogConfig = () => {
  try {
    const raw = localStorage.getItem('log_config')
    if (raw) {
      const cfg = JSON.parse(raw)
      logStoragePath.value = cfg.path || './data/logs'
      logRetentionDays.value = String(cfg.retentionDays || 30)
    }
  } catch (e) { console.error(e) }
  fetch('/api/log-config')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0 && data.data) {
        if (data.data.path) logStoragePath.value = data.data.path
        if (data.data.retentionDays) logRetentionDays.value = String(data.data.retentionDays)
      }
    })
    .catch(() => {})
}

const saveLogConfig = () => {
  const cfg = {
    path: logStoragePath.value,
    retentionDays: parseInt(logRetentionDays.value) || 30,
    enabled: logEnabled.value,
  }
  try {
    localStorage.setItem('log_config', JSON.stringify(cfg))
  } catch (e) { console.error(e) }
  fetch('/api/log-config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(cfg)
  }).catch(e => console.error(e))
  success('日志配置已保存')
  writeLog(`日志配置已更新: 路径=${cfg.path}, 保留=${cfg.retentionDays}天, 启用=${cfg.enabled}`)
}

const openFolderBrowser = () => {
  browsePath.value = logStoragePath.value || './data/logs'
  loadBrowseDirs(browsePath.value)
  showFolderBrowser.value = true
}

const loadBrowseDirs = (path) => {
  fetch(`/api/fs/browse?path=${encodeURIComponent(path)}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        browsePath.value = data.path
        browseParent.value = data.parent || ''
        browseDirs.value = data.dirs || []
        browseIsRoot.value = data.isRoot === true
      } else {
        error(data.msg || '无法读取目录')
      }
    })
    .catch(() => error('目录浏览请求失败'))
}

const navigateFolder = (path) => {
  loadBrowseDirs(path)
}

const selectFolder = (path) => {
  logStoragePath.value = path
  showFolderBrowser.value = false
}

const saveBackupRetention = (val) => {
  fetch('/api/backups/retention', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ days: val })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success('备份保留策略已保存')
        writeLog(`备份保留策略已更新: ${val === 'never' ? '永不清理' : val + '天'}`)
      } else {
        error(data.msg || '保存失败')
      }
    })
    .catch(() => error('保存失败'))
}

watch(activeTab, (tab) => {
  localStorage.setItem('settingsMainTab', tab)
})

watch(systemSubTab, (tab) => {
  localStorage.setItem('settingsSystemSubTab', tab)
})

watch(compactMode, (val) => {
  localStorage.setItem('compactMode', String(val))
  window.dispatchEvent(new CustomEvent('compact-mode-change', { detail: val }))
}, { immediate: true })

watch(refreshInterval, (val) => {
  localStorage.setItem('refreshInterval', val)
  window.dispatchEvent(new CustomEvent('refresh-interval-change', { detail: val }))
})

watch(backupRetentionDays, (val) => {
  localStorage.setItem('backupRetentionDays', val)
  saveBackupRetention(val)
})

const toggleSidebar = () => {
  window.dispatchEvent(new CustomEvent('toggle-sidebar'))
}

const currentDb = computed(() => allDatabases.value.find(db => db.id === selectedDbId.value))

const dockerInternalPort = computed(() => {
  if (!currentDb.value?.container) return ''
  if (currentDb.value.type === 'mysql') return '3306'
  if (currentDb.value.type === 'redis') return '6379'
  return ''
})

const confirmState = ref({
  open: false,
  title: '',
  description: '',
  variant: 'default',
  confirmText: '确定',
  action: '',
  payload: null,
})

const editPermState = ref({
  open: false,
  privileges: [],
  row: null,
})

const dbGrantState = ref({
  open: false,
  row: null,
  dbGrants: [],
  selectedDb: '',
  selectedPrivs: ['SELECT'],
  result: { show: false, success: false, message: '' }
})

const allDbNames = computed(() => {
  const db = currentDb.value
  if (!db) return []
  return allDatabases.value
    .filter(d => d.id === db.id && d.databases && d.databases.length > 0)
    .flatMap(d => d.databases)
})

const grantedDbMap = computed(() => {
  const map = {}
  for (const g of dbGrantState.value.dbGrants) {
    map[g.database] = g.privileges || []
  }
  return map
})

const isDbGranted = (dbName) => {
  return !!grantedDbMap.value[dbName]
}

const onDbSelectChange = (dbName) => {
  if (!dbName) {
    dbGrantState.value.selectedPrivs = ['SELECT']
    return
  }
  const existingPrivs = grantedDbMap.value[dbName]
  if (existingPrivs && existingPrivs.length > 0) {
    dbGrantState.value.selectedPrivs = [...existingPrivs]
  } else {
    dbGrantState.value.selectedPrivs = ['SELECT']
  }
}

const changePwdState = ref({ open: false, row: null, password: '', result: { show: false, success: false, message: '' } })

const changeHostState = ref({
  open: false,
  row: null,
  newHost: '%',
  password: '',
  result: { show: false, success: false, message: '' }
})

const openDbGrantDialog = (row) => {
  dbGrantState.value = {
    open: true,
    row,
    dbGrants: [],
    selectedDb: '',
    selectedPrivs: ['SELECT'],
    result: { show: false, success: false, message: '' }
  }
  loadDbGrants(row)
}

const loadDbGrants = (row) => {
  const db = currentDb.value
  if (!db) return
  fetch(`/api/mysql/users/db-grant?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`)
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        dbGrantState.value.dbGrants = (data.data || []).map(item => ({
          ...item,
          privileges: item.privileges ? item.privileges.split(',').map(p => p.trim()) : []
        }))
      } else {
        error(data.msg || '加载数据库权限失败')
      }
    })
    .catch(() => error('加载数据库权限失败'))
}

const privilegeGroups = [
  {
    label: '数据操作',
    items: [
      { value: 'SELECT', label: '查询' },
      { value: 'INSERT', label: '插入' },
      { value: 'UPDATE', label: '更新' },
      { value: 'DELETE', label: '删除' },
    ]
  },
  {
    label: '结构管理',
    items: [
      { value: 'CREATE', label: '创建' },
      { value: 'DROP', label: '删除结构' },
      { value: 'ALTER', label: '修改结构' },
      { value: 'INDEX', label: '索引管理' },
    ]
  },
  {
    label: '视图与存储过程',
    items: [
      { value: 'CREATE VIEW', label: '创建视图' },
      { value: 'SHOW VIEW', label: '查看视图' },
      { value: 'CREATE ROUTINE', label: '创建存储过程' },
      { value: 'ALTER ROUTINE', label: '修改存储过程' },
      { value: 'EXECUTE', label: '执行存储过程' },
    ]
  },
  {
    label: '全局权限',
    items: [
      { value: 'ALL', label: '全部权限' },
    ]
  },
]

const privilegeOptions = privilegeGroups.flatMap(g => g.items)

const privLabelMap = Object.fromEntries(privilegeOptions.map(p => [p.value, p.label]))

watch(instanceTab, (tab) => {
  localStorage.setItem('settingsInstanceTab', tab)
})

const smoothScrollTo = (element, targetTop, duration = 800) => {
  const startTop = element.scrollTop
  const distance = targetTop - startTop
  const startTime = performance.now()

  const animateScroll = (currentTime) => {
    const elapsed = currentTime - startTime
    const progress = Math.min(elapsed / duration, 1)
    
    const easeProgress = 1 - Math.pow(1 - progress, 3)
    
    element.scrollTop = startTop + (distance * easeProgress)
    
    if (progress < 1) {
      requestAnimationFrame(animateScroll)
    }
  }
  
  requestAnimationFrame(animateScroll)
}

const smoothScrollToElement = (targetElement, containerElement, duration = 800) => {
  if (!targetElement || !containerElement) return
  
  const containerRect = containerElement.getBoundingClientRect()
  const targetRect = targetElement.getBoundingClientRect()
  const targetTop = containerElement.scrollTop + (targetRect.top - containerRect.top)
  
  smoothScrollTo(containerElement, targetTop, duration)
}

const detailTab = ref(localStorage.getItem('settingsDetailTab') || 'info')

watch(detailTab, (tab) => {
  localStorage.setItem('settingsDetailTab', tab)
  
  if (tab === 'config' && detailCard.value && pageContainer.value) {
    setTimeout(() => {
      smoothScrollToElement(detailCard.value, pageContainer.value, 800)
    }, 50)
  } else if (pageContainer.value) {
    setTimeout(() => {
      smoothScrollTo(pageContainer.value, 0, 800)
    }, 50)
  }
})

const confirmAction = (action, payload = null) => {
  const isRemote = currentDb.value?.isRemote
  const remoteWarning = isRemote ? '（远程实例，操作将发送至远端服务器执行）' : ''
  const configs = {
    restart: {
      title: '重启确认',
      description: currentDb.value?.type === 'redis'
        ? `此操作将重启Redis服务，未持久化的数据可能丢失。是否继续？${remoteWarning}`
        : `确定要重启该数据库实例吗？${remoteWarning}`,
      variant: 'secondary',
      confirmText: '确定重启',
    },
    stop: {
      title: '停止确认',
      description: currentDb.value?.type === 'redis'
        ? `此操作将停止Redis服务，未持久化的数据可能丢失。是否继续？${remoteWarning}`
        : `确定要停止该数据库实例吗？${remoteWarning}`,
      variant: 'destructive',
      confirmText: '确定停止',
    },
    updatePort: {
      title: '修改端口确认',
      description: `确定要将端口修改为 ${currentDb.value?.type === 'redis' ? redisCommonConfig.value.port : commonConfig.value.port} 吗？`,
      variant: 'default',
      confirmText: '确定修改',
    },
    deleteUser: {
      title: '删除用户',
      description: `确定要删除用户 "${payload?.user}" 吗？`,
      variant: 'destructive',
      confirmText: '确定删除',
    },
  }
  const cfg = configs[action]
  if (!cfg) return
  confirmState.value = {
    open: true,
    ...cfg,
    action,
    payload,
  }
}

const handleConfirm = () => {
  const { action, payload } = confirmState.value
  confirmState.value.open = false
  if (action === 'restart') restartDb()
  else if (action === 'stop') stopDb()
  else if (action === 'updatePort') {
    saveAndRestartState.value = {
      open: true,
      action: 'updatePort',
    }
  }
  else if (action === 'deleteUser') deleteUser(payload)
}

const updatePortOnly = async () => {
  const newPortVal = currentDb.value?.type === 'redis' ? redisCommonConfig.value.port : commonConfig.value.port
  if (!newPortVal || newPortVal < 1024 || newPortVal > 65535) {
    warning('端口范围应为 1024-65535')
    return
  }
  if (!currentDb.value || currentDb.value.isRemote) {
    warning('远程实例不支持在线修改端口')
    return
  }
  if (newPortVal === currentDb.value.port) {
    warning('新端口与当前端口相同')
    return
  }
  updatingPort.value = true
  const db = currentDb.value
  let success = false
  await fetch(`/api/mysql/port?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ port: newPortVal })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(data.msg || '端口已修改')
        loadDatabases()
        success = true
      } else {
        error(data.msg || '修改失败')
      }
    })
    .catch(() => error('修改失败'))
    .finally(() => { updatingPort.value = false })
  return success
}

const handleSaveOnly = async () => {
  const { action } = saveAndRestartState.value
  saveAndRestartState.value.open = false

  if (action === 'saveConfig') {
    await saveConfigOnly()
  } else if (action === 'updatePort') {
    await updatePortOnly()
  }
}

const handleSaveAndRestart = async () => {
  const { action } = saveAndRestartState.value
  saveAndRestartState.value.open = false

  if (action === 'saveConfig') {
    const success = await saveConfigOnly()
    if (success) {
      restartDb()
    }
  } else if (action === 'updatePort') {
    const success = await updatePortOnly()
    if (success) {
      restartDb()
    }
  }
}

const openEditPermDialog = (row) => {
  const privs = row.privileges.length > 0 ? [...row.privileges] : []
  editPermState.value = {
    open: true,
    privileges: privs,
    row,
    result: { show: false, success: false, message: '' }
  }
}

const openChangePwdDialog = (row) => {
  changePwdState.value = {
    open: true,
    row,
    password: '',
    result: { show: false, success: false, message: '' }
  }
}

const openChangeHostDialog = (row) => {
  changeHostState.value = {
    open: true,
    row,
    newHost: '%',
    password: '',
    result: { show: false, success: false, message: '' }
  }
}

const submitChangeHost = () => {
  const db = currentDb.value
  const row = changeHostState.value.row
  if (!db || !row) return
  if (!changeHostState.value.newHost) {
    changeHostState.value.result = { show: true, success: false, message: '请输入新主机' }
    return
  }
  changeHostState.value.result = { show: false, success: false, message: '' }
  const body = {
    user: row.user,
    old_host: row.host,
    new_host: changeHostState.value.newHost,
  }
  if (changeHostState.value.password) {
    body.password = changeHostState.value.password
  }
  fetch(`/api/mysql/users/rename?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        changeHostState.value.result = { show: true, success: true, message: data.msg || '主机修改成功' }
        success(data.msg || '主机修改成功')
        writeLog(`修改主机: ${row.user}@${row.host} -> ${row.user}@${changeHostState.value.newHost} (实例: ${db.name})`)
        setTimeout(() => {
          changeHostState.value.open = false
          loadUsers(requestId)
        }, 1500)
      } else {
        const msg = data.msg || '修改失败'
        changeHostState.value.result = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch((err) => {
      const msg = '修改失败: ' + (err.message || err)
      changeHostState.value.result = { show: true, success: false, message: msg }
      error(msg)
    })
}

const submitChangePwd = () => {
  const db = currentDb.value
  const row = changePwdState.value.row
  if (!db || !row) return
  if (!changePwdState.value.password) {
    changePwdState.value.result = { show: true, success: false, message: '请输入新密码' }
    return
  }
  changePwdState.value.result = { show: false, success: false, message: '' }
  fetch(`/api/mysql/users?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password: changePwdState.value.password })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        changePwdState.value.result = { show: true, success: true, message: data.msg || '密码修改成功' }
        success(data.msg || '密码修改成功')
        writeLog(`修改密码: ${row.user}@${row.host} (实例: ${db.name})`)
        setTimeout(() => { changePwdState.value.open = false }, 1500)
      } else {
        const msg = data.msg || '修改失败'
        changePwdState.value.result = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch((err) => {
      const msg = '修改失败: ' + (err.message || err)
      changePwdState.value.result = { show: true, success: false, message: msg }
      error(msg)
    })
}

const togglePrivilege = (priv) => {
  const idx = editPermState.value.privileges.indexOf(priv)
  if (idx >= 0) {
    editPermState.value.privileges.splice(idx, 1)
  } else {
    if (priv === 'ALL') {
      editPermState.value.privileges = ['ALL']
    } else {
      const allIdx = editPermState.value.privileges.indexOf('ALL')
      if (allIdx >= 0) editPermState.value.privileges.splice(allIdx, 1)
      editPermState.value.privileges.push(priv)
    }
  }
}

const submitEditPerm = () => {
  const { privileges, row } = editPermState.value
  const value = privileges.join(', ')
  const db = currentDb.value
  editPermState.value.result = { show: false, success: false, message: '' }
  fetch(`/api/mysql/users/grant?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ privileges: value })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        editPermState.value.result = { show: true, success: true, message: data.msg || '权限修改成功' }
        success(data.msg || '权限修改成功')
        loadUsers(requestId)
        setTimeout(() => { editPermState.value.open = false }, 1500)
      } else {
        const msg = data.msg || '修改失败'
        editPermState.value.result = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch((err) => {
      const msg = '修改失败: ' + (err.message || err)
      editPermState.value.result = { show: true, success: false, message: msg }
      error(msg)
    })
}

const selectInstance = (id) => {
  selectedDbId.value = id
  const db = allDatabases.value.find(d => d.id === id)
  if (db && db.type === 'mysql' && db.username !== 'root' && detailTab.value === 'users') {
    detailTab.value = 'info'
  }
  onInstanceChange()
  if (db) {
    store.setContext({
      connectionId: instanceUid(db),
      userName: db.username || '',
      dbName: '',
      type: db.type,
      host: db.host,
      port: db.port,
      isRemote: db.isRemote || false,
      name: db.name,
    })
  }
}

const loadDatabases = () => {
  const p1 = fetch('/api/databases/db/list/all')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) return data.data || []
      return []
    })
    .catch(() => [])

  const p2 = fetch('/api/remote-servers')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        return (data.data || []).map(srv => ({ ...srv, isRemote: true }))
      }
      return []
    })
    .catch(() => [])

  Promise.all([p1, p2]).then(([local, remote]) => {
    allDatabases.value = [...local, ...remote]
    checkOnlineStatus(allDatabases.value)
    if (store.connectionId && allDatabases.value.find(d => instanceUid(d) === store.connectionId)) {
      selectedDbId.value = store.serverId
      onInstanceChange()
    } else if (allDatabases.value.length > 0 && !selectedDbId.value) {
      selectedDbId.value = allDatabases.value[0].id
      onInstanceChange()
    }
    completeProgress?.()
  })
}

const checkOnlineStatus = async (items) => {
  await healthStore.refreshAll()
}

const onInstanceChange = () => {
  if (!currentDb.value) return
  requestId++
  const rid = requestId
  configContent.value = ''
  configFilePath.value = ''
  configSource.value = 'sql'
  commonConfig.value = {
    maxConnections: 151,
    innodbBufferSize: '128M',
    port: currentDb.value.port || 3306,
    dataDir: '/var/lib/mysql',
    slowQueryTime: 10,
    bindAddress: '0.0.0.0'
  }
  redisCommonConfig.value = {
    maxmemory: '',
    maxclients: 10000,
    port: 6379,
    timeout: 0,
    databases: 16,
    bind: '0.0.0.0'
  }
  users.value = []
  if (currentDb.value.type === 'redis') {
    loadRedisConfig(rid)
  } else {
    loadConfig(rid)
    loadUsers(rid)
  }
}

const loadConfig = (rid) => {
  if (!currentDb.value) return

  const db = currentDb.value
  if (db.type === 'redis') {
    loadRedisConfig(rid)
    return
  }
  fetch(`/api/mysql/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}`)
    .then(res => res.json())
    .then(data => {
      if (rid !== requestId) return
      if (data.code === 0) {
        configContent.value = data.data.content || ''
        configFilePath.value = data.data.filePath || ''
        configSource.value = data.data.source || 'sql'
        parseCommonConfig(configContent.value)
      } else {
        error(data.msg || '读取配置失败')
      }
    })
    .catch(() => {
      if (rid !== requestId) return
      error('读取配置失败')
    })
}

const loadRedisConfig = (rid) => {
  const db = currentDb.value
  if (!db) return
  fetch(`/api/redis/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}&param=*`)
    .then(res => res.json())
    .then(data => {
      if (rid !== requestId) return
      if (data.code === 0) {
        const configs = data.data || {}
        const lines = ['# Redis 配置', `# 实例: ${db.name}`, `# 主机: ${db.host}:${db.port}`, '']
        for (const [key, val] of Object.entries(configs)) {
          lines.push(`${key} ${val}`)
        }
        configContent.value = lines.join('\n')
        redisCommonConfig.value.maxmemory = configs['maxmemory'] || ''
        redisCommonConfig.value.maxclients = parseInt(configs['maxclients']) || 10000
        redisCommonConfig.value.port = parseInt(configs['port']) || 6379
        redisCommonConfig.value.timeout = parseInt(configs['timeout']) || 0
        redisCommonConfig.value.databases = parseInt(configs['databases']) || 16
        redisCommonConfig.value.bind = configs['bind'] || '0.0.0.0'
      } else {
        error(data.msg || '读取配置失败')
      }
    })
    .catch(() => { if (rid === requestId) error('读取配置失败') })
}

const updateRedisConfig = (key) => {
  const db = currentDb.value
  if (!db) return
  const valMap = {
    maxmemory: redisCommonConfig.value.maxmemory,
    maxclients: String(redisCommonConfig.value.maxclients),
    port: String(redisCommonConfig.value.port),
    timeout: String(redisCommonConfig.value.timeout),
    databases: String(redisCommonConfig.value.databases),
    bind: redisCommonConfig.value.bind
  }
  fetch(`/api/redis/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ param: key, value: valMap[key] })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(`${key} 已更新`)
        loadRedisConfig(requestId)
      } else {
        error(data.msg || '更新失败')
      }
    })
    .catch(() => { error('更新失败') })
}

const parseCommonConfig = (content) => {
  const lines = content.split('\n')
  for (const line of lines) {
    const trimmed = line.trim()
    if (trimmed.startsWith('#') || !trimmed.includes('=')) continue
    const [key, ...valueParts] = trimmed.split('=')
    const value = valueParts.join('=').trim()
    const k = key.trim().toLowerCase()
    if (k === 'max_connections') commonConfig.value.maxConnections = parseInt(value) || 151
    else if (k === 'innodb_buffer_pool_size') commonConfig.value.innodbBufferSize = value
    else if (k === 'port') commonConfig.value.port = parseInt(value) || 3306
    else if (k === 'datadir') commonConfig.value.dataDir = value
    else if (k === 'long_query_time') commonConfig.value.slowQueryTime = parseFloat(value) || 10
    else if (k === 'bind_address') commonConfig.value.bindAddress = value
  }
}

const updateConfig = (key) => {
  const db = currentDb.value
  if (!db) return
  const map = {
    max_connections: String(commonConfig.value.maxConnections),
    innodb_buffer_pool_size: commonConfig.value.innodbBufferSize,
    port: String(commonConfig.value.port),
    long_query_time: String(commonConfig.value.slowQueryTime),
    bind_address: commonConfig.value.bindAddress
  }
  const val = map[key]
  if (!val) return

  fetch(`/api/mysql/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ variables: { [key]: val } })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(data.msg || `${key} 已修改`)
        loadConfig(requestId)
      } else {
        error(data.msg || `${key} 修改失败`)
      }
    })
    .catch(() => error(`${key} 修改失败`))
}

const saveConfigOnly = async () => {
  if (!currentDb.value) return
  savingConfig.value = true
  const db = currentDb.value
  if (db.type === 'redis') {
    info('Redis配置修改请使用下方"常用配置项"进行修改')
    savingConfig.value = false
    return
  }

  let success = false
  if (configSource.value === 'file' && configFilePath.value) {
    await fetch(`/api/mysql/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: configContent.value, filePath: configFilePath.value })
    })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success(data.msg || '配置已保存，请重启MySQL使配置生效')
          success = true
        } else {
          error(data.msg || '保存失败')
        }
      })
      .catch(() => error('保存失败'))
  } else {
    const variables = {}
    const map = {
      max_connections: String(commonConfig.value.maxConnections),
      long_query_time: String(commonConfig.value.slowQueryTime),
      bind_address: commonConfig.value.bindAddress
    }
    for (const [k, v] of Object.entries(map)) {
      if (v && v !== '0' && v !== '') {
        variables[k] = v
      }
    }
    if (Object.keys(variables).length === 0) {
      warning('没有可修改的配置项')
      savingConfig.value = false
      return
    }
    await fetch(`/api/mysql/config?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ variables })
    })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success(data.msg || '配置已在线修改')
          loadConfig(requestId)
          success = true
        } else {
          error(data.msg || '修改失败')
        }
      })
      .catch(() => error('修改失败'))
  }
  savingConfig.value = false
  return success
}

const saveConfig = () => {
  if (!currentDb.value) return
  if (currentDb.value.type === 'redis') {
    info('Redis配置修改请使用下方"常用配置项"进行修改')
    return
  }
  saveAndRestartState.value = {
    open: true,
    action: 'saveConfig',
  }
}

const restartDb = () => {
  const db = currentDb.value
  writeLog(`重启实例: ${db.name} (${db.type})`, 'warning')
  if (db.type === 'redis') {
    fetch(`/api/redis/restart?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, { method: 'POST' })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success('Redis重启中...')
          setTimeout(() => checkOnlineStatus([db]), 3000)
        } else {
          error(data.msg || '重启失败')
        }
      })
      .catch(() => error('重启失败'))
  } else {
    fetch(`/api/mysql/restart?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, { method: 'POST' })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success('MySQL重启中...')
          setTimeout(() => checkOnlineStatus([db]), 5000)
        } else {
          error(data.msg || '重启失败')
        }
      })
      .catch(() => error('重启失败'))
  }
}

const stopDb = () => {
  const db = currentDb.value
  writeLog(`停止实例: ${db.name} (${db.type})`, 'warning')
  if (db.type === 'redis') {
    fetch(`/api/redis/stop?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, { method: 'POST' })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success('Redis已停止')
          checkOnlineStatus([db])
        } else {
          error(data.msg || '停止失败')
        }
      })
      .catch(() => error('停止失败'))
  } else {
    fetch(`/api/mysql/stop?server_id=${db.id}&${sourceParam(db.isRemote || false)}`, { method: 'POST' })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0) {
          success('MySQL已停止')
          checkOnlineStatus([db])
        } else {
          error(data.msg || '停止失败')
        }
      })
      .catch(() => error('停止失败'))
  }
}

const deleteUser = (row) => {
  const db = currentDb.value
  writeLog(`删除用户: ${row.user}@${row.host} (实例: ${db.name})`, 'warning')
  fetch(`/api/mysql/users?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success('用户已删除')
        loadUsers(requestId)
      } else {
        error(data.msg || '删除失败')
      }
    })
    .catch(() => error('删除失败'))
}

const toggleUserLock = (row) => {
  const db = currentDb.value
  if (!db) return
  const newLocked = !row.account_locked
  fetch(`/api/mysql/users?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ locked: newLocked })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        success(data.msg || (newLocked ? '用户已锁定' : '用户已解锁'))
        loadUsers(requestId)
      } else {
        error(data.msg || '操作失败')
      }
    })
    .catch(() => error('操作失败'))
}

const toggleCreateUserPriv = (value) => {
  const idx = userForm.value.privileges.indexOf(value)
  if (idx > -1) {
    userForm.value.privileges.splice(idx, 1)
  } else {
    userForm.value.privileges.push(value)
  }
}

const createUser = () => {
  if (!userForm.value.username || !userForm.value.password) {
    createUserResult.value = { show: true, success: false, message: '请填写用户名和密码' }
    return
  }
  const db = currentDb.value
  if (!db) {
    createUserResult.value = { show: true, success: false, message: '未选择数据库实例' }
    return
  }
  creatingUser.value = true
  createUserResult.value = { show: false, success: false, message: '' }
  const serverId = db.id
  const isRemote = db.isRemote || false
  fetch(`/api/mysql/users?server_id=${serverId}&${sourceParam(isRemote)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: userForm.value.username,
      password: userForm.value.password,
      host: userForm.value.host || '%',
      privileges: userForm.value.privileges.join(', ')
    })
  })
    .then(res => {
      if (!res.ok) {
        return res.text().then(text => {
          try { return JSON.parse(text) } catch { return { code: res.status, msg: text || `HTTP ${res.status}` } }
        })
      }
      return res.json()
    })
    .then(data => {
      console.log('创建用户响应:', data)
      if (data && data.code === 0) {
        createUserResult.value = { show: true, success: true, message: data.msg || `用户 ${userForm.value.username}@${userForm.value.host || '%'} 创建成功` }
        success(data.msg || `用户 ${userForm.value.username}@${userForm.value.host || '%'} 创建成功`)
        writeLog(`创建用户: ${userForm.value.username}@${userForm.value.host || '%'} (实例: ${db.name})`)
        setTimeout(() => {
          showCreateUserDialog.value = false
          createUserResult.value = { show: false, success: false, message: '' }
          userForm.value = {
            username: '',
            password: '',
            host: '%',
            privileges: ['SELECT']
          }
          loadUsers(requestId)
        }, 1500)
      } else {
        const msg = (data && data.msg) || '创建失败'
        console.error('创建用户失败:', msg, data)
        createUserResult.value = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch(err => {
      console.error('创建用户请求异常:', err)
      const msg = '创建失败: ' + (err.message || err)
      createUserResult.value = { show: true, success: false, message: msg }
      error(msg)
    })
    .finally(() => { creatingUser.value = false })
}

const submitDbGrant = () => {
  const { row, selectedDb, selectedPrivs } = dbGrantState.value
  const db = currentDb.value
  if (!db || !row || !selectedDb || selectedPrivs.length === 0) return
  dbGrantState.value.result = { show: false, success: false, message: '' }
  fetch(`/api/mysql/users/db-grant?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ database: selectedDb, privileges: selectedPrivs.join(', ') })
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        dbGrantState.value.result = { show: true, success: true, message: data.msg || '权限设置成功' }
        success(data.msg || '权限设置成功')
        loadDbGrants(row)
        dbGrantState.value.selectedDb = ''
        dbGrantState.value.selectedPrivs = ['SELECT']
      } else {
        const msg = data.msg || '设置失败'
        dbGrantState.value.result = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch((err) => {
      const msg = '设置失败: ' + (err.message || err)
      dbGrantState.value.result = { show: true, success: false, message: msg }
      error(msg)
    })
}

const removeDbGrant = (database) => {
  const { row } = dbGrantState.value
  const db = currentDb.value
  if (!db || !row || !database) return
  dbGrantState.value.result = { show: false, success: false, message: '' }
  fetch(`/api/mysql/users/db-grant?server_id=${db.id}&${sourceParam(db.isRemote || false)}&user=${encodeURIComponent(row.user)}&host=${encodeURIComponent(row.host)}&database=${encodeURIComponent(database)}`, {
    method: 'DELETE'
  })
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        dbGrantState.value.result = { show: true, success: true, message: data.msg || '权限已删除' }
        success(data.msg || '权限已删除')
        loadDbGrants(row)
      } else {
        const msg = data.msg || '删除失败'
        dbGrantState.value.result = { show: true, success: false, message: msg }
        error(msg)
      }
    })
    .catch((err) => {
      const msg = '删除失败: ' + (err.message || err)
      dbGrantState.value.result = { show: true, success: false, message: msg }
      error(msg)
    })
}

const loadUsers = (rid) => {
  const db = currentDb.value
  if (!db || db.type !== 'mysql') return
  fetch(`/api/mysql/users?server_id=${db.id}&${sourceParam(db.isRemote || false)}`)
    .then(res => res.json())
    .then(data => {
      if (rid !== requestId) return
      if (data.code === 0) {
        users.value = data.data.map(item => ({
          ...item,
          privileges: item.privileges ? item.privileges.split(',').map(p => p.trim()) : []
        }))
      }
    })
    .catch(() => {})
}

onMounted(() => {
  loadDatabases()
  loadBackupRetention()
  loadHealthConfig()
  loadLogConfig()
})

onActivated(() => {
  loadDatabases()
})
</script>

<style scoped>
.action-btn:hover {
  background: color-mix(in srgb, var(--text-primary) 10%, transparent);
  color: var(--text-primary);
}
</style>