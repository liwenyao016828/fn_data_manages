<template>
  <div class="h-full overflow-y-auto flex flex-col page-padding" style="background: var(--background)">
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-5 shrink-0 fade-up" style="animation-delay: 0ms">
      <div>
        <h1 class="text-xl font-semibold tracking-tight" style="color: var(--text-primary)">仪表盘</h1>
        <p class="text-[13px] mt-0.5" style="color: var(--text-tertiary)">实时数据库监控与性能指标</p>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <!-- Auto-refresh with progress ring -->
        <span
          :class="[
            'flex items-center gap-2 text-xs cursor-pointer select-none px-2.5 py-1.5 rounded-lg transition-colors duration-200',
          ]"
          :style="polling
            ? 'color: var(--success); background: var(--success-soft)'
            : 'color: var(--text-tertiary); background: var(--muted)'"
          @click="togglePolling"
        >
          <svg v-if="polling" class="refresh-ring" width="16" height="16" viewBox="0 0 16 16">
            <circle cx="8" cy="8" r="6" fill="none" stroke="var(--border-subtle)" stroke-width="1.5" />
            <circle
              cx="8" cy="8" r="6" fill="none"
              :stroke="'var(--success)'"
              stroke-width="1.5"
              stroke-linecap="round"
              :stroke-dasharray="37.7"
              :stroke-dashoffset="37.7 * (1 - countdown / Math.max(getRefreshIntervalMs() / 1000, 1))"
              class="refresh-ring-progress"
            />
          </svg>
          <span v-else class="w-1.5 h-1.5 rounded-sm" style="background: var(--text-tertiary)" />
          {{ polling ? `${countdown}s` : '已暂停' }}
        </span>
        <!-- Time Range Selector -->
        <div class="flex items-center gap-1 p-0.5 rounded-full" style="background: var(--muted)">
          <button
            v-for="r in timeRanges"
            :key="r.value"
            :class="timeRange === r.value ? 'pill pill-active' : 'pill pill-default'"
            style="border-radius: 9999px; padding: 4px 10px; font-size: 11px; border: none;"
            @click="timeRange = r.value"
          >
            {{ r.label }}
          </button>
        </div>
        <Loader2 v-if="reloadingHistory" :size="14" class="animate-spin" style="color: var(--accent)" />
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!metrics && !loadError && databases.length === 0" class="empty-state flex-1 py-20">
      <div class="flex flex-col items-center gap-3" style="color: var(--text-tertiary)">
        <Gauge :size="56" class="empty-state-icon" />
        <span class="empty-state-text">暂无数据库实例，请先添加连接</span>
      </div>
    </div>

    <!-- Error Banner -->
    <div v-if="loadError" class="flex items-center gap-2 px-4 py-2.5 mb-4 rounded-xl fade-up" style="background: var(--danger-soft); border: 1px solid color-mix(in srgb, var(--danger) 20%, transparent); color: var(--danger); font-size: 12px;">
      <CircleX class="h-4 w-4 shrink-0" />
      <span>{{ loadError }}</span>
      <Button size="sm" variant="ghost" class="ml-auto h-auto px-2 py-0.5 text-xs" style="color: var(--danger)" @click="loadError = ''; fetchMetrics()">重试</Button>
    </div>

    <template v-if="metrics">
      <!-- Offline Banner -->
      <div v-if="metrics.online === false" class="flex items-center gap-2 px-4 py-2.5 mb-4 rounded-xl fade-up" style="background: var(--warning-soft); border: 1px solid color-mix(in srgb, var(--warning) 20%, transparent); color: var(--warning); font-size: 12px;">
        <CircleX class="h-4 w-4 shrink-0" />
        <span>该数据库实例已离线，图表显示历史数据</span>
      </div>

      <!-- Stat Cards -->
      <div :class="['grid grid-cols-2 md:grid-cols-4 grid-gap section-gap shrink-0 fade-up', contentFading ? 'content-fading' : '']" style="animation-delay: 120ms">
        <!-- Instance Card (drum picker) -->
        <div
          :class="['content-card cursor-pointer relative hover-lift instance-card']"
          @click="toggleInstancePicker"
          @wheel.prevent="onPickerWheel"
          @mouseleave="onCardMouseLeave"
          @mouseenter="onCardMouseEnter"
        >
          <!-- Normal view -->
          <template v-if="!showInstancePicker">
            <div ref="instanceNameContainer" class="flex items-center justify-center h-full w-full">
              <div
                class="font-bold text-center leading-tight instance-name-text"
                :style="{ fontSize: instanceFontSize + 'px', opacity: instanceNameVisible ? 1 : 0 }"
              >{{ instanceName || '选择实例' }}</div>
            </div>
          </template>
          <!-- Drum picker view (replaces card content) -->
          <template v-else>
            <div class="drum-viewport" @click.stop>
              <div
                class="drum-track"
                :style="{ transform: `translateY(${-drumVirtualPos * ROW_H}px)` }"
              >
                <div
                  v-for="(item, vi) in drumVirtualList"
                  :key="vi"
                  :class="['drum-item', vi === drumVirtualPos ? 'drum-item-active' : '']"
                  :style="drumItemStyle(vi)"
                  @click.stop="drumSelectVirtual(vi)"
                >
                  <StatusDot :status="vi === drumVirtualPos ? 'selected' : (onlineStatus[instanceUid(item)] !== false ? 'online' : 'offline')" size="xs" />
                  <span :class="vi === drumVirtualPos ? 'text-[13px] font-semibold' : 'text-[11px] font-medium'" class="truncate flex-1 min-w-0" :style="vi === drumVirtualPos ? 'color: var(--text-primary)' : 'color: var(--text-tertiary)'">{{ item.name }}</span>
                  <span :class="['instance-type-badge ml-auto shrink-0', vi !== drumVirtualPos ? 'instance-type-badge-dim' : '']">{{ item.type }}</span>
                </div>
              </div>
              <!-- center highlight -->
              <div class="drum-highlight" />
            </div>
          </template>
        </div>

        <!-- Uptime Card -->
        <div :class="['content-card hover-lift stat-padding cursor-pointer', uptimeFlipping ? 'card-flip' : '']" @click="flipCard('uptime')">
          <div class="flex items-start justify-between">
            <div ref="addrContainer" class="flex-1 min-w-0">
              <template v-if="!uptimeShowAddr">
                <div class="stat-label">运行时间</div>
                <div class="stat-value text-base" style="color: var(--text-primary)">{{ metrics.uptime_display || '-' }}</div>
              </template>
              <template v-else>
                <div class="stat-label">连接地址</div>
                <div
                  ref="addrEl"
                  class="stat-value font-mono-data"
                  style="color: var(--text-primary); overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; word-break: break-all; transition: font-size 0.15s ease"
                  :style="{ fontSize: addrFontSize + 'px' }"
                >{{ uptimeAddrDisplay }}</div>
              </template>
            </div>
            <div class="stat-icon shrink-0" style="background: var(--success-soft); color: var(--success)">
              <Clock v-if="!uptimeShowAddr" class="h-4 w-4" />
              <Globe v-else class="h-4 w-4" />
            </div>
          </div>
        </div>

        <!-- Connections Card -->
        <div
          :class="[
            'content-card hover-lift stat-padding cursor-pointer',
          ]"
          :style="connStatusClass === 'danger'
            ? 'border-color: color-mix(in srgb, var(--danger) 30%, transparent); background: var(--danger-soft)'
            : connStatusClass === 'warn'
            ? 'border-color: color-mix(in srgb, var(--warning) 30%, transparent); background: var(--warning-soft)'
            : ''"
          @click="showProcessPanel = !showProcessPanel"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="stat-label">
                当前连接
                <span class="text-[10px] ml-0.5" style="color: var(--text-tertiary)">{{ showProcessPanel ? '▲' : '▼' }}</span>
              </div>
              <div class="stat-value text-base truncate font-mono-data" style="color: var(--text-primary)">{{ connCurrent }} / {{ metrics.max_connections || '-' }}</div>
              <div class="flex items-center gap-2 mt-1.5">
                <div class="flex-1 h-1 rounded-full overflow-hidden" style="background: var(--border-subtle)">
                  <div
                    class="h-full rounded-full transition-all duration-500"
                    :style="{
                      width: connUsageNum + '%',
                      background: connStatusClass === 'danger' ? 'var(--danger)' : connStatusClass === 'warn' ? 'var(--warning)' : 'var(--success)'
                    }"
                  />
                </div>
                <span class="text-[11px] font-mono-data shrink-0 leading-[16px]" style="color: var(--text-tertiary)">{{ metrics.connection_usage === 'N/A' ? 'N/A' : connUsageNum.toFixed(1) + '%' }}</span>
              </div>
            </div>
            <div
              class="stat-icon"
              :style="connStatusClass === 'danger'
                ? 'background: var(--danger-soft); color: var(--danger)'
                : connStatusClass === 'warn'
                ? 'background: var(--warning-soft); color: var(--warning)'
                : 'background: var(--success-soft); color: var(--success)'"
            >
              <Link class="h-4 w-4" />
            </div>
          </div>
        </div>

        <!-- QPS Card -->
        <div :class="['content-card hover-lift stat-padding cursor-pointer', qpsFlipping ? 'card-flip' : '']" @click="flipCard('qps')">
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <template v-if="!qpsShowTps">
                <div class="stat-label">{{ qpsLabel }}</div>
                <div class="stat-value text-base truncate font-mono-data" style="color: var(--text-primary)">{{ metrics.qps || metrics.ops_per_sec || '-' }}</div>
              </template>
              <template v-else>
                <div class="stat-label">{{ tpsLabel }}</div>
                <div class="stat-value text-base truncate font-mono-data" style="color: var(--text-primary)">{{ metrics.tps || '-' }}</div>
              </template>
            </div>
            <div class="stat-icon" style="background: var(--warning-soft); color: var(--warning)">
              <TrendingUp class="h-4 w-4" />
            </div>
          </div>
        </div>
      </div>

      <!-- Process Panel -->
      <div v-if="showProcessPanel && (metrics.type === 'mysql' || metrics.type === 'postgresql') && metrics.processlist" :class="['section-gap shrink-0 fade-up', contentFading ? 'content-fading' : '']" style="animation-delay: 180ms">
        <div class="content-card stat-padding">
          <Table>
            <TableHeader>
              <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">用户</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">来源</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">数据库</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">命令</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">时长</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">状态</TableHead>
                <TableHead class="text-[11px] font-normal max-w-[200px]" style="color: var(--text-tertiary)">信息</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow
                v-for="p in metrics.processlist"
                :key="p.id"
                :class="p.time > 10 ? 'process-row-danger' : 'process-row-normal'"
                style="border-bottom: 1px solid var(--border-subtle)"
              >
                <TableCell>
                  <Badge variant="secondary" class="text-[11px] font-medium" style="background: var(--muted); color: var(--text-primary)">{{ p.user }}</Badge>
                </TableCell>
                <TableCell class="font-mono-data text-[11px]" style="color: var(--text-secondary)">{{ p.host }}</TableCell>
                <TableCell class="text-xs" style="color: var(--text-primary)">{{ p.db || '-' }}</TableCell>
                <TableCell>
                  <span class="badge-status badge-status-success text-[10px] font-semibold rounded-full">{{ p.command }}</span>
                </TableCell>
                <TableCell :class="['font-mono-data text-[11px]', p.time > 10 ? 'font-semibold' : '']" :style="p.time > 10 ? 'color: var(--danger)' : 'color: var(--text-primary)'">{{ p.time }}s</TableCell>
                <TableCell class="text-xs max-w-[100px] truncate" style="color: var(--text-primary)">{{ p.state || '-' }}</TableCell>
                <TableCell class="text-xs max-w-[200px] truncate" style="color: var(--text-primary)" :title="p.info">{{ p.info || '-' }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </div>

      <!-- Chart Cards -->
      <div :class="['grid grid-cols-1 lg:grid-cols-2 grid-gap section-gap shrink-0 fade-up', contentFading ? 'content-fading' : '']" style="animation-delay: 240ms">
        <div class="content-card stat-padding">
          <div class="flex justify-between items-center mb-3">
            <span class="text-xs font-semibold" style="color: var(--text-primary)">网络</span>
          </div>
          <v-chart style="height: 180px; width: 100%" :option="netChartOption" autoresize />
        </div>

        <div class="content-card stat-padding">
          <div class="flex justify-between items-center mb-3">
            <span class="text-xs font-semibold" style="color: var(--text-primary)">{{ metrics.type === 'redis' ? 'Ops/sec 趋势' : 'QPS/TPS 双驱趋势' }}</span>
          </div>
          <v-chart style="height: 180px; width: 100%" :option="qpsTpsChartOption" autoresize />
        </div>
      </div>

      <Transition name="detail-cards" mode="out-in">
      <div v-if="metrics.type === 'mysql'" key="mysql" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 grid-gap shrink-0">
        <!-- Threads -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">线程</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">运行中</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--accent)">{{ metrics.threads_running }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">已连接</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.threads_connected }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">缓存中</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.threads_cached }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">历史最大连接</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.max_used_connection }}</span>
            </div>
          </div>
        </div>

        <!-- Connection Trend -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">连接数趋势</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">当前</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--accent)">{{ connCurrent }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">最大</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.max_used_connection }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">使用率</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.connection_usage }}%</span>
            </div>
          </div>
        </div>

        <!-- Query Distribution -->
        <div v-if="metrics" class="content-card stat-padding flex flex-col">
          <div class="flex items-center justify-between mb-3 pb-2" style="border-bottom: 1px solid var(--border-subtle)">
            <span class="text-xs font-semibold" style="color: var(--text-primary)">查询分布</span>
            <span v-if="rangeStatsAvailable" class="text-[10px] px-1.5 py-0.5 rounded" style="background: var(--accent-soft); color: var(--accent)">{{ timeRangeLabel }}</span>
          </div>
          <div v-if="queryTotal > 0" class="relative flex-1" style="min-height: 80px;">
            <!-- Donut center -->
            <div class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2" style="width: min(50%, 100px); aspect-ratio: 1;">
              <svg viewBox="0 0 36 36" class="w-full h-full -rotate-90">
                <circle cx="18" cy="18" r="14" fill="none" stroke="var(--border-subtle)" stroke-width="4" />
                <circle v-for="(seg, i) in queryDonutSegments" :key="i"
                  cx="18" cy="18" r="14" fill="none"
                  :stroke="seg.color" stroke-width="4"
                  :stroke-dasharray="`${seg.arc} ${seg.gap}`"
                  :stroke-dashoffset="seg.offset"
                  stroke-linecap="round"
                  class="transition-all duration-500"
                />
              </svg>
              <div class="absolute inset-0 flex flex-col items-center justify-center">
                <span class="text-[11px] font-bold font-mono-data" style="color: var(--text-primary)">{{ formatNum(queryTotal) }}</span>
                <span class="text-[7px]" style="color: var(--text-tertiary)">总数</span>
              </div>
            </div>
            <!-- Top Left: SELECT -->
            <div class="absolute top-0 left-0">
              <div class="flex items-center gap-0.5">
                <span class="w-1.5 h-1.5 rounded-full shrink-0" :style="{ background: queryLegend[0]?.color }" />
                <span class="text-[9px] font-semibold" style="color: var(--text-secondary)">{{ queryLegend[0]?.label }}</span>
              </div>
              <span class="text-[8px] font-mono-data pl-2" style="color: var(--text-tertiary)">{{ formatNum(queryLegend[0]?.value) }}</span>
            </div>
            <!-- Top Right: INSERT -->
            <div class="absolute top-0 right-0">
              <div class="flex items-center gap-0.5">
                <span class="text-[9px] font-semibold" style="color: var(--text-secondary)">{{ queryLegend[1]?.label }}</span>
                <span class="w-1.5 h-1.5 rounded-full shrink-0" :style="{ background: queryLegend[1]?.color }" />
              </div>
              <span class="text-[8px] font-mono-data pr-2" style="color: var(--text-tertiary)">{{ formatNum(queryLegend[1]?.value) }}</span>
            </div>
            <!-- Bottom Left: UPDATE -->
            <div class="absolute bottom-0 left-0">
              <div class="flex items-center gap-0.5">
                <span class="w-1.5 h-1.5 rounded-full shrink-0" :style="{ background: queryLegend[2]?.color }" />
                <span class="text-[9px] font-semibold" style="color: var(--text-secondary)">{{ queryLegend[2]?.label }}</span>
              </div>
              <span class="text-[8px] font-mono-data pl-2" style="color: var(--text-tertiary)">{{ formatNum(queryLegend[2]?.value) }}</span>
            </div>
            <!-- Bottom Right: DELETE -->
            <div class="absolute bottom-0 right-0">
              <div class="flex items-center gap-0.5">
                <span class="text-[9px] font-semibold" style="color: var(--text-secondary)">{{ queryLegend[3]?.label }}</span>
                <span class="w-1.5 h-1.5 rounded-full shrink-0" :style="{ background: queryLegend[3]?.color }" />
              </div>
              <span class="text-[8px] font-mono-data pr-2" style="color: var(--text-tertiary)">{{ formatNum(queryLegend[3]?.value) }}</span>
            </div>
          </div>
          <div v-else class="flex items-center justify-center py-4 text-[12px]" style="color: var(--text-tertiary)">
            暂无查询数据
          </div>
        </div>

        <!-- InnoDB Buffer Pool -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">InnoDB 缓冲池</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">命中率</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="(parseFloat(metrics.innodb_buffer_pool_hit_rate) || 0) > 95 ? 'color: var(--success)' : 'color: var(--warning)'">
                {{ metrics.innodb_buffer_pool_hit_rate || '0.00' }}%
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">空闲页</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.innodb_buffer_pool_pages_free || '-' }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">总页数</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.innodb_buffer_pool_pages_total || '-' }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">脏页</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.innodb_buffer_pool_pages_dirty || '-' }}</span>
            </div>
          </div>
          <div class="mt-3">
            <div class="h-1.5 rounded-full overflow-hidden" style="background: var(--border-subtle)">
              <div
                class="h-full rounded-full transition-all"
                :style="{
                  width: Math.min(parseFloat(metrics.innodb_buffer_pool_hit_rate) || 0, 100) + '%',
                  background: (parseFloat(metrics.innodb_buffer_pool_hit_rate) || 0) > 95 ? 'var(--success)' : 'var(--warning)'
                }"
              />
            </div>
          </div>
        </div>

        <!-- Efficiency Metrics -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">效率指标</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">慢查询</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="parseInt(metrics.slow_queries) > 0 ? 'color: var(--warning)' : 'color: var(--text-primary)'">
                {{ metrics.slow_queries }}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">表锁等待</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.table_locks_waited }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">磁盘临时表</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="parseInt(metrics.tmp_table_disk) > 500 ? 'color: var(--warning)' : 'color: var(--text-primary)'">{{ metrics.tmp_table_disk }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">中断连接</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="parseInt(metrics.aborted_connects) > 10 ? 'color: var(--warning)' : 'color: var(--text-primary)'">{{ metrics.aborted_connects }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="metrics.type === 'redis'" key="redis" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 grid-gap shrink-0">
        <!-- Memory -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">内存</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">当前使用</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.used_memory }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">峰值</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.used_memory_peak }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">碎片率</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.mem_fragmentation }}</span>
            </div>
          </div>
        </div>

        <!-- Clients -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">客户端</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">已连接</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.connected_clients }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">阻塞中</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.blocked_clients }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">最大限制</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.maxclients }}</span>
            </div>
          </div>
        </div>

        <!-- Keyspace -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">键空间</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">命中</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.keyspace_hits }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">未命中</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.keyspace_misses }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">命中率</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="(parseFloat(metrics.hit_rate) || 0) > 90 ? 'color: var(--success)' : 'color: var(--warning)'">
                {{ metrics.hit_rate || '0.00' }}%
              </span>
            </div>
          </div>
        </div>

        <!-- Persistence -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">持久化</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">AOF</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="metrics.aof_enabled === '1' ? 'color: var(--success)' : 'color: var(--text-primary)'">
                {{ metrics.aof_enabled === '1' ? '已启用' : '未启用' }}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">RDB 变更数</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.rdb_changes }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="metrics.type === 'postgresql'" key="postgresql" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 grid-gap shrink-0">
        <!-- Cache Hit Rate -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">缓存命中率</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">命中率</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="(parseFloat(metrics.cache_hit_rate) || 0) > 95 ? 'color: var(--success)' : 'color: var(--warning)'">
                {{ metrics.cache_hit_rate || '0.00' }}%
              </span>
            </div>
            <div class="mt-1">
              <div class="h-1.5 rounded-full overflow-hidden" style="background: var(--border-subtle)">
                <div
                  class="h-full rounded-full transition-all"
                  :style="{
                    width: Math.min(parseFloat(metrics.cache_hit_rate) || 0, 100) + '%',
                    background: (parseFloat(metrics.cache_hit_rate) || 0) > 95 ? 'var(--success)' : 'var(--warning)'
                  }"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Transaction Stats -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">事务统计</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">已提交</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--success)">{{ formatNum(metrics.xact_commit) }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">已回滚</span>
              <span :class="['text-xs font-semibold font-mono-data']" :style="parseInt(metrics.xact_rollback) > 0 ? 'color: var(--warning)' : 'color: var(--text-primary)'">{{ formatNum(metrics.xact_rollback) }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">数据库大小</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.data_total_size || '-' }}</span>
            </div>
          </div>
        </div>

        <!-- Connection Info -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">连接信息</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">当前连接</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--accent)">{{ metrics.threads_connected }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">最大连接</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.max_connections }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">使用率</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.connection_usage }}%</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="metrics.type === 'sqlite'" key="sqlite" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 grid-gap shrink-0">
        <!-- Database Info -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">数据库信息</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">表数量</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--accent)">{{ metrics.table_count || '0' }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">文件大小</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.data_total_size || '-' }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">文件路径</span>
              <span class="text-xs font-mono-data truncate max-w-[180px]" style="color: var(--text-tertiary)" :title="metrics.host">{{ metrics.host }}</span>
            </div>
          </div>
        </div>

        <!-- Journal Mode -->
        <div class="content-card stat-padding">
          <div class="text-xs font-semibold mb-3 pb-2" style="color: var(--text-primary); border-bottom: 1px solid var(--border-subtle)">存储引擎</div>
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">日志模式</span>
              <span class="text-xs font-semibold font-mono-data" :style="metrics.journal_mode === 'wal' ? 'color: var(--success)' : 'color: var(--text-primary)'">{{ metrics.journal_mode || '-' }}</span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-xs" style="color: var(--text-secondary)">版本</span>
              <span class="text-xs font-semibold font-mono-data" style="color: var(--text-primary)">{{ metrics.version || '-' }}</span>
            </div>
          </div>
        </div>
      </div>
      </Transition>

      <!-- PostgreSQL Process Panel -->
      <div v-if="showProcessPanel && metrics.type === 'postgresql' && metrics.processlist" :class="['section-gap shrink-0 fade-up', contentFading ? 'content-fading' : '']" style="animation-delay: 180ms">
        <div class="content-card stat-padding">
          <Table>
            <TableHeader>
              <TableRow class="hover:bg-transparent" style="border-bottom: 1px solid var(--border-subtle)">
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">PID</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">用户</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">来源</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">状态</TableHead>
                <TableHead class="text-[11px] font-normal" style="color: var(--text-tertiary)">时长</TableHead>
                <TableHead class="text-[11px] font-normal max-w-[300px]" style="color: var(--text-tertiary)">查询</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow
                v-for="p in metrics.processlist"
                :key="p.id"
                :class="p.time > 10 ? 'process-row-danger' : 'process-row-normal'"
                style="border-bottom: 1px solid var(--border-subtle)"
              >
                <TableCell class="font-mono-data text-[11px]" style="color: var(--text-primary)">{{ p.id }}</TableCell>
                <TableCell>
                  <Badge variant="secondary" class="text-[11px] font-medium" style="background: var(--muted); color: var(--text-primary)">{{ p.user || '-' }}</Badge>
                </TableCell>
                <TableCell class="font-mono-data text-[11px]" style="color: var(--text-secondary)">{{ p.host || '-' }}</TableCell>
                <TableCell>
                  <span class="badge-status badge-status-success text-[10px] font-semibold rounded-full">{{ p.command || '-' }}</span>
                </TableCell>
                <TableCell :class="['font-mono-data text-[11px]', p.time > 10 ? 'font-semibold' : '']" :style="p.time > 10 ? 'color: var(--danger)' : 'color: var(--text-primary)'">{{ p.time }}s</TableCell>
                <TableCell class="text-xs max-w-[300px] truncate" style="color: var(--text-primary)" :title="p.info">{{ p.info || '-' }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
defineOptions({ name: 'DashboardView' })
import { ref, computed, onMounted, onActivated, onDeactivated, onUnmounted, watch, inject, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppContext } from '../stores/context'
import { useHealthStore } from '../stores/health'
import { useThemeStore } from '../stores/theme'
import { sourceParam, instanceUid } from '@/lib/instance'
import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { RefreshCw, Gauge, CircleX, Loader2, Monitor, Clock, Link, TrendingUp, Globe } from 'lucide-vue-next'
import StatusDot from './StatusDot.vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { STORAGE_KEYS, safeStorage } from '@/lib/storageKeys'

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
const contentFading = ref(false)
const showProcessPanel = ref(false)
const showInstancePicker = ref(false)
const uptimeShowAddr = ref(false)
const qpsShowTps = ref(false)
const uptimeFlipping = ref(false)
const qpsFlipping = ref(false)
let uptimeAutoSwitch = false // track if we auto-switched

// Horizontal flip animation for cards
const flipCard = (card) => {
  // Don't flip uptime card during drum scrolling
  if (card === 'uptime' && showInstancePicker.value) return

  const flippingRef = card === 'uptime' ? uptimeFlipping : qpsFlipping
  const stateRef = card === 'uptime' ? uptimeShowAddr : qpsShowTps

  flippingRef.value = true
  // At halfway (90deg), toggle the content
  setTimeout(() => {
    stateRef.value = !stateRef.value
  }, 200)
  // After animation completes, remove flip class
  setTimeout(() => {
    flippingRef.value = false
  }, 400)
}

// When picker opens, auto-switch uptime card to show address
watch(showInstancePicker, (val) => {
  if (val) {
    if (!uptimeShowAddr.value) {
      uptimeAutoSwitch = true
      flipCard('uptime')
    }
  } else {
    if (uptimeAutoSwitch) {
      uptimeAutoSwitch = false
      flipCard('uptime')
    }
  }
})

// Display address that follows drum preview
const uptimeAddrDisplay = computed(() => {
  if (showInstancePicker.value && drumVirtualPos.value >= 0) {
    const dbs = sortedDatabases.value
    if (dbs.length > 0) {
      const db = dbs[drumVirtualPos.value % dbs.length]
      if (db) return `${db.host}:${db.port}`
    }
  }
  return `${metrics.value?.host || ''}:${metrics.value?.port || ''}`
})
const polling = ref(true)
const countdown = ref(3)
const connHistory = ref([])
const qpsHistory = ref([])
const tpsHistory = ref([])
const healthStore = useHealthStore()
const themeStore = useThemeStore()
const { statusMap: onlineStatus } = storeToRefs(healthStore)
const prevRawCounters = ref({ questions: 0, writes: 0, timestamp: 0 })

const isDark = computed(() => themeStore.theme === 'dark')
const chartAxisColor = computed(() => isDark.value ? '#64748b' : '#94a3b8')
const chartGridColor = computed(() => isDark.value ? 'rgba(148, 163, 184, 0.08)' : 'rgba(148, 163, 184, 0.15)')
const chartTooltipBg = computed(() => isDark.value ? 'rgba(0, 0, 0, 0.85)' : 'rgba(255, 255, 255, 0.95)')
const chartTooltipBorder = computed(() => isDark.value ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.08)')
const chartTooltipText = computed(() => isDark.value ? '#fff' : '#0f172a')
const timeRanges = [
  { value: 900, label: '15分钟' },
  { value: 3600, label: '1小时' },
  { value: 21600, label: '6小时' },
  { value: 86400, label: '24小时' },
  { value: 604800, label: '7天' },
]
const TIME_RANGE_KEY = STORAGE_KEYS.DASHBOARD_TIME_RANGE
const timeRange = ref(parseInt(safeStorage.get(TIME_RANGE_KEY)) || 3600)
const MAX_HISTORY_POINTS = 15000
let pollTimer = null
let countdownTimer = null
let metricsRequestId = 0

const getRefreshIntervalMs = () => {
  const val = safeStorage.get(STORAGE_KEYS.DASHBOARD_REFRESH_INTERVAL)
  if (!val || val === 'off') return 0
  const ms = parseInt(val)
  if (isNaN(ms) || ms <= 0) return 0
  return ms
}

const store = useAppContext()
const { connectionId } = storeToRefs(store)
const { favorites } = storeToRefs(store)

const instanceName = computed(() => {
  if (!selectedUid.value) return ''
  return selectedDb.value ? selectedDb.value.name : ''
})

const instanceNameContainer = ref(null)
const instanceFontSize = ref(28)
const instanceNameVisible = ref(true)

// 根据数据库类型返回QPS标签
const qpsLabel = computed(() => {
  const t = metrics.value?.type
  if (t === 'redis') return '每秒操作'
  if (t === 'postgresql') return 'QPS'
  if (t === 'sqlite') return '读/秒'
  return 'QPS'
})

// 根据数据库类型返回TPS标签
const tpsLabel = computed(() => {
  const t = metrics.value?.type
  if (t === 'redis') return '命中/秒'
  if (t === 'postgresql') return 'TPS'
  if (t === 'sqlite') return '写/秒'
  return 'TPS'
})

// 连接地址自适应字号
const addrContainer = ref(null)
const addrEl = ref(null)
const addrFontSize = ref(16)

const calcAddrFontSize = () => {
  const container = addrContainer.value
  const el = addrEl.value
  if (!container || !el) return

  const containerW = container.clientWidth
  if (!containerW) return

  // 临时设置nowrap确保测量准确
  const origWS = el.style.whiteSpace
  el.style.whiteSpace = 'nowrap'

  let size = 16
  el.style.fontSize = size + 'px'
  while (size > 9 && el.scrollWidth > containerW) {
    size -= 1
    el.style.fontSize = size + 'px'
  }
  addrFontSize.value = size

  // 恢复
  el.style.whiteSpace = origWS
}

watch(uptimeShowAddr, (v) => {
  if (v) nextTick(() => calcAddrFontSize())
})
watch(uptimeAddrDisplay, () => {
  if (uptimeShowAddr.value) nextTick(() => calcAddrFontSize())
})

// 动态计算最大字体大小：尽可能大，但不超出容器
const CHAR_WIDTH_RATIO = 0.6 // 中英混合字符平均宽度与字号的比例

// 纯计算字号，不依赖DOM
const calcFontSizeForName = (name) => {
  const container = instanceNameContainer.value
  if (!container) return 28

  const containerW = container.clientWidth
  const containerH = container.clientHeight
  if (!name || !containerW || !containerH) return 28

  const charCount = name.length
  const singleLineSize = Math.min(containerH, Math.floor(containerW / (charCount * CHAR_WIDTH_RATIO)))

  if (singleLineSize >= 18) {
    return Math.min(singleLineSize, 48)
  }

  const halfChars = Math.ceil(charCount / 2)
  const twoLineSize = Math.min(
    Math.floor(containerH / 2.4),
    Math.floor(containerW / (halfChars * CHAR_WIDTH_RATIO)),
    28
  )
  return Math.max(twoLineSize, 14)
}

const calcInstanceFontSize = () => {
  const name = instanceName.value || '选择实例'
  instanceFontSize.value = calcFontSizeForName(name)
}

watch(instanceName, (newName) => {
  // 先隐藏
  instanceNameVisible.value = false
  // 立即算好字号（容器尺寸没变，可以直接算）
  instanceFontSize.value = calcFontSizeForName(newName || '选择实例')
  // 下一帧淡入（此时字号已确定，名称和字号同时渲染）
  requestAnimationFrame(() => {
    instanceNameVisible.value = true
  })
})

onMounted(() => {
  nextTick(() => calcInstanceFontSize())
  // 监听容器大小变化
  if (instanceNameContainer.value) {
    const ro = new ResizeObserver(() => calcInstanceFontSize())
    ro.observe(instanceNameContainer.value)
    onUnmounted(() => ro.disconnect())
  }
})

const toggleInstancePicker = () => {
  if (showInstancePicker.value) {
    showInstancePicker.value = false
  } else {
    showInstancePicker.value = true
    initDrumPos()
  }
}

const pickInstance = (db) => {
  selectInstance(db)
}

// Drum picker with virtual looping list
let mouseLeaveTimer = null
const ROW_H = 26
const DRUM_REPEATS = 20

const drumIndex = computed(() => {
  if (!selectedUid.value) return 0
  const idx = sortedDatabases.value.findIndex(d => instanceUid(d) === selectedUid.value)
  return idx >= 0 ? idx : 0
})

// Virtual list: repeat sortedDatabases N times
const drumVirtualList = computed(() => {
  const dbs = sortedDatabases.value
  if (dbs.length === 0) return []
  const list = []
  for (let r = 0; r < DRUM_REPEATS; r++) {
    for (let i = 0; i < dbs.length; i++) {
      list.push(dbs[i])
    }
  }
  return list
})

// Track virtual position directly (no recalculation from real index)
const drumVirtualPos = ref(0)

// Initialize virtual position when opening picker
const initDrumPos = () => {
  const dbs = sortedDatabases.value
  drumVirtualPos.value = Math.floor(DRUM_REPEATS / 2) * dbs.length + drumIndex.value
}

const drumItemStyle = (vi) => {
  const dist = Math.abs(vi - drumVirtualPos.value)
  const scale = dist === 0 ? 1 : Math.max(0.82, 1 - dist * 0.1)
  const opacity = dist === 0 ? 1 : Math.max(0.3, 1 - dist * 0.25)
  return {
    transform: `scale(${scale})`,
    opacity,
  }
}

// Click a virtual row
const drumSelectVirtual = (vi) => {
  const dbs = sortedDatabases.value
  if (dbs.length === 0) return
  if (vi === drumVirtualPos.value) {
    // Confirm current selection
    const realIdx = vi % dbs.length
    const db = dbs[realIdx]
    if (db) selectInstance(db)
    showInstancePicker.value = false
  } else {
    // Preview: just move to that position
    drumVirtualPos.value = vi
  }
}

const onPickerWheel = (e) => {
  const dbs = sortedDatabases.value
  if (dbs.length <= 1) return
  // 如果 picker 没打开，先打开再切换
  if (!showInstancePicker.value) {
    showInstancePicker.value = true
    initDrumPos()
  }
  // 确保运行时间卡片翻转到地址面
  if (!uptimeShowAddr.value) {
    uptimeAutoSwitch = true
    flipCard('uptime')
  }
  const dir = e.deltaY > 0 ? 1 : -1
  drumVirtualPos.value += dir
}

const onCardMouseLeave = () => {
  if (!showInstancePicker.value) return
  mouseLeaveTimer = setTimeout(() => {
    showInstancePicker.value = false
  }, 1000)
}

const onCardMouseEnter = () => {
  if (mouseLeaveTimer) {
    clearTimeout(mouseLeaveTimer)
    mouseLeaveTimer = null
  }
}

// Close picker when clicking outside (handled by overlay click.self)

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
  if (val == null || val === '-') return '-'
  const n = parseFloat(val)
  if (isNaN(n)) return '-'
  if (n < 0) return '-'
  if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

const queryBarWidth = (val) => {
  const numVal = Math.max(parseInt(val) || 0, 0)
  if (numVal === 0) return '0%'
  const clampZero = (v) => Math.max(parseInt(v) || 0, 0)
  let values
  if (rangeStatsAvailable.value) {
    values = [
      clampZero(rangeStats.deltaComSelect),
      clampZero(rangeStats.deltaComInsert),
      clampZero(rangeStats.deltaComUpdate),
      clampZero(rangeStats.deltaComDelete)
    ]
  } else {
    values = [
      clampZero(metrics.value?.com_select),
      clampZero(metrics.value?.com_insert),
      clampZero(metrics.value?.com_update),
      clampZero(metrics.value?.com_delete)
    ]
  }
  const max = Math.max(...values, 1)
  const width = (numVal / max) * 100
  return `${Math.max(width, 1)}%`
}

const queryValues = computed(() => {
  const clampZero = (v) => Math.max(parseInt(v) || 0, 0)
  if (rangeStatsAvailable.value) {
    const rs = rangeStats.value
    const deltas = [
      clampZero(rs.deltaComSelect),
      clampZero(rs.deltaComInsert),
      clampZero(rs.deltaComUpdate),
      clampZero(rs.deltaComDelete)
    ]
    return deltas
  }
  return [
    clampZero(metrics.value?.com_select),
    clampZero(metrics.value?.com_insert),
    clampZero(metrics.value?.com_update),
    clampZero(metrics.value?.com_delete)
  ]
})

const queryTotal = computed(() => queryValues.value.reduce((a, b) => a + b, 0))

const queryLegend = computed(() => {
  const labels = ['SELECT', 'INSERT', 'UPDATE', 'DELETE']
  const colors = ['var(--accent)', 'var(--success)', 'var(--warning)', 'var(--danger)']
  const total = Math.max(queryTotal.value, 1)
  return queryValues.value.map((v, i) => ({
    label: labels[i],
    color: colors[i],
    value: v,
    pct: ((v / total) * 100).toFixed(1)
  }))
})

const queryDonutSegments = computed(() => {
  const total = Math.max(queryTotal.value, 1)
  const colors = ['var(--accent)', 'var(--success)', 'var(--warning)', 'var(--danger)']
  const circumference = 2 * Math.PI * 14 // ~87.96
  let cumulativeOffset = 0
  return queryValues.value.map((v, i) => {
    const ratio = v / total
    const arc = ratio * circumference
    const seg = { color: colors[i], arc: arc.toFixed(2), gap: (circumference - arc).toFixed(2), offset: (-cumulativeOffset).toFixed(2) }
    cumulativeOffset += arc
    return seg
  })
})

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
  if (metrics.value.type === 'sqlite') return '1'
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
      color: chartAxisColor.value,
      fontSize: 9,
      formatter: formatter,
      hideOverlap: true,
      interval: 'auto',
    },
    splitLine: { show: false },
  }
})
const yAxisTemplate = computed(() => ({
  type: 'value',
  axisLine: { show: false },
  axisTick: { show: false },
  axisLabel: { color: chartAxisColor.value, fontSize: 9 },
  splitLine: { lineStyle: { color: chartGridColor.value } },
  minInterval: 1,
}))
const gridTemplate = { left: '8px', right: '8px', bottom: '8px', top: '24px', containLabel: true }
const tooltipTemplate = computed(() => ({
  trigger: 'axis',
  axisPointer: { type: 'cross' },
  backgroundColor: chartTooltipBg.value,
  borderColor: chartTooltipBorder.value,
  borderWidth: 1,
  textStyle: { color: chartTooltipText.value, fontSize: 11 },
}))

