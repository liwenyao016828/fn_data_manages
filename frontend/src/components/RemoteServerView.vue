<template>
  <div class="page-padding h-full overflow-y-auto">
    <!-- Page Header -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h2 class="text-[15px] font-semibold" style="color: var(--text-primary)">远程服务器</h2>
        <p class="text-[12px] mt-0.5" style="color: var(--text-tertiary)">管理所有远程数据库服务器连接</p>
      </div>
      <Button variant="primary" size="sm" class="h-[32px] text-[13px]" @click="handleAdd">
        <Plus class="h-3.5 w-3.5 mr-1.5" />添加远程服务器
      </Button>
    </div>

    <!-- Server Cards Grid -->
    <div
      v-if="serverList.length > 0"
      class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 grid-gap"
    >
      <div
        v-for="(row, idx) in serverList"
        :key="row.id"
        class="server-card group"
        :style="{ '--type-color': getTypeColor(row.type), '--type-soft': getTypeSoftColor(row.type), '--card-stagger': `${idx * 45}ms` }"
      >
        <!-- Top accent bar (revealed on hover) -->
        <span class="server-card-accent" aria-hidden="true" />

        <!-- Card Body -->
        <div class="server-card-body">
          <!-- Header: Type Icon + Name + Status -->
          <div class="server-card-head">
            <div class="server-icon">
              <component :is="getTypeIcon(row.type)" class="h-[18px] w-[18px]" />
            </div>
            <div class="server-head-text">
              <div class="server-name-row">
                <span class="server-name">{{ row.name }}</span>
                <StatusDot :status="serverStatus(row)" size="xs" />
                <span v-if="row.ssl" class="server-ssl-badge" title="SSL 已启用">
                  <ShieldCheck class="h-3 w-3" />
                </span>
              </div>
              <div class="server-meta-line">
                <span class="server-type-text">{{ getTypeLabel(row.type) }}</span>
                <span class="server-meta-sep">·</span>
                <span class="server-status-text" :data-status="serverStatus(row)">{{ statusLabel(row) }}</span>
              </div>
            </div>
          </div>

          <!-- Connection info grid -->
          <div class="server-info-grid">
            <div class="server-info-item">
              <span class="server-info-label">地址</span>
              <span class="server-info-value font-mono-data">{{ row.host }}:{{ row.port }}</span>
            </div>
            <div class="server-info-item">
              <span class="server-info-label">用户</span>
              <span class="server-info-value font-mono-data">{{ row.username || '—' }}</span>
            </div>
          </div>

          <!-- Description -->
          <p v-if="row.description" class="server-description">{{ row.description }}</p>
          <p v-else class="server-description server-description--empty">未填写描述</p>

          <!-- Footer -->
          <div class="server-card-footer">
            <div class="server-footer-meta">
              <span v-if="row.version" class="server-footer-chip" :title="`数据库版本 ${row.version}`">
                <Tag class="h-3 w-3" />{{ row.version }}
              </span>
              <span v-else class="server-footer-chip server-footer-chip--muted">
                <Tag class="h-3 w-3" />版本未获取
              </span>
              <span v-if="row.disk" class="server-footer-chip" :title="`数据占用 ${row.disk}`">
                <HardDrive class="h-3 w-3" />{{ row.disk }}
              </span>
            </div>
            <div class="server-actions" @click.stop>
              <button
                class="server-action"
                :class="{ 'server-action--loading': testingId === row.id }"
                :disabled="testingId === row.id"
                title="测试连接"
                @click="handleTest(row)"
              >
                <Loader2 v-if="testingId === row.id" class="h-3.5 w-3.5 animate-spin" />
                <Zap v-else class="h-3.5 w-3.5" />
              </button>
              <button
                class="server-action"
                title="编辑"
                @click="handleEdit(row)"
              >
                <Pencil class="h-3.5 w-3.5" />
              </button>
              <button
                class="server-action server-action--danger"
                title="删除"
                @click="openDeleteDialog(row)"
              >
                <Trash2 class="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state mt-16 fade-up">
      <div class="empty-state-icon">
        <Inbox class="h-12 w-12" />
      </div>
      <div class="empty-state-text">暂无远程服务器</div>
      <Button variant="primary" size="sm" class="h-[32px] text-[13px] mt-4" @click="handleAdd">
        <Plus class="h-3.5 w-3.5 mr-1.5" />添加远程服务器
      </Button>
    </div>

    <RemoteServerDialog v-model="dialogVisible" :type="dialogType" :data="currentServer" @success="loadServers" />

    <Dialog v-model:open="showDeleteDialog">
      <DialogContent class="sm:max-w-[425px] rounded-xl">
        <div class="flex flex-col gap-y-1.5 text-center sm:text-left">
          <DialogTitle class="text-[15px]">删除确认</DialogTitle>
          <DialogDescription class="text-[13px]" style="color: var(--text-tertiary)">
            确定要删除服务器 "{{ deleteTarget?.name }}" 吗？此操作不可恢复。
          </DialogDescription>
        </div>
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" class="h-[32px] text-[13px]" @click="showDeleteDialog = false">取消</Button>
          <Button variant="destructive" class="h-[32px] text-[13px]" @click="confirmDelete">确定删除</Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onActivated } from 'vue'
