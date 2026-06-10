# niudb

一款基于 **Vue 3 + Go** 的跨平台数据库可视化管理工具，支持 MySQL、MariaDB、PostgreSQL、Redis、SQLite 等多种数据库的集中管理、实时监控、数据操作与定时备份。支持部署为 **fnOS（飞牛 NAS）** 原生应用。

---

## ✨ 功能特点

### 📊 数据库连接与管理
- **多类型支持**：MySQL / MariaDB / PostgreSQL / Redis / SQLite
- **连接管理**：实例的增删改查、连接测试、密码加密存储
- **本地自动检测**：扫描本机端口并识别运行中的数据库实例
- **Docker 容器检测**：自动发现本机 Docker 中的数据库容器并添加
- **远程服务器**：通过 SSH 管理远程主机上的数据库
- **颜色区分**：每种数据库类型拥有独立品牌色（MySQL 蓝、MariaDB 青、PostgreSQL 紫、Redis 橙、SQLite 绿）

### 🎯 数据操作
- **表浏览**：查看数据库/表/视图，支持分页、关键字段显示
- **SQL 控制台**：内置语法高亮编辑器，支持执行任意 SQL 语句
- **Redis 操作**：Key 浏览、类型识别、值查看与命令执行

### 📈 实时监控看板
- **运行指标**：QPS、TPS、连接数、慢查询、缓冲池命中率
- **历史趋势**：ECharts 图表展示指标历史变化
- **实例状态**：在线状态检测、运行时长可视化
- **健康检查**：定时心跳检测，异常状态自动告警

### 💾 备份与恢复
- **即时备份**：一键备份 MySQL / MariaDB / PostgreSQL 数据库
- **定时备份**：Cron 表达式配置，支持自定义执行计划
- **保留策略**：可配置备份保留份数，自动清理历史备份
- **导入恢复**：从备份文件快速恢复

### 🎨 主题与体验
- **深色 / 浅色主题**：一键切换，偏好持久化到 localStorage
- **系统跟随**：自动检测系统配色偏好
- **响应式布局**：桌面端 / 平板自适应
- **紧凑模式**：减小间距，在小屏幕显示更多内容

### 🔌 fnOS 原生集成
- **FPK 打包**：一键打包为飞牛 NAS 原生应用包
- **系统服务**：以非 root 权限运行，自动加入 docker 组以获得容器访问能力

---

## 🧱 技术栈

| 层级 | 技术 |
|------|------|
| **前端** | Vue 3 · Vite 5 · Pinia 3 · Tailwind CSS 4 · ECharts 6 · Vue ECharts · Reka UI · Lucide Vue Next · Vue Sonner · Axios |
| **后端** | Go 1.26 · Gin · GORM · SQLite (纯 Go 驱动) · go-mysql · go-redis · Logrus |
| **图标** | Lucide |
| **部署** | Linux/Windows 二进制 · fnOS FPK 原生应用 |

---

## 🚀 快速开始

### 前置要求

- **Go 1.21+**（推荐 1.26+）
- **Node.js 18+ / npm 9+**
- （可选）`fnpack` — 用于构建 fnOS FPK 包

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/liwenyao016828/data_manages.git
cd data_manages

# 2. 安装前端依赖并启动开发服务器
cd frontend
npm install
npm run dev          # http://localhost:5173

# 3. 另起终端启动后端
cd ../server
go run .             # http://localhost:8080
```

### 生产构建

```bash
# 前端
cd frontend
npm run build        # 输出到 frontend/dist/

# 后端 (Linux amd64)
cd ../server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o niudb-server .

# 后端 (Windows)
CGO_ENABLED=0 go build -o niudb-server.exe .
```

### 打包为 fnOS FPK 应用

```powershell
# Windows
.\build-fpk.ps1

