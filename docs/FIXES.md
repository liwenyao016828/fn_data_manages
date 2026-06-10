# 代码审查与修复记录

> 本文档记录 `data_manages` 项目全量代码审查（2026-06-08）发现的 18 项问题及修复方案。
> 修复原则：内网使用、逐项修改、确保不影响其他功能。

## 风险等级说明

- **critical**：安全/数据风险，必须修复
- **major**：健壮性/可维护性问题
- **minor**：风格/清理/低风险

## 修复进度总览

| No. | 等级 | 标题 | 状态 | 涉及文件 |
|---|---|---|---|---|
| 1 | critical | 加密密钥硬编码 + 密文识别 | ✅ | server/crypto.go |
| 2 | critical | Docker exec 容器名/路径校验 | ✅ | server/util.go + handler_mysql.go |
| 3 | critical | fsBrowse 任意目录读取 | ✅ | server/util.go |
| 4 | critical | 完全无鉴权 + CORS \* | ⏸️ 内网暂缓 | server/main.go |
| 5 | critical | mysqlExecute 黑名单可绕过 | ✅ | server/db_safe.go + handler_mysql.go |
| 6 | major | DDL/授权 fmt.Sprintf 拼接 | ✅ | server/handler_mysql.go + db_safe.go |
| 7 | major | 全局 mutex 拆分 | ✅ | server/model.go, store.go, handler_scheduled.go |
| 8 | major | SQLite DSN 注入 | ✅ | server/connection.go |
| 9 | major | 调度备份持锁 | ✅ | server/handler_scheduled.go |
| 10 | major | atomicWriteFile 非原子 | ✅ | server/store.go |
| 11 | major | App.vue 监听器泄漏 | ✅ | frontend/App.vue |
| 12 | major | DashboardView 自动翻卡片 | ✅ | frontend/components/DashboardView.vue |
| 13 | major | window.confirm 阻塞 | ✅ | frontend/components/DataManageView.vue |
| 14 | major | 前端 DANGEROUS_SQL_PATTERNS | ✅ | frontend/components/DataManageView.vue |
| 15 | major | localStorage 键名混乱 | ✅ | frontend 全局 |
| 16 | minor | 健康轮询硬编码 15s | ✅ | frontend/App.vue + lib/storageKeys.js |
| 17 | minor | DataManageView 拆分 | ✅ | frontend/components/SqlConsoleTab.vue（新增） |
| 18 | minor | FLUSH PRIVILEGES 错误吞 | ✅ | server/handler_mysql.go |
| 19 | extra | PostgreSQL 用户密码 SQL 拼接 | ✅ | server/handler_postgresql.go + db_safe.go |
| 20 | extra | PostgreSQL 备份约束查询 SQL 注入 + 死代码 | ✅ | server/handler_backup.go |
| 21 | extra | 日志路径白名单（避免读敏感文件） | ✅ | server/util.go |
| 22 | extra | fmt.Errorf 非格式化字符串 | ✅ | server/connection.go |
| ~~23~~ | ~~revert~~ | ~~hostPort IPv6 改造 — 内网项目不需要，已回滚~~ | ❌ | — |
| 24 | extra | loadData JSON 解析失败导致数据丢失 | ✅ | server/store.go |
| 25 | extra | Redis 命中率除零风险 | ✅ | frontend/components/DataManageView.vue |

> #4 鉴权问题项目内网使用且需要配合前端联调，单独排期。

---

## 详细修复方案

### #1 加密密钥强制 + 密文前缀识别
**文件**：[server/crypto.go](server/crypto.go)
**改动**：
1. 启动时若 `DM_ENCRYPT_KEY` 未设置，打印强警告日志并写系统日志，但不强制退出（向后兼容）
2. `encryptPassword` 输出加 `enc:v1:` 前缀
3. `decryptPassword` 先看前缀，无前缀视为明文（兼容老数据）
4. `isPasswordEncrypted` 改为检查前缀
5. `rand.Read` 错误返回 error

### #2 Docker exec 容器名/路径校验
**文件**：[server/util.go](server/util.go)
**改动**：
1. 新增 `validateContainerName(name) error` 用 `^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$` 校验
2. `readContainerFile` / `readLocalOrContainerFile` 入口先校验容器名
3. 文件路径限制在 `mysqlDataDir` 子树内（绝对路径检查 + `..` 拦截）

### #3 fsBrowseHandler 路径白名单
**文件**：[server/util.go](server/util.go)
**改动**：
1. 接受一组白名单根目录（如 `./data`、`$TRIM_PKGVAR`、用户选定的根）
2. 解析后的 `reqPath` 必须在某个白名单根目录下（用 `EvalSymlinks` 后 `HasPrefix` 判断）
3. 保留自动 `MkdirAll`（仅在白名单内），删除根目录外创建行为

### #5 mysqlExecute 黑名单 → 白名单
**文件**：[server/handler_mysql.go](server/handler_mysql.go:572-586)
**改动**：
1. 把 `dangerousKeywords` 黑名单替换为 `allowedKeywords` 白名单
2. 解析 SQL 时跳过注释与字符串字面量，避免误判
3. 第一个非空 token 不在白名单 → 拒绝

### #6 DDL/授权严格校验
**文件**：[server/handler_mysql.go](server/handler_mysql.go) + [server/db_safe.go](server/db_safe.go)
**改动**：
1. 新增 `validatePrivilegeList` 严格按 MySQL 权限枚举
2. 表名/库名/列名走 `validateIdentifier`（已有）
3. 涉及 `SET GLOBAL var = val` 的地方 `val` 必须 `^[0-9]+$` 或 `'string'`
4. 把现有 30+ 处 `fmt.Sprintf` 拼 SQL 的地方系统性替换

