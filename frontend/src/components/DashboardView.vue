<template>
  <div class="h-full overflow-y-auto bg-muted flex flex-col page-padding">
    <div class="flex items-center gap-2 flex-wrap section-gap shrink-0">
      <div class="flex gap-1.5 flex-wrap flex-1">
        <div
          v-for="db in sortedDatabases"
          :key="instanceUid(db)"
          :class="[
            'flex items-center gap-1.5 px-3 py-1.5 rounded-full border cursor-pointer transition-all duration-200 text-xs',
            selectedUid === instanceUid(db)
              ? 'border-primary/40 bg-primary/8 shadow-sm shadow-primary/10'
              : 'border-border bg-white hover:border-primary/30 hover:shadow-sm',
          ]"
          @click="selectInstance(db)"
        >
          <StatusDot :status="selectedUid === instanceUid(db) ? 'selected' : (onlineStatus[instanceUid(db)] !== false ? 'online' : 'offline')" size="xs" />
          <span class="font-medium text-foreground">{{ db.name }}</span>
          <span class="text-muted-foreground text-[11px] font-mono-data">{{ db.host }}:{{ db.port }}</span>
        </div>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <span
          :class="[
            'flex items-center gap-1.5 text-xs cursor-pointer select-none px-2 py-1 rounded-md transition-colors duration-200',
            polling ? 'text-[#16a34a] hover:bg-[#16a34a]/10' : 'text-muted-foreground hover:bg-muted',
          ]"
          @click="togglePolling"
        >
          <span v-if="polling" class="w-1.5 h-1.5 rounded-full bg-[#16a34a] animate-pulse" />
          <span v-else class="w-1.5 h-1.5 rounded-sm bg-muted-foreground" />
          {{ polling ? `自动刷新 ${countdown}s` : '已暂停 · 点击恢复' }}
        </span>
        <div class="flex items-center gap-1.5">
          <Button v-for="r in timeRanges" :key="r.value" variant="ghost" size="sm"
            :class="[timeRange === r.value ? 'bg-cta text-cta-foreground hover:bg-cta/90' : 'text-secondary-foreground hover:bg-muted', 'h-[28px] text-[12px]']"
            @click="timeRange = r.value">
            {{ r.label }}
          </Button>
          <Loader2 v-if="reloadingHistory" :size="14" class="animate-spin text-primary ml-1" />
        </div>
      </div>
    </div>

    <div v-if="!metrics && !loadError" class="empty-state">
      <div class="flex flex-col items-center gap-3 text-muted-foreground">
        <Gauge :size="56" class="empty-state-icon" />
        <span class="empty-state-text">选择一个数据库实例查看实时指标</span>
      </div>
    </div>

    <div v-if="loadError" class="flex items-center gap-2 px-3 py-2 mb-2.5 rounded-lg bg-red-50 border border-red-200 text-red-500 text-xs shrink-0">
      <CircleX class="h-4 w-4 shrink-0" />
      <span>{{ loadError }}</span>
      <Button size="sm" variant="ghost" class="ml-auto text-red-500 hover:text-red-600 h-auto px-2 py-0.5 text-xs" @click="loadError = ''; fetchMetrics()">重试</Button>
    </div>

    <div v-if="switching" class="flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground text-sm">
      <Loader2 :size="32" class="animate-spin text-primary" />
      <span>正在加载 {{ switchingName }} 的指标...</span>
    </div>

    <template v-if="metrics">
      <div v-if="metrics.online === false" class="flex items-center gap-2 px-3 py-2 mb-2.5 rounded-lg bg-orange-50 border border-orange-200 text-orange-600 text-xs shrink-0">
        <CircleX class="h-4 w-4 shrink-0" />
        <span>该数据库实例已离线，图表显示历史数据</span>
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-4 grid-gap section-gap shrink-0 fade-slide-in">
        <div class="content-card-interactive flex items-center gap-3 stat-padding shadow-sm hover:shadow-md transition-shadow duration-200">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-primary/10 text-primary">
            <Monitor class="h-4 w-4" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-muted-foreground mb-0.5">当前实例</div>
            <div class="text-sm font-semibold text-foreground truncate">{{ instanceName }}</div>
            <div class="text-[11px] text-muted-foreground font-mono-data">{{ metrics.host }}:{{ metrics.port }}</div>
          </div>
        </div>

        <div class="content-card-interactive flex items-center gap-3 stat-padding shadow-sm hover:shadow-md transition-shadow duration-200">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-[#16a34a]/10 text-[#16a34a]">
            <Clock class="h-4 w-4" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-muted-foreground mb-0.5">运行时间</div>
            <div class="text-sm font-semibold text-foreground truncate">{{ metrics.uptime_display || '-' }}</div>
          </div>
        </div>

        <div
          :class="[
            'content-card-interactive flex items-center gap-3 stat-padding shadow-sm transition-shadow duration-200 cursor-pointer',
            connStatusClass === 'danger' ? 'border-red-200 bg-red-50 hover:shadow-md' :
            connStatusClass === 'warn' ? 'border-amber-200 bg-amber-50 hover:shadow-md' :
            'border-border hover:shadow-md',
          ]"
          @click="showProcessPanel = !showProcessPanel"
        >
          <div :class="[
            'w-9 h-9 rounded-lg flex items-center justify-center shrink-0',
            connStatusClass === 'danger' ? 'bg-red-500/10 text-red-500' :
            connStatusClass === 'warn' ? 'bg-amber-500/10 text-amber-500' :
            'bg-[#16a34a]/10 text-[#16a34a]',
          ]">
            <Link class="h-4 w-4" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-muted-foreground mb-0.5">
              当前连接 <span class="text-[10px] text-muted-foreground ml-0.5">{{ showProcessPanel ? '▲' : '▼' }}</span>
            </div>
            <div class="text-sm font-semibold text-foreground truncate font-mono-data">{{ connCurrent }} / {{ metrics.max_connections || '-' }}</div>
            <div class="flex items-center gap-2 mt-1">
              <div class="flex-1 h-1 rounded-full bg-muted overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500"
                  :class="connStatusClass === 'danger' ? 'bg-red-500' : connStatusClass === 'warn' ? 'bg-amber-500' : 'bg-[#16a34a]'"
                  :style="{ width: connUsageNum + '%' }"
                />
              </div>
              <span class="text-[11px] text-muted-foreground shrink-0 font-mono-data leading-[16px]">{{ metrics.connection_usage }}%</span>
            </div>
          </div>
        </div>

        <div class="content-card-interactive flex items-center gap-3 stat-padding shadow-sm hover:shadow-md transition-shadow duration-200">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-amber-500/10 text-amber-500">
            <TrendingUp class="h-4 w-4" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[11px] text-muted-foreground mb-0.5">{{ metrics.type === 'redis' ? '每秒操作' : 'QPS' }}</div>
            <div class="text-sm font-semibold text-foreground truncate font-mono-data">{{ metrics.qps || metrics.ops_per_sec || '-' }}</div>
          </div>
        </div>
      </div>



      <div v-if="showProcessPanel && metrics.type === 'mysql' && metrics.processlist" class="section-gap shrink-0">
        <div class="content-card stat-padding shadow-sm">
          <div class="flex items-center gap-2 text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">
            当前连接
            <span class="text-[11px] font-normal text-muted-foreground">{{ metrics.processlist.length }} 个</span>
          </div>
          <Table>
            <TableHeader>
              <TableRow class="hover:bg-transparent border-b border-[#F0F0F0]">
                <TableHead class="text-[11px] text-muted-foreground font-normal">用户</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal">来源</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal">数据库</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal">命令</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal">时长</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal">状态</TableHead>
                <TableHead class="text-[11px] text-muted-foreground font-normal max-w-[200px]">信息</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="p in metrics.processlist" :key="p.id" :class="p.time > 10 ? 'bg-red-50' : 'hover:bg-muted'" class="border-b border-[#F0F0F0]">
                <TableCell>
                  <Badge variant="secondary" class="text-[11px] font-medium bg-muted text-foreground">{{ p.user }}</Badge>
                </TableCell>
                <TableCell class="font-mono-data text-[11px] text-secondary-foreground">{{ p.host }}</TableCell>
                <TableCell class="text-xs text-foreground">{{ p.db || '-' }}</TableCell>
                <TableCell>
                  <Badge variant="outline" class="text-[10px] font-semibold bg-[#16a34a]/10 text-[#16a34a] border-[#16a34a]/30 rounded-full">{{ p.command }}</Badge>
                </TableCell>
                <TableCell :class="['font-mono-data text-[11px] text-foreground', p.time > 10 && 'text-red-500 font-semibold']">{{ p.time }}s</TableCell>
                <TableCell class="text-xs text-foreground max-w-[100px] truncate">{{ p.state || '-' }}</TableCell>
                <TableCell class="text-xs text-foreground max-w-[200px] truncate" :title="p.info">{{ p.info || '-' }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 grid-gap section-gap shrink-0">
        <div class="content-card stat-padding shadow-sm">
          <div class="flex justify-between items-center mb-2">
            <span class="text-xs font-semibold text-foreground">网络</span>
          </div>
          <v-chart style="height: 180px; width: 100%" :option="netChartOption" autoresize />
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="flex justify-between items-center mb-2">
            <span class="text-xs font-semibold text-foreground">{{ metrics.type === 'redis' ? 'Ops/sec 趋势' : 'QPS/TPS 双驱趋势' }}</span>
          </div>
          <v-chart style="height: 180px; width: 100%" :option="qpsTpsChartOption" autoresize />
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 grid-gap shrink-0" v-if="metrics.type === 'mysql'">
        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">线程</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">运行中</span>
              <span class="text-xs font-semibold text-primary font-mono-data">{{ metrics.threads_running }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">已连接</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.threads_connected }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">缓存中</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.threads_cached }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">历史最大连接</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.max_used_connection }}</span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">连接数趋势</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">当前</span>
              <span class="text-xs font-semibold text-primary font-mono-data">{{ connCurrent }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">最大</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.max_used_connection }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">使用率</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.connection_usage }}%</span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="flex justify-between items-center mb-2 pb-1.5 border-b border-[#F0F0F0]"><span class="text-xs font-semibold text-foreground">查询分布</span></div>
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-2">
              <span class="w-[52px] text-[11px] font-semibold text-muted-foreground text-right shrink-0">SELECT</span>
              <div class="flex-1 h-[18px] bg-muted rounded overflow-hidden">
                <div class="h-full rounded bg-gradient-to-r from-[#4facfe] to-[#00f2fe] transition-[width] duration-500" :style="{ width: queryBarWidth(rangeStatsAvailable ? rangeStats.deltaComSelect : metrics.com_select) }" />
              </div>
              <span class="w-[52px] text-xs font-mono-data text-foreground text-right shrink-0">{{ formatNum(rangeStatsAvailable ? rangeStats.deltaComSelect : metrics.com_select) }}</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="w-[52px] text-[11px] font-semibold text-muted-foreground text-right shrink-0">INSERT</span>
              <div class="flex-1 h-[18px] bg-muted rounded overflow-hidden">
                <div class="h-full rounded bg-gradient-to-r from-[#16a34a] to-[#4ade80] transition-[width] duration-500" :style="{ width: queryBarWidth(rangeStatsAvailable ? rangeStats.deltaComInsert : metrics.com_insert) }" />
              </div>
              <span class="w-[52px] text-xs font-mono-data text-foreground text-right shrink-0">{{ formatNum(rangeStatsAvailable ? rangeStats.deltaComInsert : metrics.com_insert) }}</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="w-[52px] text-[11px] font-semibold text-muted-foreground text-right shrink-0">UPDATE</span>
              <div class="flex-1 h-[18px] bg-muted rounded overflow-hidden">
                <div class="h-full rounded bg-gradient-to-r from-[#e6a23c] to-[#f59e0b] transition-[width] duration-500" :style="{ width: queryBarWidth(rangeStatsAvailable ? rangeStats.deltaComUpdate : metrics.com_update) }" />
              </div>
              <span class="w-[52px] text-xs font-mono-data text-foreground text-right shrink-0">{{ formatNum(rangeStatsAvailable ? rangeStats.deltaComUpdate : metrics.com_update) }}</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="w-[52px] text-[11px] font-semibold text-muted-foreground text-right shrink-0">DELETE</span>
              <div class="flex-1 h-[18px] bg-muted rounded overflow-hidden">
                <div class="h-full rounded bg-gradient-to-r from-[#ef4444] to-[#f97316] transition-[width] duration-500" :style="{ width: queryBarWidth(rangeStatsAvailable ? rangeStats.deltaComDelete : metrics.com_delete) }" />
              </div>
              <span class="w-[52px] text-xs font-mono-data text-foreground text-right shrink-0">{{ formatNum(rangeStatsAvailable ? rangeStats.deltaComDelete : metrics.com_delete) }}</span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">InnoDB 缓冲池</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">命中率</span>
              <span :class="['text-xs font-semibold font-mono-data', parseFloat(metrics.innodb_buffer_pool_hit_rate) > 95 ? 'text-[#16a34a]' : 'text-amber-500']">
                {{ metrics.innodb_buffer_pool_hit_rate }}%
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">空闲页</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.innodb_buffer_pool_pages_free }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">总页数</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.innodb_buffer_pool_pages_total }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">脏页</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.innodb_buffer_pool_pages_dirty }}</span>
            </div>
          </div>
          <div class="mt-2 px-1">
            <div class="h-2 rounded-full bg-muted overflow-hidden">
              <div
                class="h-full rounded-full transition-all"
                :class="parseFloat(metrics.innodb_buffer_pool_hit_rate) > 95 ? 'bg-[#16a34a]' : 'bg-amber-500'"
                :style="{ width: Math.min(parseFloat(metrics.innodb_buffer_pool_hit_rate) || 0, 100) + '%' }"
              />
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">效率指标</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">慢查询</span>
              <span :class="['text-xs font-semibold font-mono-data', parseInt(metrics.slow_queries) > 0 ? 'text-amber-500' : 'text-foreground']">
                {{ metrics.slow_queries }}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">表锁等待</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.table_locks_waited }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">磁盘临时表</span>
              <span :class="['text-xs font-semibold font-mono-data', parseInt(metrics.tmp_table_disk) > 500 ? 'text-orange-500' : 'text-foreground']">{{ metrics.tmp_table_disk }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">中断连接</span>
              <span :class="['text-xs font-semibold font-mono-data', parseInt(metrics.aborted_connects) > 10 ? 'text-orange-500' : 'text-foreground']">{{ metrics.aborted_connects }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 grid-gap shrink-0" v-if="metrics.type === 'redis'">
        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">内存</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">当前使用</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.used_memory }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">峰值</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.used_memory_peak }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">碎片率</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.mem_fragmentation }}</span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">客户端</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">已连接</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.connected_clients }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">阻塞中</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.blocked_clients }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">最大限制</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.maxclients }}</span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">键空间</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">命中</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.keyspace_hits }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">未命中</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.keyspace_misses }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">命中率</span>
              <span :class="['text-xs font-semibold font-mono-data', parseFloat(metrics.hit_rate) > 90 ? 'text-[#16a34a]' : 'text-amber-500']">
                {{ metrics.hit_rate }}%
              </span>
            </div>
          </div>
        </div>

        <div class="content-card stat-padding shadow-sm">
          <div class="text-xs font-semibold text-foreground mb-2 pb-1.5 border-b border-[#F0F0F0]">持久化</div>
          <div class="flex flex-col gap-1.5">
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">AOF</span>
              <span :class="['text-xs font-semibold font-mono-data', metrics.aof_enabled === '1' ? 'text-[#16a34a]' : 'text-foreground']">
                {{ metrics.aof_enabled === '1' ? '已启用' : '未启用' }}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs text-muted-foreground">RDB 变更数</span>
              <span class="text-xs font-semibold text-foreground font-mono-data">{{ metrics.rdb_changes }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
defineOptions({ name: 'DashboardView' })
import { ref, computed, onMounted, onActivated, onDeactivated, onUnmounted, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { sourceParam, instanceUid } from '@/lib/instance'
import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { RefreshCw, Gauge, CircleX, Loader2, Monitor, Clock, Link, TrendingUp } from 'lucide-vue-next'
import StatusDot from './StatusDot.vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'

const completeProgress = inject('completeProgress')

use([CanvasRenderer, LineChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const databases = ref([])
const selectedUid = ref(null)
const selectedDb = computed(() => databases.value.find(d => instanceUid(d) === selectedUid.value))
const selectedId = computed(() => selectedDb.value?.id ?? null)
const metrics = ref(null)
const loading = ref(false)
const reloadingHistory = ref(false)
const loadError = ref('')
const switching = ref(false)
const switchingName = ref('')
const showProcessPanel = ref(false)
const polling = ref(true)
const countdown = ref(3)
const connHistory = ref([])
const qpsHistory = ref([])
const tpsHistory = ref([])
const healthStore = useHealthStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const prevRawCounters = ref({ questions: 0, writes: 0, timestamp: 0 })
const timeRanges = [
  { value: 900, label: '15分钟' },
  { value: 3600, label: '1小时' },
  { value: 21600, label: '6小时' },
  { value: 86400, label: '24小时' },
  { value: 604800, label: '7天' },
]
const TIME_RANGE_KEY = 'dashboard_time_range'
const timeRange = ref(parseInt(localStorage.getItem(TIME_RANGE_KEY)) || 3600)
const MAX_HISTORY_POINTS = 15000
let pollTimer = null
let countdownTimer = null
let metricsRequestId = 0

const getRefreshIntervalMs = () => {
  const val = localStorage.getItem('refreshInterval')
  if (!val || val === 'off') return 0
  const ms = parseInt(val)
  return ms > 0 ? ms : 0
}

const store = useAppContext()
const { connectionId } = storeToRefs(store)
const { favorites } = storeToRefs(store)

const instanceName = computed(() => {
  if (!selectedUid.value) return ''
  return selectedDb.value ? selectedDb.value.name : ''
})

const sortedDatabases = computed(() => {
  return [...databases.value].sort((a, b) => {
    const aOnline = onlineStatus.value[instanceUid(a)] !== false ? 1 : 0
    const bOnline = onlineStatus.value[instanceUid(b)] !== false ? 1 : 0
    if (aOnline !== bOnline) return bOnline - aOnline
    const favA = favorites.value.find(f => f.id === a.id)
    const favB = favorites.value.find(f => f.id === b.id)
    return (favB?.count || 0) - (favA?.count || 0)
  })
})

watch(onlineStatus, (status) => {
  if (!selectedUid.value) return
  if (status[selectedUid.value] === false) {
    const firstOnline = sortedDatabases.value.find(d => instanceUid(d) !== selectedUid.value && (status[instanceUid(d)] !== false))
    if (firstOnline) {
      selectInstance(firstOnline)
    }
  }
}, { deep: true })

const formatNum = (val) => {
  if (!val && val !== 0) return '-'
  const n = parseInt(val)
  if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return String(n)
}

const queryBarWidth = (val) => {
  if (!val) return '0%'
  const values = [
    parseInt(metrics.value?.com_select) || 0,
    parseInt(metrics.value?.com_insert) || 0,
    parseInt(metrics.value?.com_update) || 0,
    parseInt(metrics.value?.com_delete) || 0
  ]
  const max = Math.max(...values, 1)
  return ((parseInt(val) / max) * 100).toFixed(0) + '%'
}

const connUsageNum = computed(() => {
  return Math.min(parseFloat(metrics.value?.connection_usage) || 0, 100)
})

const connStatusClass = computed(() => {
  const u = connUsageNum.value
  if (u > 80) return 'danger'
  if (u > 60) return 'warn'
  return 'normal'
})

const connCurrent = computed(() => {
  if (!metrics.value) return '-'
  if (metrics.value.type === 'redis') return metrics.value.connected_clients
  return metrics.value.threads_connected
})

const latestDataTime = computed(() => {
  const connLast = connHistory.value.length > 0 ? connHistory.value[connHistory.value.length - 1].timestamp : 0
  const qpsLast = qpsHistory.value.length > 0 ? qpsHistory.value[qpsHistory.value.length - 1].timestamp : 0
  const base = metrics.value?.traffic?.echarts
  let netLast = 0
  if (base && base.network && base.network.series) {
    const netIn = base.network.series[0]?.data || []
    const netOut = base.network.series[1]?.data || []
    if (netIn.length > 0) netLast = Math.max(netLast, netIn[netIn.length - 1][0])
    if (netOut.length > 0) netLast = Math.max(netLast, netOut[netOut.length - 1][0])
  }
  return Math.max(connLast, qpsLast, netLast) || Date.now()
})

const rangeStats = computed(() => {
  return metrics.value?.rangeStats || {}
})

const rangeStatsAvailable = computed(() => {
  return rangeStats.value.available === true
})

const timeRangeLabel = computed(() => {
  const r = timeRanges.find(r => r.value === timeRange.value)
  return r ? r.label : ''
})

const formatBytes = (bytes) => {
  const v = Number(bytes || 0)
  if (!Number.isFinite(v) || v <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exp = Math.max(0, Math.min(units.length - 1, Math.floor(Math.log(v) / Math.log(1024))))
  const n = v / Math.pow(1024, exp)
  return n.toFixed(exp === 0 ? 0 : n >= 100 ? 0 : 1) + ' ' + units[exp]
}



const xAxisOption = computed(() => {
  const range = timeRange.value
  const latest = latestDataTime.value
  let formatter = '{HH}:{mm}'
  if (range >= 86400) {
    formatter = '{MM}-{dd}'
  } else if (range >= 21600) {
    formatter = '{MM}-{dd}'
  }
  return {
    type: 'time',
    min: latest - range * 1000,
    max: latest,
    boundaryGap: false,
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: {
      color: '#8C8C8C',
      fontSize: 9,
      formatter: formatter,
      hideOverlap: true,
      interval: 'auto',
    },
    splitLine: { show: false },
  }
})
const yAxisTemplate = {
  type: 'value',
  axisLine: { show: false },
  axisTick: { show: false },
  axisLabel: { color: '#8C8C8C', fontSize: 9 },
  splitLine: { lineStyle: { color: '#F5F5F5' } },
  minInterval: 1,
}
const gridTemplate = { left: '8px', right: '8px', bottom: '8px', top: '24px', containLabel: true }
const tooltipTemplate = {
  trigger: 'axis',
  axisPointer: { type: 'cross' },
  backgroundColor: 'rgba(0, 0, 0, 0.85)',
  borderColor: 'rgba(255,255,255,0.1)',
  borderWidth: 1,
  textStyle: { color: '#fff', fontSize: 11 },
}

const connChartOption = computed(() => {
  const data = connHistory.value.map(d => [d.timestamp, d.value])
  return {
    animation: true,
    animationDuration: 420,
    animationDurationUpdate: 500,
    animationEasingUpdate: 'cubicInOut',
    tooltip: tooltipTemplate,
    grid: gridTemplate,
    xAxis: xAxisOption.value,
    yAxis: yAxisTemplate,
    series: [{
      name: '连接数', type: 'line', smooth: true, symbol: 'none',
      lineStyle: { color: '#4facfe', width: 2 }, itemStyle: { color: '#4facfe' },
      areaStyle: { opacity: 0.05, color: '#4facfe' },
      emphasis: { focus: 'series' },
      connectNulls: true,
      data: data.length > 0 ? data : [],
    }],
  }
})

const netChartOption = computed(() => {
  const base = metrics.value?.traffic?.echarts
  if (!base || !base.network) {
    return {
      animation: true, animationDuration: 420, animationDurationUpdate: 500, animationEasingUpdate: 'cubicInOut',
      tooltip: tooltipTemplate, grid: gridTemplate, xAxis: xAxisOption.value, yAxis: { type: 'value', axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: '#8C8C8C', fontSize: 9, formatter: (v) => formatBytes(v) + '/s' }, splitLine: { lineStyle: { color: '#F5F5F5' } }, min: 0 },
      series: [],
    }
  }
  const netData = base.network
  const inData = (netData.series && netData.series[0] && netData.series[0].data) || []
  const outData = (netData.series && netData.series[1] && netData.series[1].data) || []
  
  return {
    animation: true,
    animationDuration: 420,
    animationDurationUpdate: 500,
    animationEasingUpdate: 'cubicInOut',
    tooltip: {
      ...tooltipTemplate,
      formatter: (params) => {
        const arr = Array.isArray(params) ? params : [params]
        const t = arr[0]?.axisValueLabel ?? ''
        const lines = [t]
        for (const p of arr) {
          const name = String(p?.seriesName ?? '')
          const value = Array.isArray(p?.data) ? p.data[1] : p?.value
          lines.push(`${p?.marker ?? ''} ${name}: ${formatBytes(Number(value))}/s`)
        }
        return lines.join('<br/>')
      },
    },
    grid: gridTemplate,
    xAxis: xAxisOption.value,
    yAxis: { 
      type: 'value', 
      axisLine: { show: false }, 
      axisTick: { show: false }, 
      axisLabel: { color: '#8C8C8C', fontSize: 9, formatter: (v) => formatBytes(v) + '/s' }, 
      splitLine: { lineStyle: { color: '#F5F5F5' } }, 
      min: 0,
    },
    series: [
      {
        name: '接收', type: 'line', smooth: true, symbol: 'none',
        lineStyle: { color: '#4facfe', width: 2 }, itemStyle: { color: '#4facfe' },
        areaStyle: { opacity: 0.05, color: '#4facfe' },
        emphasis: { focus: 'series' }, connectNulls: true, data: inData,
      },
      {
        name: '发送', type: 'line', smooth: true, symbol: 'none',
        lineStyle: { color: '#16a34a', width: 2 }, itemStyle: { color: '#16a34a' },
        areaStyle: { opacity: 0.05, color: '#16a34a' },
        emphasis: { focus: 'series' }, connectNulls: true, data: outData,
      },
    ],
    legend: { data: ['接收', '发送'], right: '24px', top: '0px', itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 10, color: '#8C8C8C' } },
  }
})

const qpsTpsChartOption = computed(() => {
  const mk = (arr) => arr.map(d => [d.timestamp, d.value])
  const series = [{
    name: 'QPS', type: 'line', smooth: true, symbol: 'none',
    lineStyle: { color: '#4facfe', width: 2 }, itemStyle: { color: '#4facfe' },
    areaStyle: { opacity: 0.05, color: '#4facfe' },
    emphasis: { focus: 'series' },
    connectNulls: true,
    data: mk(qpsHistory.value),
  }]
  const legendData = ['QPS']
  if (metrics.value?.type === 'mysql') {
    series.push({
      name: 'TPS', type: 'line', smooth: true, symbol: 'none',
      lineStyle: { color: '#16a34a', width: 2, type: 'dashed' }, itemStyle: { color: '#16a34a' },
      areaStyle: { opacity: 0.05, color: '#16a34a' },
      emphasis: { focus: 'series' },
      connectNulls: true,
      data: mk(tpsHistory.value),
    })
    legendData.push('TPS')
  }
  return {
    animation: true,
    animationDuration: 420,
    animationDurationUpdate: 500,
    animationEasingUpdate: 'cubicInOut',
    tooltip: tooltipTemplate, grid: gridTemplate, xAxis: xAxisOption.value, yAxis: yAxisTemplate, series,
    legend: { data: legendData, right: '24px', top: '0px', itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 10, color: '#8C8C8C' } }
  }
})



const loadDatabases = () => {
  const p1 = fetch('/api/databases/db/list/all')
    .then(res => res.json())
    .then(data => data.code === 0 ? (data.data || []) : [])
    .catch(() => [])

  const p2 = fetch('/api/remote-servers')
    .then(res => res.json())
    .then(data => data.code === 0 ? (data.data || []).map(s => ({ ...s, isRemote: true })) : [])
    .catch(() => [])

  Promise.all([p1, p2]).then(([local, remote]) => {
    databases.value = [...local, ...remote]
    checkAllOnlineStatus()
    if (store.connectionId) {
      const matchDb = databases.value.find(d => instanceUid(d) === store.connectionId)
      if (matchDb) {
        selectedUid.value = instanceUid(matchDb)
        fetchMetrics(true)
      }
    } else if (databases.value.length > 0 && !selectedUid.value) {
      selectedUid.value = instanceUid(databases.value[0])
      fetchMetrics(true)
    }
    completeProgress?.()
  })
}

const checkAllOnlineStatus = async () => {
  await healthStore.refreshAll()
}

const selectInstance = (db) => {
  if (selectedUid.value === instanceUid(db)) return
  switchingName.value = db ? db.name : ''
  switching.value = true
  loadError.value = ''
  showProcessPanel.value = false
  selectedUid.value = instanceUid(db)
  metrics.value = null
  prevRawCounters.value = { questions: 0, writes: 0, timestamp: 0 }
  fetchMetrics(true)
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

const fetchMetrics = (rebuildHistory = false) => {
  if (!selectedUid.value) return
  loading.value = true
  if (rebuildHistory) {
    reloadingHistory.value = true
  }
  loadError.value = ''
  metricsRequestId++
  const rid = metricsRequestId
  const url = `/api/dashboard/metrics?server_id=${selectedId.value}&time_range=${timeRange.value}&${sourceParam(selectedDb.value?.isRemote)}`
  fetch(url)
    .then(res => res.json())
    .then(data => {
      if (rid !== metricsRequestId) return
      if (data.code === 0) {
        metrics.value = data.data
        if (data.data.online !== undefined && selectedUid.value) {
          onlineStatus.value = { ...onlineStatus.value, [selectedUid.value]: data.data.online }
        }
        loadError.value = ''

        if (rebuildHistory && data.data.traffic?.echarts) {
          const echarts = data.data.traffic.echarts
          const series = echarts.series || []
          const connSeriesData = series[0]?.data || []
          const qpsSeriesData = series[1]?.data || []
          const tpsSeriesData = series[2]?.data || []

          connHistory.value = connSeriesData.map(([ts, val]) => ({ timestamp: ts, value: val }))
          qpsHistory.value = qpsSeriesData.map(([ts, val]) => ({ timestamp: ts, value: val }))
          tpsHistory.value = tpsSeriesData.map(([ts, val]) => ({ timestamp: ts, value: val }))

          prevRawCounters.value = {
            questions: parseInt(data.data.questions) || 0,
            writes: (parseInt(data.data.com_insert) || 0) + (parseInt(data.data.com_update) || 0) + (parseInt(data.data.com_delete) || 0),
            timestamp: Date.now(),
          }
        } else if (!rebuildHistory) {
          const connVal = data.data.type === 'redis'
            ? parseInt(data.data.connected_clients) || 0
            : parseInt(data.data.threads_connected) || 0
          const nowTs = Date.now()
          connHistory.value.push({ timestamp: nowTs, value: connVal })

          const qpsVal = parseFloat(data.data.qps || data.data.ops_per_sec || 0)
          qpsHistory.value.push({ timestamp: nowTs, value: qpsVal })

          let tpsVal = 0
          if (data.data.type === 'mysql') {
            const curQuestions = parseInt(data.data.questions) || 0
            const curWrites = (parseInt(data.data.com_insert) || 0) + (parseInt(data.data.com_update) || 0) + (parseInt(data.data.com_delete) || 0)
            const prev = prevRawCounters.value
            if (prev.timestamp > 0 && nowTs > prev.timestamp) {
              const elapsedSec = (nowTs - prev.timestamp) / 1000
              if (elapsedSec > 0) {
                const deltaQ = curQuestions - prev.questions
                const deltaW = curWrites - prev.writes
                if (deltaQ >= 0) {
                  qpsHistory.value[qpsHistory.value.length - 1].value = parseFloat((deltaQ / elapsedSec).toFixed(2))
                }
                if (deltaW >= 0) {
                  tpsVal = parseFloat((deltaW / elapsedSec).toFixed(2))
                }
              }
            }
            prevRawCounters.value = { questions: curQuestions, writes: curWrites, timestamp: nowTs }
          }
          tpsHistory.value.push({ timestamp: nowTs, value: tpsVal })

          const rangeMs = timeRange.value * 1000
          const cutoff = nowTs - rangeMs
          connHistory.value = connHistory.value.filter(d => d.timestamp >= cutoff)
          qpsHistory.value = qpsHistory.value.filter(d => d.timestamp >= cutoff)
          tpsHistory.value = tpsHistory.value.filter(d => d.timestamp >= cutoff)
          if (connHistory.value.length > MAX_HISTORY_POINTS) connHistory.value = connHistory.value.slice(-MAX_HISTORY_POINTS)
          if (qpsHistory.value.length > MAX_HISTORY_POINTS) qpsHistory.value = qpsHistory.value.slice(-MAX_HISTORY_POINTS)
          if (tpsHistory.value.length > MAX_HISTORY_POINTS) tpsHistory.value = tpsHistory.value.slice(-MAX_HISTORY_POINTS)
        }
      } else {
        loadError.value = data.msg || '获取指标失败'
        if (!metrics.value) metrics.value = null
      }
    })
    .catch(() => {
      if (rid !== metricsRequestId) return
      loadError.value = '无法连接到服务器，请确认服务已启动'
      if (selectedUid.value) {
        onlineStatus.value = { ...onlineStatus.value, [selectedUid.value]: false }
      }
    })
    .finally(() => {
      if (rid !== metricsRequestId) return
      loading.value = false
      reloadingHistory.value = false
      switching.value = false
    })
}

const togglePolling = () => {
  polling.value = !polling.value
  if (polling.value) {
    startPolling()
  } else {
    stopPolling()
  }
}

const startPolling = () => {
  stopPolling()
  const intervalMs = getRefreshIntervalMs()
  if (intervalMs === 0) {
    polling.value = false
    return
  }
  const countdownSec = Math.floor(intervalMs / 1000)
  countdown.value = countdownSec
  pollTimer = setInterval(() => {
    if (selectedUid.value) {
      fetchMetrics(false)
      countdown.value = countdownSec
    }
  }, intervalMs)
  countdownTimer = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--
    }
  }, 1000)
}

const stopPolling = () => {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
}

watch(connectionId, (newId) => {
  if (newId) {
    const matchDb = databases.value.find(d => instanceUid(d) === newId)
    if (matchDb && instanceUid(matchDb) !== selectedUid.value) {
      selectedUid.value = instanceUid(matchDb)
      metrics.value = null
      fetchMetrics(true)
    }
  }
})

watch(timeRange, (newVal) => {
  localStorage.setItem(TIME_RANGE_KEY, newVal)
  if (selectedUid.value) {
    loadError.value = ''
    fetchMetrics(true)
  }
})

onMounted(() => {
  loadDatabases()
  startPolling()
  window.addEventListener('refresh-interval-change', onRefreshIntervalChange)
})

onActivated(() => {
  loadDatabases()
  if (!pollTimer && polling.value) startPolling()
})

onDeactivated(() => {
  stopPolling()
})

onUnmounted(() => {
  stopPolling()
  window.removeEventListener('refresh-interval-change', onRefreshIntervalChange)
})

const onRefreshIntervalChange = () => {
  if (polling.value) {
    startPolling()
  }
}
</script>

<style scoped>
.fade-slide-in {
  animation: fadeSlideIn 0.3s ease-out;
}

@keyframes fadeSlideIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>