# 输出: fpk/niudb.fpk
```

FPK 构建流程：构建前端 → 交叉编译 Linux amd64 二进制 → 复制到 `fpk/app/` → `fnpack build`。

### 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `TRIM_SERVICE_PORT` | HTTP 服务端口 | `8080` |
| `TRIM_PKGVAR` | 数据目录（SQLite 配置库、日志、备份文件） | `./data/` |

---

## 📁 项目结构

```
data_manages/
├── main.go                    # 应用入口（权限降级、路由注册、服务启动）
├── go.mod / go.sum
├── server/                    # Go 后端源码
│   ├── main.go                # 服务入口
│   ├── model.go               # 数据模型 (Database / Backup / Metrics / ScheduledTask)
│   ├── store.go               # SQLite 持久化封装
│   ├── crypto.go              # 密码加解密
│   ├── connection.go          # 数据库连接管理
│   ├── health_check.go        # 实例健康检测服务
│   ├── db_safe.go             # 安全相关（SQL 白名单等）
│   ├── util.go                # 通用工具
│   ├── handler_database.go    # 实例 CRUD 接口
│   ├── handler_mysql.go       # MySQL / MariaDB 数据操作接口
│   ├── handler_postgresql.go  # PostgreSQL 数据操作接口
│   ├── handler_redis.go       # Redis 数据操作接口
│   ├── handler_sqlite.go      # SQLite 数据操作接口
│   ├── handler_backup.go      # 即时备份 / 恢复接口
│   ├── handler_scheduled.go   # 定时备份接口
│   ├── handler_detect.go      # 本地 & Docker 实例自动检测
│   ├── handler_dashboard.go   # 监控指标接口
│   ├── handler_sync.go        # 端口同步接口
│   └── handler_health.go      # 健康检查接口
├── frontend/                  # Vue 3 前端
│   ├── index.html             # 含预加载主题脚本，避免闪烁
│   ├── vite.config.js
│   └── src/
│       ├── main.js            # 入口
│       ├── App.vue            # 主布局（侧边栏 + 内容区）
│       ├── api/               # Axios API 封装
│       │   ├── database.js    # 实例/数据库/检测接口
│       │   ├── health.js      # 健康检查接口
│       │   └── log.js         # 日志接口
│       ├── assets/
│       │   └── index.css      # Tailwind + 主题 CSS 变量
│       ├── components/
│       │   ├── ui/            # 通用组件 (Button / Card / Dialog / Input / Select / Switch / Table / Tabs / Tooltip ...)
│       │   ├── DashboardView.vue   # 监控看板
│       │   ├── DataManageView.vue  # 数据管理与 SQL 控制台
│       │   ├── ManagementView.vue  # 连接管理
│       │   ├── BackupView.vue      # 备份管理
│       │   ├── LogsView.vue        # 日志中心
│       │   ├── SettingsView.vue    # 设置（主题 / 实例 / 备份策略）
│       │   ├── RemoteServerView.vue# 远程服务器管理
│       │   ├── SqlConsoleTab.vue   # SQL 控制台 Tab
│       │   ├── DetectDialog.vue    # 数据库自动检测弹窗
│       │   ├── InstanceDialog.vue  # 实例编辑弹窗
│       │   ├── BackupDialog.vue    # 备份配置弹窗
│       │   ├── ConnectionDialog.vue# 连接编辑弹窗
│       │   ├── StatusDot.vue       # 状态指示圆点
│       │   └── ThemeToggle.vue     # 主题切换按钮
│       ├── composables/
│       │   ├── useMessage.js   # Toast 消息封装
│       │   └── useConfirm.js   # 确认对话框封装
│       ├── lib/
│       │   ├── instance.js     # 实例 UID 计算
│       │   ├── storageKeys.js  # localStorage key 常量
│       │   └── utils.js        # 类型颜色 / 标签 / 徽章工具
│       └── stores/
│           ├── context.js      # 全局上下文（当前实例、用户名）
│           ├── health.js       # 健康检查状态
│           └── theme.js        # 主题状态（light / dark / auto）
├── fpk/                        # fnOS FPK 打包目录
│   ├── manifest                # 应用元数据
│   ├── LICENSE
│   ├── config/
│   │   ├── privilege           # 权限声明（run-as, username, groupname）
│   │   └── resource            # 资源配额
│   ├── cmd/
│   │   ├── main                # 启动/停止脚本
│   │   ├── install_init        # 安装后执行（加入 docker 组）
│   │   └── uninstall_callback  # 卸载前清理
│   └── app/
│       ├── server              # 编译后的 Linux 二进制
│       └── ui/
│           ├── config          # fnOS UI 启动配置
│           ├── images/         # 应用图标
│           └── dist/           # 前端构建产物
├── build-fpk.ps1               # Windows FPK 一键打包脚本
├── docs/
│   ├── update-design.md        # 在线升级设计文档
│   └── FIXES.md               # 历史问题修复记录
├── LICENSE                     # MIT
└── README.md
```

---

## 🖼️ 截图与快速使用

### 连接管理
1. 进入「连接管理」页面
2. 点击「检测」自动扫描本地 / Docker 容器中的数据库
3. 或点击「添加」手动填写连接信息（主机、端口、用户名、密码）
4. 点击实例卡片底部「数据」进入数据管理，「备份」进入备份管理，「日志」查看运行日志

### 执行 SQL 查询
1. 在侧边栏选择目标数据库实例
2. 进入「数据管理」页面
3. 选择数据库和表，或直接在 SQL 控制台编写语句：

```sql
SELECT id, name, created_at
FROM users
WHERE created_at > '2025-01-01'
ORDER BY id DESC
LIMIT 100;
```

### 创建定时备份
1. 进入「备份管理」页面
2. 切换到「定时备份」标签
3. 点击「创建计划」，选择实例与数据库
4. 配置 Cron 表达式（例如 `0 2 * * *` = 每天凌晨 2 点）
5. 设置保留份数并启用

---

## 📦 FPK 包文件清单

打包后生成的 `niudb.fpk` 约 10MB，包含：
- `app/server` — Linux amd64 静态编译二进制（CGO_ENABLED=0）
- `app/ui/dist/` — Vue 前端构建产物
- `app/ui/images/icon_64.png`、`icon_256.png` — 应用图标
- `cmd/main`、`cmd/install_init`、`cmd/uninstall_callback` — fnOS 生命周期脚本
- `config/privilege`、`config/resource` — 权限与资源声明
- `manifest` — 应用元数据（名称、版本、端口、启动入口）

在 fnOS 上安装后：
- 应用数据位于 `/vol*/@appdata/niudb/`
- 二进制位于 `/vol*/@appcenter/niudb/`
- 默认监听端口 `8080`

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'feat: add some amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 发起 Pull Request

---

## 📄 许可证

本项目基于 **MIT License** 开源，详见 [LICENSE](LICENSE)。

---

## 🙏 致谢

- [Vue.js](https://vuejs.org/) — 渐进式 JS 框架
- [Vite](https://vitejs.dev/) — 下一代前端构建工具
- [Tailwind CSS](https://tailwindcss.com/) — 实用优先 CSS 框架
- [Gin](https://gin-gonic.com/) — Go Web 框架
- [GORM](https://gorm.io/) — Go ORM
- [ECharts](https://echarts.apache.org/) — 强大的图表库
- [go-mysql](https://github.com/go-mysql-org/go-mysql) — Go MySQL 生态工具
- [go-redis](https://github.com/redis/go-redis) — Go Redis 客户端
- [Lucide](https://lucide.dev/) — 精美开源图标库
- [飞牛 fnOS](https://www.fnnas.com/) — 轻量 NAS 操作系统
