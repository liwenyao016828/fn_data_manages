# data_manages

## 项目概述

本项目是一个跨平台的**数据库可视化管理工具（Web 应用）**，旨在为开发者和运维人员提供一站式的数据库管理、监控与备份解决方案。项目采用 **Vue 3 + Go** 全栈开发，具有界面美观、操作便捷、响应迅速等特性，支持本地部署并可作为 **fnOS（飞牛 NAS 系统）** 原生应用运行。

## 功能特点

- **数据库连接管理**：支持 MySQL 和 Redis 数据库的集中管理，包括连接的增删改查、连接测试、自动检测本地数据库实例、端口同步等。
- **数据管理与查询**：提供可视化的 MySQL 数据库/表浏览、表结构查看、SQL 语句执行、数据查询与编辑、用户权限管理等功能，以及 Redis 的 Key 浏览、信息查看和命令执行等功能。
- **系统监控看板**：实时展示数据库运行指标（QPS、TPS、连接数、慢查询、缓冲池命中率等），支持历史数据图表展示，帮助快速定位性能问题。
- **备份与恢复**：支持数据库的即时备份、备份文件导入恢复、定时备份计划（Cron 表达式）、备份保留策略配置，保障数据安全。
- **远程服务器管理**：支持配置远程服务器连接，实现跨主机的数据库管理。
- **健康检查与日志**：内置服务健康检查机制，支持数据库实例在线状态检测；提供系统日志和数据库操作日志的查询与清理功能。
- **深色/浅色主题**：支持暗色/亮色主题一键切换，适配不同使用环境。
- **fnOS 原生集成**：提供 FPK 打包脚本，可一键构建为飞牛 NAS 原生应用。

## 技术栈

- **前端**：Vue 3, Vite, Element Plus, Tailwind CSS, Pinia, ECharts, Reka UI, Lucide Vue Next, Axios, Vue Sonner
- **后端**：Go (net/http 标准库), GORM
- **数据库**：SQLite（本地配置与数据存储）, MySQL（管理对象）, Redis（管理对象）
- **开发工具**：VS Code, Git, npm, Go Modules
- **其他**：Sass, NProgress, go-mysql（MySQL 驱动）, Logrus（日志）

## 安装与使用

### 前提条件

- Go 1.21+
- Node.js 18+ / npm 9+
- （可选）fnpack（用于 FPK 打包）

### 安装步骤

1. 克隆本仓库到本地

   ```bash
   git clone https://github.com/liwenyao016828/data_manages.git
   cd data_manages
   ```

2. 构建前端

   ```bash
   cd frontend
   npm install
   npm run build
   cd ..
   ```

3. 构建后端

   ```bash
   cd server
   go build -o data_manages.exe .
   cd ..
   ```

4. （可选）打包为 fnOS 应用

   ```powershell
   # Windows
   .\build-fpk.ps1

   # Linux
   bash build-fpk.sh
   ```

### 运行项目

```bash
# 进入 server 目录启动服务
cd server
go run .

# 或者运行编译后的可执行文件
./data_manages.exe
```

服务默认监听 `http://localhost:8080`，可通过环境变量 `TRIM_SERVICE_PORT` 自定义端口。

数据文件默认存储在 `./data/` 目录下，可通过环境变量 `TRIM_PKGVAR` 自定义路径。

### 运行测试

```bash
# 前端 TypeScript 类型检查
cd frontend
npx vue-tsc --noEmit
```

## 项目结构

```
data_manages/
├── frontend/                   - 前端源码（Vue 3 + Vite）
│   └── src/
│       ├── api/                - API 请求封装（数据库、健康检查、日志）
│       ├── assets/             - 静态资源（CSS）
│       ├── components/         - Vue 组件
│       │   ├── ui/             - 通用 UI 组件（Button, Card, Dialog 等）
│       │   ├── BackupView.vue  - 备份管理页面
│       │   ├── DashboardView.vue - 控制台/监控看板页面
│       │   ├── DataManageView.vue - 数据管理页面
│       │   ├── ManagementView.vue - 连接管理页面
│       │   ├── LogsView.vue    - 日志中心页面
│       │   └── SettingsView.vue - 设置页面
│       ├── composables/        - 可复用组合式函数（消息提示等）
│       ├── lib/                - 工具库（实例标识、通用工具函数）
│       ├── stores/             - Pinia 状态管理（上下文、健康检查、主题）
│       ├── App.vue             - 根组件
│       └── main.js             - 入口文件
├── server/                     - 后端源码（Go）
│   ├── main.go                 - 服务入口，路由注册与启动
│   ├── model.go                - 数据模型定义（Database, Backup, Metrics 等）
│   ├── store.go                - 本地数据持久化（JSON 文件存储）
│   ├── crypto.go               - 密码加密/解密
│   ├── connection.go           - 数据库连接管理
│   ├── health_check.go         - 健康检查服务
│   ├── db_safe.go              - 数据库安全相关
│   ├── util.go                 - 通用工具函数
│   ├── handler_database.go     - 数据库 CRUD 接口处理
│   ├── handler_mysql.go        - MySQL 管理接口处理
│   ├── handler_redis.go        - Redis 管理接口处理
│   ├── handler_backup.go       - 备份管理接口处理
│   ├── handler_scheduled.go    - 定时备份接口处理
│   ├── handler_remote.go       - 远程服务器接口处理
│   ├── handler_dashboard.go    - 监控看板接口处理
│   ├── handler_detect.go       - 数据库自动检测接口处理
│   ├── handler_sync.go         - 端口同步接口处理
│   └── handler_health.go       - 健康检查接口处理
├── go.mod                      - Go 模块定义
├── go.sum                      - Go 依赖校验
├── build-fpk.ps1               - Windows FPK 打包脚本
├── build-fpk.sh                - Linux FPK 打包脚本
├── LICENSE                     - MIT 许可证
└── README.md
```

## 使用示例

### 添加 MySQL 数据库连接

1. 打开「连接管理」页面
2. 点击「添加数据库」按钮
3. 填写连接信息（名称、类型选择 MySQL、主机、端口、用户名、密码等）
4. 点击「测试连接」验证连通性
5. 点击「保存」

### 执行 SQL 查询

1. 在左侧边栏选择要操作的数据库实例
2. 进入「数据管理」页面
3. 在 SQL 编辑器中输入查询语句，例如：

```sql
SELECT * FROM users WHERE created_at > '2025-01-01' LIMIT 10;
```

4. 点击「执行」按钮查看结果

### 创建定时备份计划

1. 进入「备份管理」页面
2. 切换到「定时备份」标签
3. 点击「创建计划」
4. 选择目标服务器和数据库
5. 配置 Cron 表达式（如 `0 2 * * *` 表示每天凌晨 2 点执行）
6. 设置保留份数并启用

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 许可证

本项目基于 MIT License 开源，详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架
- [Element Plus](https://element-plus.org/) - Vue 3 UI 组件库
- [GORM](https://gorm.io/) - Go ORM 框架
- [ECharts](https://echarts.apache.org/) - 数据可视化图表库
- [Tailwind CSS](https://tailwindcss.com/) - 实用优先的 CSS 框架
- [go-mysql](https://github.com/go-mysql-org/go-mysql) - Go MySQL 驱动库
- [飞牛 fnOS](https://www.fnnas.com/) - NAS 操作系统平台