const connChartOption = computed(() => {
  const data = connHistory.value.map(d => [d.timestamp, d.value])
  return {
    animation: true,
    animationDuration: 420,
    animationDurationUpdate: 500,
    animationEasingUpdate: 'cubicInOut',
    tooltip: tooltipTemplate.value,
    grid: gridTemplate,
    xAxis: xAxisOption.value,
    yAxis: yAxisTemplate.value,
    legend: { data: ['连接数'], right: '24px', top: '0px', itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 10, color: chartAxisColor.value } },
    series: [{
      name: '连接数', type: 'line', smooth: true, symbol: 'none',
      lineStyle: { color: '#3b82f6', width: 2 }, itemStyle: { color: '#3b82f6' },
      areaStyle: { opacity: isDark.value ? 0.08 : 0.12, color: '#3b82f6' },
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
      tooltip: tooltipTemplate.value, grid: gridTemplate, xAxis: xAxisOption.value, yAxis: { type: 'value', axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: chartAxisColor.value, fontSize: 9, formatter: (v) => formatBytes(v) + '/s' }, splitLine: { lineStyle: { color: chartGridColor.value } }, min: 0 },
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
      ...tooltipTemplate.value,
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
      axisLabel: { color: chartAxisColor.value, fontSize: 9, formatter: (v) => formatBytes(v) + '/s' },
      splitLine: { lineStyle: { color: chartGridColor.value } },
      min: 0,
    },
    series: [
      {
        name: '接收', type: 'line', smooth: true, symbol: 'none',
        lineStyle: { color: '#3b82f6', width: 2 }, itemStyle: { color: '#3b82f6' },
        areaStyle: { opacity: isDark.value ? 0.08 : 0.12, color: '#3b82f6' },
        emphasis: { focus: 'series' }, connectNulls: true, data: inData,
      },
      {
        name: '发送', type: 'line', smooth: true, symbol: 'none',
        lineStyle: { color: '#8b5cf6', width: 2 }, itemStyle: { color: '#8b5cf6' },
        areaStyle: { opacity: isDark.value ? 0.08 : 0.12, color: '#8b5cf6' },
        emphasis: { focus: 'series' }, connectNulls: true, data: outData,
      },
    ],
    legend: { data: ['接收', '发送'], right: '24px', top: '0px', itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 10, color: chartAxisColor.value } },
  }
})