import { storeToRefs } from 'pinia'
import { useMessage } from '../composables/useMessage'
import {
  Plus, Inbox, Zap, Pencil, Trash2, Loader2, ShieldCheck,
  Database, Server, Tag, HardDrive,
} from 'lucide-vue-next'
import RemoteServerDialog from './RemoteServerDialog.vue'
import { Button } from '@/components/ui/Button.vue'
import { Badge } from '@/components/ui/Badge.vue'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table.vue'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/Dialog.vue'
import { useHealthStore } from '../stores/health'
import { getTypeLabel, getTypeColor, getTypeSoftColor } from '@/lib/utils'
import StatusDot from './StatusDot.vue'

const { success, error, warning } = useMessage()
const healthStore = useHealthStore()
const { statusMap } = storeToRefs(healthStore)

const serverList = ref([])
const previousIds = ref(new Set())
const dialogVisible = ref(false)
const dialogType = ref('create')
const currentServer = ref(null)
const showDeleteDialog = ref(false)
const deleteTarget = ref(null)
const testingId = ref(null)

const TYPE_ICONS = {
  mysql: Database,
  mariadb: Database,
  postgresql: Database,
  redis: Server,
  sqlite: Database,
}

const getTypeIcon = (type) => TYPE_ICONS[type] || Database

const uidOf = (row) => 'r:' + row.id

const serverStatus = (row) => {
  const status = statusMap.value?.[uidOf(row)]
  if (status === true) return 'online'
  if (status === false) return 'offline'
  return 'default'
}

const statusLabel = (row) => {
  const s = serverStatus(row)
  if (s === 'online') return '在线'
  if (s === 'offline') return '离线'
  return '未检测'
}

const loadServers = () => {
  fetch('/api/remote-servers')
    .then(res => res.json())
    .then(data => {
      if (data.code === 0) {
        const newList = data.data || []
        const newIds = new Set(newList.map(s => 'r:' + s.id))
        const newServerIds = [...newIds].filter(id => !previousIds.value.has(id))
        serverList.value = newList
        previousIds.value = newIds
        healthStore.cleanup([...newIds])
        newServerIds.forEach(id => {
          healthStore.forceCheckOne(id)
        })
      }
    })
    .catch((e) => { console.error(e) })
}

const handleAdd = () => { dialogType.value = 'create'; currentServer.value = null; dialogVisible.value = true }
const handleEdit = (row) => { dialogType.value = 'edit'; currentServer.value = { ...row }; dialogVisible.value = true }

const handleTest = (row) => {
  if (row.password === '••••••••') {
    warning('密码为掩码，请先编辑修改密码后再测试连接')
    return
  }
  testingId.value = row.id
  fetch('/api/remote-servers/test', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ host: row.host, port: row.port, username: row.username, password: row.password, type: row.type }) })
    .then(res => res.json())
    .then(data => { if (data.code === 0) { success('连接成功') } else { error(data.msg || '连接失败') } })
    .catch(() => { error('连接失败') })
    .finally(() => { if (testingId.value === row.id) testingId.value = null })
}

