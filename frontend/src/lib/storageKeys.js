// 集中管理所有 localStorage 键
// 新格式：niudb.<scope>.<key>
// 老格式（兼容读取，迁移后会删除）

export const STORAGE_KEYS = {
  THEME: 'niudb.theme.value',
  THEME_MODE: 'niudb.theme.mode',
  CONTEXT_CURRENT: 'niudb.context.current',
  CONTEXT_FAVORITES: 'niudb.context.favorites',
  LOG_CONFIG: 'niudb.log.config',
  DASHBOARD_TIME_RANGE: 'niudb.dashboard.timeRange',
  DASHBOARD_REFRESH_INTERVAL: 'niudb.dashboard.refreshInterval',
  HEALTH_POLL_INTERVAL_MS: 'niudb.health.pollIntervalMs',
  UI_COMPACT_MODE: 'niudb.ui.compactMode',
  UI_BACKUP_RETENTION_DAYS: 'niudb.ui.backupRetentionDays',
  SETTINGS_MAIN_TAB: 'niudb.settings.mainTab',
  SETTINGS_SYSTEM_SUB_TAB: 'niudb.settings.systemSubTab',
  SETTINGS_INSTANCE_TAB: 'niudb.settings.instanceTab',
  SETTINGS_DETAIL_TAB: 'niudb.settings.detailTab',
  BACKUP_FILTERS: 'niudb.backup.filters'
}

// 老 key 映射到新 key（用于一次性迁移）
const LEGACY_KEYS = {
  'db_manager_theme': STORAGE_KEYS.THEME,
  'db_manager_theme_mode': STORAGE_KEYS.THEME_MODE,
  'current_context': STORAGE_KEYS.CONTEXT_CURRENT,
  'preferred_connections': STORAGE_KEYS.CONTEXT_FAVORITES,
  'log_config': STORAGE_KEYS.LOG_CONFIG,
  'dashboard_time_range': STORAGE_KEYS.DASHBOARD_TIME_RANGE,
  'refreshInterval': STORAGE_KEYS.DASHBOARD_REFRESH_INTERVAL,
  'compactMode': STORAGE_KEYS.UI_COMPACT_MODE,
  'backupRetentionDays': STORAGE_KEYS.UI_BACKUP_RETENTION_DAYS,
  'settingsMainTab': STORAGE_KEYS.SETTINGS_MAIN_TAB,
  'settingsSystemSubTab': STORAGE_KEYS.SETTINGS_SYSTEM_SUB_TAB,
  'settingsInstanceTab': STORAGE_KEYS.SETTINGS_INSTANCE_TAB,
  'settingsDetailTab': STORAGE_KEYS.SETTINGS_DETAIL_TAB,
  'backup_view_filters': STORAGE_KEYS.BACKUP_FILTERS
}

const MIGRATION_FLAG = 'niudb.migration.v1.done'

// 一次性迁移：在新值不存在时把老 key 的值复制过来，并删除老 key
export function migrateLocalStorage() {
  try {
    if (typeof localStorage === 'undefined') return
    if (localStorage.getItem(MIGRATION_FLAG) === '1') return

    for (const [oldKey, newKey] of Object.entries(LEGACY_KEYS)) {
      const oldVal = localStorage.getItem(oldKey)
      if (oldVal === null) continue
      if (localStorage.getItem(newKey) === null) {
        localStorage.setItem(newKey, oldVal)
      }
      localStorage.removeItem(oldKey)
    }
    localStorage.setItem(MIGRATION_FLAG, '1')
  } catch (e) {
    // 隐私模式可能抛错，忽略
    console.warn('[storage] migration skipped:', e.message)
  }
}

// 带 try/catch 的 localStorage 访问工具
export const safeStorage = {
  get(key) {
    try { return localStorage.getItem(key) } catch (e) { return null }
  },
  set(key, value) {
    try { localStorage.setItem(key, value); return true } catch (e) { return false }
  },
  remove(key) {
    try { localStorage.removeItem(key); return true } catch (e) { return false }
  }
}