const qpsTpsChartOption = computed(() => {
  const mk = (arr) => arr.map(d => [d.timestamp, d.value])
  const series = [{
    name: 'QPS', type: 'line', smooth: true, symbol: 'none',
    lineStyle: { color: '#3b82f6', width: 2 }, itemStyle: { color: '#3b82f6' },
    areaStyle: { opacity: isDark.value ? 0.08 : 0.12, color: '#3b82f6' },
    emphasis: { focus: 'series' },
    connectNulls: true,
    data: mk(qpsHistory.value),
  }]
  const legendData = ['QPS']
  if (metrics.value?.type === 'mysql') {
    series.push({
      name: 'TPS', type: 'line', smooth: true, symbol: 'none',
      lineStyle: { color: '#8b5cf6', width: 2, type: 'dashed' }, itemStyle: { color: '#8b5cf6' },
      areaStyle: { opacity: isDark.value ? 0.08 : 0.12, color: '#8b5cf6' },
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
    tooltip: tooltipTemplate.value, grid: gridTemplate, xAxis: xAxisOption.value, yAxis: yAxisTemplate.value, series,
    legend: { data: legendData, right: '24px', top: '0px', itemWidth: 10, itemHeight: 10, textStyle: { fontSize: 10, color: chartAxisColor.value } }
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

const checkAllOnlineStatus = () => {
  healthStore.refreshAll()
}

const selectInstance = (db) => {
  if (selectedUid.value === instanceUid(db)) return
  loadError.value = ''
  showProcessPanel.value = false
  selectedUid.value = instanceUid(db)
  prevRawCounters.value = { questions: 0, writes: 0, timestamp: 0 }
  contentFading.value = true
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

        // 自动翻卡片：轮询刷新时交替显示地址/名称
        if (!rebuildHistory) {
          flipCard('uptime')
          flipCard('qps')
        }

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
      contentFading.value = false
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
  let intervalMs = getRefreshIntervalMs()
  if (intervalMs === 0 || intervalMs < 5000) {
    intervalMs = 10000
    safeStorage.set(STORAGE_KEYS.DASHBOARD_REFRESH_INTERVAL, '10000')
  }
  polling.value = true
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
      contentFading.value = true
      fetchMetrics(true)
    }
  }
})

watch(timeRange, (newVal) => {
  safeStorage.set(TIME_RANGE_KEY, newVal)
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
  // 仅在数据为空时重新加载，避免 KeepAlive 切换时重复请求
  if (databases.value.length === 0) loadDatabases()
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
  const intervalMs = getRefreshIntervalMs()
  if (intervalMs === 0) {
    polling.value = false
    stopPolling()
  } else {
    if (polling.value) {
      startPolling()
    }
  }
}
</script>

<style scoped>
/* Card horizontal flip animation */
.card-flip {
  animation: cardFlipX 0.4s ease-in-out;
}
@keyframes cardFlipX {
  0% { transform: perspective(600px) rotateY(0deg); }
  50% { transform: perspective(600px) rotateY(90deg); }
  100% { transform: perspective(600px) rotateY(0deg); }
}

/* Instance name text - smooth opacity transition */
.instance-name-text {
  color: var(--text-primary);
  transition: opacity 0.25s ease, font-size 0.15s ease;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* Instance card - compact padding to match other cards */
.instance-card {
  padding: 6px var(--stat-padding-x);
  display: flex;
  flex-direction: column;
  justify-content: center;
  overflow: hidden;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease;
}
.instance-card:hover {
  transform: scale(1.06);
  z-index: 10;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

/* Drum picker - iOS style sliding */
.drum-viewport {
  position: relative;
  height: 78px;
  overflow: hidden;
  mask-image: linear-gradient(to bottom, transparent 0%, black 20%, black 80%, transparent 100%);
  -webkit-mask-image: linear-gradient(to bottom, transparent 0%, black 20%, black 80%, transparent 100%);
}
.drum-track {
  transition: transform 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
  padding-top: 26px;
}
.drum-item {
  display: flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding: 0 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
  white-space: nowrap;
}
.drum-item:hover {
  background: var(--muted);
}
.drum-item-active {
  background: var(--surface);
  border: 1px solid var(--border);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  padding: 0 10px;
}
.drum-highlight {
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: 26px;
  transform: translateY(-50%);
  border-radius: 6px;
  border: 1px solid color-mix(in srgb, var(--accent) 15%, transparent);
  background: color-mix(in srgb, var(--accent) 4%, transparent);
  pointer-events: none;
}
.instance-type-badge-dim {
  opacity: 0.5;
}

/* Instance selector */
.instance-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 9999px;
  cursor: pointer;
  transition: all var(--transition-normal);
  white-space: nowrap;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-primary);
}
.instance-pill:hover {
  border-color: color-mix(in srgb, var(--accent) 30%, transparent);
  box-shadow: var(--card-shadow-hover);
  transform: translateY(-1px);
}
.instance-pill-active {
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
  background: var(--accent-soft);
}
.instance-pill-active .instance-type-badge {
  background: color-mix(in srgb, var(--accent) 15%, transparent);
  color: var(--accent);
}
.instance-type-badge {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.02em;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--muted);
  color: var(--text-secondary);
  text-transform: uppercase;
}

/* Refresh ring animation */
.refresh-ring {
  transform: rotate(-90deg);
}

.refresh-ring-progress {
  transition: stroke-dashoffset 1s linear;
}

/* Process table rows */
.process-row-danger {
  background: var(--danger-soft);
}

.process-row-normal:hover {
  background: color-mix(in srgb, var(--accent) 5%, transparent);
}

/* Fade-up animation with stagger */
.fade-up {
  animation: fadeUp 0.3s cubic-bezier(0.16, 1, 0.3, 1) both;
}

@keyframes fadeUp {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Content fading during instance switch */
.content-fading {
  opacity: 0.35;
  transition: opacity 0.2s ease;
  pointer-events: none;
}

/* Detail cards transition - out-in mode */
.detail-cards-leave-active {
  transition: opacity 0.15s ease-in, transform 0.15s ease-in;
}
.detail-cards-leave-to {
  opacity: 0;
  transform: scale(0.97);
}
.detail-cards-enter-active {
  transition: opacity 0.1s ease;
}
.detail-cards-enter-from {
  opacity: 0;
}
.detail-cards-enter-active .content-card {
  animation: cardSlideIn 0.35s cubic-bezier(0.16, 1, 0.3, 1) both;
}
.detail-cards-enter-active .content-card:nth-child(1) { animation-delay: 0ms; }
.detail-cards-enter-active .content-card:nth-child(2) { animation-delay: 60ms; }
.detail-cards-enter-active .content-card:nth-child(3) { animation-delay: 120ms; }
.detail-cards-enter-active .content-card:nth-child(4) { animation-delay: 180ms; }
.detail-cards-enter-active .content-card:nth-child(5) { animation-delay: 240ms; }

@keyframes cardSlideIn {
  from {
    opacity: 0;
    transform: translateX(-20px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}
</style>