const openDeleteDialog = (row) => { deleteTarget.value = row; showDeleteDialog.value = true }

const confirmDelete = () => {
  if (!deleteTarget.value) return
  fetch(`/api/remote-servers/${deleteTarget.value.id}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(data => { if (data.code === 0) { success('删除成功'); loadServers() } else { error(data.msg || '删除失败') } })
    .catch(() => { error('删除失败') })
    .finally(() => { showDeleteDialog.value = false })
}

onMounted(() => { loadServers() })
onActivated(() => { loadServers() })
</script>

<style scoped>
/* ═══════════════════════════════════════════════════════════════
   Remote Server Card — layered, type-colored, animated
   ═══════════════════════════════════════════════════════════════ */
.server-card {
  position: relative;
  border-radius: var(--card-radius);
  border: var(--card-border);
  background: var(--surface);
  box-shadow: var(--card-shadow);
  overflow: hidden;
  transition:
    transform 280ms cubic-bezier(0.16, 1, 0.3, 1),
    box-shadow 280ms cubic-bezier(0.16, 1, 0.3, 1),
    border-color 220ms ease;
  cursor: default;
  animation: serverCardEnter 480ms cubic-bezier(0.16, 1, 0.3, 1) both;
  animation-delay: var(--card-stagger, 0ms);
  will-change: transform, box-shadow;
}

.server-card::before {
  /* soft type-colored ambient wash revealed on hover */
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(
    420px circle at 0% 0%,
    color-mix(in srgb, var(--type-color, var(--accent)) 12%, transparent),
    transparent 60%
  );
  opacity: 0;
  transition: opacity 320ms ease;
}

.server-card:hover {
  transform: translateY(-3px);
  border-color: color-mix(in srgb, var(--type-color, var(--accent)) 32%, var(--border));
  box-shadow:
    var(--card-shadow-hover),
    0 8px 24px color-mix(in srgb, var(--type-color, var(--accent)) 14%, transparent);
}

.server-card:hover::before { opacity: 1; }

.server-card:active {
  transform: translateY(-1px);
  box-shadow: var(--card-shadow);
}

@keyframes serverCardEnter {
  from { opacity: 0; transform: translateY(14px) scale(0.985); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}

/* Top accent bar — reveals on hover */
.server-card-accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    var(--type-color, var(--accent)) 50%,
    transparent 100%
  );
  transform: scaleX(0.15);
  transform-origin: center;
  opacity: 0.55;
  transition: transform 420ms cubic-bezier(0.16, 1, 0.3, 1), opacity 280ms ease;
  pointer-events: none;
}

.server-card:hover .server-card-accent {
  transform: scaleX(1);
  opacity: 1;
}

.server-card-body {
  position: relative;
  padding: 1rem 1.125rem 0.875rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* ── Header ────────────────────────────────────────────────── */
.server-card-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.server-icon {
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.625rem;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #fff;
  background: linear-gradient(
    135deg,
    var(--type-color, var(--accent)) 0%,
    color-mix(in srgb, var(--type-color, var(--accent)) 70%, #000 0%) 100%
  );
  box-shadow:
    0 4px 10px color-mix(in srgb, var(--type-color, var(--accent)) 30%, transparent),
    inset 0 1px 0 rgba(255, 255, 255, 0.18);
  transition: transform 320ms cubic-bezier(0.16, 1, 0.3, 1), box-shadow 280ms ease;
}

.server-card:hover .server-icon {
  transform: rotate(-4deg) scale(1.04);
  box-shadow:
    0 6px 16px color-mix(in srgb, var(--type-color, var(--accent)) 38%, transparent),
    inset 0 1px 0 rgba(255, 255, 255, 0.22);
}

.server-head-text {
  flex: 1;
  min-width: 0;
}

.server-name-row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}

.server-name {
  font-size: 0.875rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 0 1 auto;
}

.server-ssl-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.125rem;
  height: 1.125rem;
  border-radius: 0.3125rem;
  color: var(--success);
  background: var(--success-soft);
  flex-shrink: 0;
  transition: transform 220ms ease;
}

.server-card:hover .server-ssl-badge {
  transform: scale(1.08);
}

.server-meta-line {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.125rem;
  font-size: 0.6875rem;
  color: var(--text-tertiary);
  letter-spacing: 0.01em;
}

.server-type-text {
  color: var(--type-color, var(--accent));
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-size: 0.625rem;
}

.server-meta-sep {
  color: var(--border-strong);
  font-weight: 400;
}

.server-status-text {
  font-variant-numeric: tabular-nums;
  transition: color 200ms ease;
}

.server-status-text[data-status='online']  { color: var(--success); }
.server-status-text[data-status='offline'] { color: var(--danger); }

/* ── Info Grid ─────────────────────────────────────────────── */
.server-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  border-radius: 0.625rem;
  background: color-mix(in srgb, var(--surface) 100%, transparent);
  border: 1px solid var(--border-subtle);
  position: relative;
  overflow: hidden;
}

.server-info-grid::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--type-color, var(--accent)) 4%, transparent),
    transparent 60%
  );
  pointer-events: none;
}

.server-info-item {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.server-info-label {
  font-size: 0.625rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-tertiary);
  font-weight: 500;
}

.server-info-value {
  font-size: 0.8125rem;
  color: var(--text-primary);
  font-weight: 500;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Description ───────────────────────────────────────────── */
.server-description {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 2.25rem;
}

.server-description--empty {
  color: var(--text-tertiary);
  font-style: italic;
  opacity: 0.7;
}

/* ── Footer ────────────────────────────────────────────────── */
.server-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding-top: 0.625rem;
  margin-top: auto;
  border-top: 1px dashed var(--border-subtle);
}

.server-footer-meta {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
  min-width: 0;
  flex: 1;
}

.server-footer-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  height: 1.375rem;
  padding: 0 0.5rem;
  border-radius: 999px;
  background: var(--muted);
  color: var(--text-secondary);
  font-size: 0.6875rem;
  font-weight: 500;
  border: 1px solid var(--border-subtle);
  white-space: nowrap;
  transition: all 200ms ease;
}

.server-footer-chip--muted {
  color: var(--text-tertiary);
  font-style: italic;
}

.server-card:hover .server-footer-chip:not(.server-footer-chip--muted) {
  border-color: color-mix(in srgb, var(--type-color, var(--accent)) 25%, transparent);
  color: var(--text-primary);
}

/* ── Action group ──────────────────────────────────────────── */
.server-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.125rem;
  padding: 0.1875rem;
  border-radius: 999px;
  background: var(--muted);
  border: 1px solid var(--border-subtle);
  opacity: 0;
  transform: translateX(4px);
  transition: opacity 240ms ease, transform 280ms cubic-bezier(0.16, 1, 0.3, 1);
}

.server-card:hover .server-actions,
.server-card:focus-within .server-actions {
  opacity: 1;
  transform: translateX(0);
}

.server-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.625rem;
  height: 1.625rem;
  border-radius: 999px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 200ms cubic-bezier(0.16, 1, 0.3, 1);
  position: relative;
}

.server-action:hover {
  background: var(--surface);
  color: var(--accent);
  transform: scale(1.08);
}

.server-action:active {
  transform: scale(0.92);
}

.server-action:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.server-action:disabled {
  cursor: wait;
  opacity: 0.6;
}

.server-action--loading {
  color: var(--accent);
  background: var(--accent-soft);
}

.server-action--danger:hover {
  color: var(--danger);
  background: var(--danger-soft);
}

/* ── Responsive ────────────────────────────────────────────── */
@media (max-width: 640px) {
  .server-card-body { padding: 0.875rem 0.875rem 0.75rem; gap: 0.625rem; }
  .server-info-grid { grid-template-columns: 1fr; }
  .server-actions { opacity: 1; transform: translateX(0); }
}

/* ── Reduced motion ────────────────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .server-card,
  .server-card-accent,
  .server-icon,
  .server-actions,
  .server-action,
  .server-footer-chip {
    animation: none !important;
    transition: none !important;
  }
  .server-card:hover { transform: none; }
}
</style>