### #7 全局 mutex 拆分
**文件**：[server/model.go](server/model.go) + 全部使用方
**改动**：
1. 把一把 `mutex` 拆为 `dbMu/remoteMu/backupMu/schedMu/metricsMu`
2. `saveData` 拷快照 → 释放锁 → 写盘
3. 长时间 I/O（`os.Stat`/`migrateMetricsHistory`）放锁外

### #8 SQLite DSN 路径校验
**文件**：[server/connection.go](server/connection.go:242-261)
**改动**：
1. `openSQLite` 入口拒绝 Host 含 `? # \ : * ? " < > |` 的字符
2. 改用 `url.Values` 拼查询参数（保险）

### #9 调度备份释放锁后执行
**文件**：[server/handler_scheduled.go](server/handler_scheduled.go:279-300)
**改动**：
1. 持锁只做"检查是否到期 + 标记开始 + 拷条目"
2. 释放锁后 `exec.Command("mysqldump", ...)`
3. 重新加锁更新 `LastRun`/`LastResult`

### #10 atomicWriteFile Windows 原子性
**文件**：[server/store.go](server/store.go:52-61)
**改动**：
1. 增加 file lock 防止并发写
2. 临时文件名加 `os.Getpid()` 避免冲突
3. `os.Remove` 失败仅当目标存在时记录 warning

### #11 App.vue 监听器清理
**文件**：[frontend/src/App.vue](frontend/src/App.vue:42-54)
**改动**：
```js
const sidebarHandler = () => { sidebarCollapsed.value = !sidebarCollapsed.value }
const compactHandler = (e) => { compactMode.value = e.detail }
onMounted(() => {
  window.addEventListener('toggle-sidebar', sidebarHandler)
  window.addEventListener('compact-mode-change', compactHandler)
  healthStore.startPolling(15000)
})
onUnmounted(() => {
  window.removeEventListener('toggle-sidebar', sidebarHandler)
  window.removeEventListener('compact-mode-change', compactHandler)
  healthStore.stopPolling()
})
```

### #12 DashboardView 移除自动翻卡片
**文件**：[frontend/src/components/DashboardView.vue](frontend/src/components/DashboardView.vue:1389-1391)
**改动**：删除 `fetchMetrics` 里的 `flipCard('uptime')` 与 `flipCard('qps')`。

### #13 window.confirm → showConfirm
**文件**：[frontend/src/components/DataManageView.vue](frontend/src/components/DataManageView.vue:1423)
**改动**：
- 检查 `useMessage` 是否提供 `showConfirm` 风格的弹窗
- 已有则替换；没有则封装一个 Promise-based confirm

### #14 前端 SQL 黑名单注释化
**文件**：[frontend/src/components/DataManageView.vue](frontend/src/components/DataManageView.vue:1411-1416)
**改动**：
- 保留黑名单仅作为"风险提示 + 二次确认"
- 加注释说明"后端有独立校验"

### #15 localStorage 键名统一
**文件**：全部 `frontend/src/`
**改动**：
1. 新增 `frontend/src/lib/storageKeys.js` 导出所有 key
2. 全局替换旧 key
3. 加一次性迁移函数：读旧 key → 写到新 key → 删旧 key

### #16 健康轮询读取配置
**文件**：[frontend/src/App.vue](frontend/src/App.vue:49)
**改动**：
- 改为 `healthStore.startPolling(healthConfig.value.intervalSec * 1000)`
- 或在 health store 内根据 `healthConfig` 自动启停

### #17 DataManageView 拆分（最小化）
**文件**：[frontend/src/components/DataManageView.vue](frontend/src/components/DataManageView.vue)
**改动**：仅抽出 `<RedisKeysPanel>` `<SqlConsolePanel>` 作为子组件，主文件用 `<component :is>` 引用
- 注：完全拆分工作量大，仅做最小拆分

### #18 FLUSH PRIVILEGES 错误日志化
**文件**：[server/handler_mysql.go](server/handler_mysql.go)
**改动**：把 `_, _ = db.Exec("FLUSH PRIVILEGES")` 全部改为 `if _, err := ...; err != nil { sysLogWarn("MYSQL", "FLUSH PRIVILEGES failed: " + err.Error()) }`

---

## 验证策略

每修一项必须：
1. Go 文件：`go build ./...` 通过
2. Vue 文件：`npm run lint`（若配置）+ 浏览器控制台无新增报错
3. 涉及存储的：写一个测试用值，刷新页面后能读回
4. 涉及 API 的：用 curl 或 Postman 调用一次确认行为没变

## 修复顺序（按风险从低到高）

1. #11, #12（前端小改，影响范围最小）
2. #18（后端小改，错误处理替换）
3. #16（前端，配置化）
4. #14（前端，注释化）
5. #13（前端小改）
6. #8（后端，加字符检查）
7. #10（后端，加 lock）
8. #9（后端，调度备份重构）
9. #7（后端，mutex 拆分）
10. #6（后端，多处 SQL 重写）
11. #5（后端，mysqlExecute 改白名单）
12. #3（后端，fsBrowse 白名单）
13. #2（后端，docker exec 校验）
14. #1（后端，加密体系）
15. #15（前端，重命名 + 迁移）
16. #17（前端，拆分）
