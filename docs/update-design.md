# niudb 应用内自动更新 — 设计文档

> 状态：设计稿（待评审）
> 目标：在不依赖 fnOS 应用中心手动操作的前提下，让运行中的 `niudb` 应用能够：① 周期 / 手动检查新版本；② 在前端看到更新提示和进度；③ 在用户授权后由应用自管完成下载、校验与安装；④ 安装完成后由 fnOS 框架重启服务。

---

## 1. 背景与现状

### 1.1 现状
- 当前 [fpk/manifest](../../fpk/manifest) 中 `version=1.0.0` 是写死的。
- [fpk/cmd/upgrade_init](../../fpk/cmd/upgrade_init)、[fpk/cmd/upgrade_callback](../../fpk/cmd/upgrade_callback) 是空脚本（`exit 0`），与参考项目 [kci-lnk/fn-knock-turborepo](https://github.com/kci-lnk/fn-knock-turborepo) 行为一致。
- [fpk/cmd/main](../../fpk/cmd/main) 已经实现了 stop/status/start 子命令，但缺少 upgrade 入口。
- 整个项目没有"检查更新"链路，更新只能由用户去 fnOS 应用中心手动操作。

### 1.2 目标
1. **远端清单** (`latest.json`) 描述最新版本与下载资源。
2. **后端**定时 + 按需检查，解析清单后保存状态，向前端暴露 REST API。
3. **前端**在 Settings 页提供"检查更新"卡片，展示版本号、release notes、下载进度、安装结果。
4. **后端**接到前端"开始更新"指令后，下载 `.fpk`、校验 SHA256、调用 fnOS `appcenter-cli`（或调用 `cmd/main` stop 后再触发安装），让框架走标准 stop → install → start 流程。
5. **构建脚本**在打包时把 Go 编译期版本号同步到 `manifest` 的 `version=` 和 `latest.json` 的 `version=`，避免多源版本不一致。

### 1.3 非目标
- 不替代 fnOS 应用中心的官方升级通道，只是补充。
- 不实现差分 / 增量更新，每次都是整包。
- 不做多节点灰度 / A-B 测试。

---

## 2. 架构概览

```
┌──────────────────────────────────────────────────────────────────────┐
│                        niudb (fpk)                                   │
│                                                                      │
│   ┌─────────────┐    /api/update/*     ┌──────────────────────────┐   │
│   │  Vue 前端   │ ◀──────────────────▶ │  Go 后端 (server)        │   │
│   │ SettingsView│                      │  handler_update.go       │   │
│   └──────▲──────┘   WebSocket 进度     │  + update.Manager        │   │
│          │                              └─────────┬────────────────┘   │
│          │                                        │                    │
│          │ SSE / WS 进度                           │ 调用                │
│          │                                        ▼                    │
│   ┌──────┴──────────┐                  ┌────────────────────────┐     │
│   │ MessageToast    │                  │ appcenter-cli upgrade  │     │
│   └─────────────────┘                  │ 或: cmd/main stop +    │     │
│                                        │  install .fpk          │     │
│                                        └────────────┬───────────┘     │
│                                                     │                  │
└─────────────────────────────────────────────────────┼──────────────────┘
                                                      ▼
                                fnOS 框架 (stop → install_callback → start)
                                                      │
                                                      ▼
                                  重新拉起 ${TRIM_APPDEST}/server

                远端：HTTPS CDN
                ┌──────────────────────────────┐
                │  https://cdn.example.com/    │
                │  niudb/latest.json           │
                │  niudb/niudb-1.2.0-amd64.fpk │
                │  niudb/niudb-1.2.0-arm64.fpk │
                └──────────────────────────────┘
```

---

## 3. 数据结构

### 3.1 远端清单 `latest.json`

托管在公开 HTTPS 端（CDN / GitHub Release / 自有对象存储）。最小字段：

```json
{
  "version": "1.2.0",
  "update_available": true,
  "force_update": false,
  "release_notes": "## 1.2.0\n- 新增 xxx\n- 修复 yyy\n",
  "published_at": "2026-06-09T10:00:00Z",
  "min_supported_version": "1.0.0",
  "packages": {
    "amd64": {
      "download_url": "https://cdn.example.com/niudb/niudb-1.2.0-amd64.fpk",
      "sha256": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
      "size": 25165824
    },
    "arm64": {
      "download_url": "https://cdn.example.com/niudb/niudb-1.2.0-arm64.fpk",
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "size": 24867200
    }
  }
}
```

> 字段命名参考 fn-knock 项目的 `OtaLatestManifest`，并扩展 `packages` 子对象以同时承载两架构信息。

### 3.2 Go 内部状态

放在 `server/internal/update/state.go`，进程内单例；如需多实例协同可替换为 `TRIM_PKGVAR/update-state.json` 文件 + 文件锁。

```go
type State struct {
    LocalVersion   string    // 编译时注入
    Latest         *Manifest // 最近一次成功拉取的清单
    LastCheckedAt  time.Time
    CheckError     string
    Download       DownloadState
}

type DownloadState struct {
    Status          string  // idle|downloading|verifying|downloaded|installing|error|done
    Percent         int
    DownloadedBytes int64
    TotalBytes      int64
    Error           string
    TargetVersion   string
    TargetArch      string  // amd64|arm64
    TargetPath      string  // /tmp/niudb-updates/niudb-1.2.0-amd64.fpk
}
```

### 3.3 Redis / SQLite 持久化（可选）

如果后端有 Redis（fn-knock 模式）或 SQLite（niudb 现有 `data/served.js` 风格），可以存：
- `update:pending` — 已发起安装、等待框架回调确认。
- `update:confirm` — 启动后比对 `LocalVersion` 写回，用于前端弹"更新成功"提示。

无 Redis 时退化到文件 `update-state.json` 即可。

---

## 4. HTTP API 设计

所有接口挂 `/api/update/*` 前缀，统一返回 `{"code":0,"data":{...}}` 或 `{"code":1,"msg":"..."}`，与项目现有 handler 风格一致。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET`  | `/api/update/status`         | 返回当前 LocalVersion、Latest 摘要、LastCheckedAt、Download 状态。前端进入 Settings 时调用 |
| `POST` | `/api/update/check`          | 立即触发一次清单拉取与版本比较，结果回写到内存状态 |
| `POST` | `/api/update/download`       | 开始下载当前 Latest 对应当前架构的 .fpk，**返回后下载在后台进行** |
| `GET`  | `/api/update/progress`       | 拉取 Download 状态（百分比、字节数、阶段）。前端轮询，间隔 500ms |
| `GET`  | `/api/update/progress/stream`| （可选）SSE/WS 长连接推送，前端实现更简单。`/progress` 已够用，先用轮询 |
| `POST` | `/api/update/install`        | 校验文件 SHA256 成功、用户已确认后，调 `appcenter-cli upgrade --file <path>` 或 `cmd/main stop` + 解压安装 |
| `POST` | `/api/update/cancel`         | 取消正在进行的下载（删除临时文件、重置 DownloadState） |
| `POST` | `/api/update/dismiss`        | 用户在 UI 上"忽略此版本"，写入 `dismissed_version` 到 `update-state.json`，下次 check 跳过 |

### 4.1 响应示例

`GET /api/update/status`：
```json
{
  "code": 0,
  "data": {
    "local_version": "1.0.0",
    "latest": {
      "version": "1.2.0",
      "update_available": true,
      "force_update": false,
      "release_notes": "## 1.2.0\n- 新增 xxx\n",
      "published_at": "2026-06-09T10:00:00Z",
      "size_amd64": 25165824,
      "size_arm64": 24867200
    },
    "last_checked_at": "2026-06-09T12:00:00Z",
    "check_error": "",
    "download": { "status": "idle", "percent": 0, "downloaded_bytes": 0, "total_bytes": 0, "error": "", "target_version": "" }
  }
}
```

---

## 5. 模块设计

### 5.1 后端 Go

新增文件：
- `server/internal/version/version.go` — 编译期变量 + `ldflags` 注入
  ```go
  package version
  var (
      AppVersion = "1.0.0-dev"  // 编译时覆盖
      GitCommit  = "unknown"
      BuildTime  = "unknown"
  )
  ```
- `server/internal/update/manager.go` — `Manager` 结构，对应 fn-knock 的 `UpdateManager`
  - `CheckNow(ctx) error`
  - `StartDownload(ctx) error`（异步，写 goroutine）
  - `Install(ctx) error`（同步，返回后框架接管 stop/install/start）
  - `Cancel() error`
  - `State() State`
  - `dismissedVersion() / setDismissedVersion()`
- `server/internal/update/manifest.go` — 远端清单解析与校验
- `server/handler_update.go` — HTTP 路由
- `server/main.go` — `init` 时启动 `Manager`，注册 cron 风格的周期检查（`time.NewTicker(2*time.Hour)`）

构建时版本注入（在 `build-fpk.ps1` 的 `go build` 命令里加 `-ldflags`）：
```
go build -ldflags "-X github.com/.../internal/version.AppVersion=1.2.0 -X github.com/.../internal/version.BuildTime=$(Get-Date -Format o)" -o $OutputPath .
```

### 5.2 前端 Vue

修改文件：
- `frontend/src/api/update.js` — 新建，封装上述 7 个 API
- `frontend/src/views/SettingsView.vue`（或 `components/SettingsView.vue`） — 新增 `<UpdateCard />` 组件
- `frontend/src/components/UpdateCard.vue` — 新建
  - 进入页面调 `/api/update/status` 渲染
  - 显眼的"检查更新"按钮 → `POST /check`
  - 有新版本时显示版本号、release notes、是否强制、架构、预估大小
  - 主按钮"立即更新" → 先 `POST /download` 然后 `setInterval` 轮询 `/progress`
  - 进度条 + 阶段文字（下载中/校验中/已下载/安装中/完成/错误）
  - 完成后提示"系统将重启服务"并显示倒计时

### 5.3 打包脚本

修改 `build-fpk.ps1`：
1. 在 `go build` 之前用语义化版本号（读自 git tag / 单一变量 `$VERSION`）传给 `-ldflags`。
2. `go build` 之后用同一 `$VERSION` 覆盖 `fpk/manifest` 的 `version=` 行（参考 fn-knock 的 `sync_manifest_version`）。
3. 构建产物归档一份 `niudb-1.2.0-amd64.fpk` / `niudb-1.2.0-arm64.fpk`，计算 SHA256，然后由发布脚本（独立脚本，不在本设计范围）上传到 CDN 并写 `latest.json`。

### 5.4 FPK 钩子脚本（可选升级）

现状两个钩子是空的，可以补一些**幂等的迁移逻辑**：

`fpk/cmd/upgrade_init`：
```bash
#!/bin/bash
# 升级前：备份 SQLite 数据库
VAR_DIR="${TRIM_PKGVAR:-/tmp}"
BACKUP_FILE="${VAR_DIR}/pre-upgrade-$(date +%Y%m%d%H%M%S).db"
if [ -f "${VAR_DIR}/data.db" ]; then
    cp "${VAR_DIR}/data.db" "${BACKUP_FILE}" 2>/dev/null
    echo "pre-upgrade backup: ${BACKUP_FILE}" >> "${TRIM_TEMP_LOGFILE:-/dev/null}"
fi
exit 0
```

`fpk/cmd/upgrade_callback`：
```bash
#!/bin/bash
# 升级后：清理临时文件、强制 +x
chmod +x "${TRIM_APPDEST}/server" 2>/dev/null
# 触发应用内"已升级"提示
if [ -n "${TRIM_PKGVAR}" ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') upgrade completed" >> "${TRIM_PKGVAR}/info.log"
fi
exit 0
```

这些钩子在"fnOS 应用中心升级"和"应用内自升级"两条路径上**都会被调用**，是统一升级体验的兜底。

---

## 6. 关键流程

### 6.1 启动时

```
main.go 启动
  ├─ 读 TRIM_APPDEST/version（应用内自升级写入的版本号文件）
  ├─ 启动 update.Manager
  │    ├─ 从 update-state.json 读 dismissed_version
  │    └─ 启动周期 ticker（2h）
  └─ 注册 /api/update/* 路由
```

### 6.2 检查更新（手动 / 周期）

```
前端 / 周期 tick
   → POST /api/update/check
       → Manager.CheckNow()
            → GET ${OTA_LATEST_URL}（加 cache-bust query）
            → 解析 JSON，强类型校验
            → compareVersion(latest.version, local_version)
            → 若 hasUpdate 且 !dismissed → emit 内部事件（写日志、可选推前端）
            → 更新内存 State.LastCheckedAt / Latest
       → 返回 Latest 摘要
```

### 6.3 下载 + 安装

```
用户点"立即更新"
   → POST /api/update/download
       → 校验 hasUpdate && !dismissed
       → 启动 goroutine:
            DownloadState = downloading
            HTTP GET download_url（流式）
            按 chunk 写临时文件 + 更新 Percent
            DownloadState = verifying
            计算 SHA256
            若匹配：rename 到正式路径，DownloadState = downloaded
            若不匹配：删除 + DownloadState = error
   → POST /api/update/install
       → 二次校验文件存在 + SHA256
       → 记 update:pending（version=latest.version）
       → DownloadState = installing
       → 调 exec.Command("appcenter-cli", "upgrade", "--file", fpkPath)
          （若该命令不可用，回退到：cmd/main stop + 解压到 TRIM_APPDEST + 触发 cmd/main start）
       → 返回 200，框架接管 stop→install→start
       → 应用进程被 kill，前端轮询 /progress 会断
   → 服务重启后 /api/update/status 中的 local_version 已更新
       → 前端在恢复连接后显示"已升级到 1.2.0"
```

### 6.4 错误处理矩阵

| 场景 | 行为 |
|---|---|
| latest.json 拉取失败 | DownloadState 不动；前端展示"检查失败：xxx"，可重试 |
| 下载中途断网 | goroutine 内 5s 退避 + 最多 3 次重试；最终失败 → DownloadState = error |
| SHA256 不匹配 | 删除临时文件 → error；前端提示"校验失败，请重试" |
| 架构不匹配 | 启动时检测 `runtime.GOARCH`；前端在卡片显示当前架构 |
| 用户取消 | Cancel() 中断 goroutine + 删除临时文件 + 重置 DownloadState |
| 强制更新 (`force_update=true`) | 前端禁用"忽略"和"稍后"按钮；只有一个"立即更新"主按钮 |
| 应用进程被 kill | 升级路径天然如此；前端 WS 重连后展示成功状态 |
| 同版本反复触发 | Manager 内部幂等：state.downloadedPath == target && status==downloaded → 直接进入 install 阶段 |

---

## 7. 安全考虑

1. **HTTPS 强制**：远端清单和包 URL 必须 HTTPS，避免中间人。
2. **SHA256 强校验**：必填，校验失败禁止安装。
3. **签名升级（未来项）**：可在 `latest.json` 加 `signature` 字段，对应公钥内置到 Go 二进制，用 ed25519 验签。
4. **CSRF**：与项目现有 handler 一致，使用相同来源策略。
5. **资源限制**：`fetch` 加 `signal: AbortSignal.timeout(300_000)`（5 分钟超时），避免挂死。
6. **磁盘安全**：临时目录 `/tmp/niudb-updates/`，包大小超出 `latest.json.size * 1.2` 即停止写入。
7. **权限**：`/api/update/install` 应当与 `cmd/main` 相同，要求 fnOS 给应用 `install_type=root` 或具备 `appcenter-cli` 调用权限。

---

## 8. 发布流程

```
1. 在 frontend/ 与 server/ 完成开发，commit
2. git tag v1.2.0
3. CI / 本地执行：
   - build-fpk.ps1 -Version 1.2.0
     → 前端 build
     → 后端交叉编译 amd64 + arm64（注入 ldflags）
     → 同步 version= 到 manifest
     → 生成两个 .fpk
     → 计算 SHA256
4. 上传两个 .fpk 到 CDN，得到两个 URL
5. 写 latest.json（手工 / 脚本），填好两个 URL + 对应 SHA256
6. 上传 latest.json 到同一 CDN 根目录
7. 在 fnOS 内部署的 niudb 实例中打开 Settings → "检查更新"
   → 应能看到 "1.2.0 可用"
   → 点 "立即更新" → 下载 → 安装 → 自动重启
```

---

## 9. 兼容性与向后兼容

- 旧版本 niudb（< 1.1.0）没有 `/api/update/*`，前端写新卡片时要做特性检测：先 `GET /api/update/status`，404 或超时则隐藏整个卡片。
- `latest.json` 新增字段时保持 Go 端解析为可选（缺省值），不破坏老清单。
- 升级后 `manifest` 中 `version` 升一档，老客户端拉新清单依然能正常比较。
- 如未来需要回滚：在 `latest.json` 中把 `update_available=false` 即可，前端 30 分钟内（cron 周期）会同步到本地。

---

## 10. 后续可扩展项

- [ ] 差分更新：远端同时发 `delta_url` + `base_version`，客户端做 binary diff。
- [ ] 多源镜像：失败时回退到 GitHub Releases 官方 URL。
- [ ] 进度推送从轮询改为 SSE / WebSocket。
- [ ] 在前端 Dashboard 加"有更新"红点徽标。
- [ ] 启动时检测"上次更新成功"标志，弹一个一次性 toast 提示"已升级到 1.2.0，看 changelog"。
- [ ] 自动预下载：检测到新版本时若非强制且当前空闲，后台悄悄下载好，下一次用户点确认即可秒装。
- [ ] 灰度：`latest.json` 加 `rollout_percent`，客户端按 hash(instId) % 100 < rollout 才显示。

---

## 11. 评审检查清单

实施前请确认：

- [ ] 远端 CDN 选型确定（GitHub Release / 自有 OSS / 七牛 / 阿里云）
- [ ] `appcenter-cli` 在目标 fnOS 版本上可用性确认；若不可用，回退到 `cmd/main stop` + 自解压方案
- [ ] `install_type=root` 是否需要（fn-knock 用了 `root` + `install_dep_apps`）
- [ ] 是否同时支持 arm64（fnOS 已有 ARM 设备）
- [ ] 是否需要 release notes 多语言
- [ ] "强制更新"语义：仅阻断 UI，还是阻断服务启动（启动时若 hasForceUpdate && local<min → 直接进入 install 流）
- [ ] 是否需要管理员鉴权（`/api/update/*` 限制为 admin 角色）
