# DataManages 前端 UI 重构计划

> **目标:** 将前端 UI 重构为极简 SaaS 风格，苹果级高级感，卡片式布局，精致动效，保留所有现有功能。

**设计理念:** 少即是多。克制用色，大量留白，精致阴影，流畅动效。

**技术栈:** Vue 3 + Tailwind CSS v4 + reka-ui + ECharts + Lucide Icons

---

## 设计系统

### 色彩体系（极简双色 + 功能色）

| Token | 亮色模式 | 暗色模式 | 用途 |
|-------|---------|---------|------|
| `--background` | `#FAFAFA` | `#0A0A0A` | 页面背景 |
| `--surface` | `#FFFFFF` | `#141414` | 卡片/面板 |
| `--surface-elevated` | `#FFFFFF` | `#1C1C1E` | 悬浮卡片/弹窗 |
| `--border` | `#E5E5E5` | `#2C2C2E` | 边框/分割线 |
| `--border-subtle` | `#F0F0F0` | `#1E1E20` | 轻边框 |
| `--text-primary` | `#171717` | `#F5F5F5` | 主文本 |
| `--text-secondary` | `#737373` | `#A3A3A3` | 次要文本 |
| `--text-tertiary` | `#A3A3A3` | `#525252` | 辅助文本 |
| `--accent` | `#2563EB` | `#3B82F6` | 主强调色（蓝） |
| `--accent-soft` | `#EFF6FF` | `#1E3A5F` | 强调色背景 |
| `--success` | `#16A34A` | `#22C55E` | 成功/在线 |
| `--warning` | `#D97706` | `#F59E0B` | 警告 |
| `--danger` | `#DC2626` | `#EF4444` | 危险/错误 |

### 字体

- **主字体:** Inter（Google Fonts）— 极简、专业、Apple 风格
- **数据字体:** JetBrains Mono — 代码/数据展示
- **标题:** Inter 600/700, 20-32px
- **正文:** Inter 400, 14px
- **辅助:** Inter 400, 12-13px

### 圆角 & 阴影

| 元素 | 圆角 | 阴影 |
|------|------|------|
| 卡片 | 16px | `0 1px 3px rgba(0,0,0,0.04), 0 1px 2px rgba(0,0,0,0.06)` |
| 卡片悬浮 | 16px | `0 10px 40px rgba(0,0,0,0.08)` |
| 弹窗 | 20px | `0 25px 50px rgba(0,0,0,0.15)` |
| 按钮 | 10px | 无（用背景色层次） |
| 输入框 | 10px | 内阴影 `inset 0 1px 2px rgba(0,0,0,0.05)` |
| 标签/徽章 | 8px | 无 |

### 动效规范

| 场景 | 时长 | 缓动 | 属性 |
|------|------|------|------|
| 悬浮反馈 | 200ms | ease-out | transform, box-shadow |
| 页面切换 | 300ms | ease-out | opacity, translateY |
| 弹窗打开 | 250ms | cubic-bezier(0.16,1,0.3,1) | opacity, scale |
| 弹窗关闭 | 200ms | ease-in | opacity, scale |
| 侧边栏 | 300ms | cubic-bezier(0.16,1,0.3,1) | width |
| 数值变化 | 600ms | ease-out | — (count-up) |
| 加载骨架 | 1.5s | ease-in-out | background-position (循环) |

---

## 重构任务分解

### 阶段一：设计基础层（核心样式 & 布局）

#### Task 1: 重构全局样式系统 `index.css`

**文件:** `frontend/src/assets/index.css`

- [ ] **Step 1:** 重写 `:root` 和 `.dark` 的 CSS 变量，替换为新色彩体系
- [ ] **Step 2:** 重写 `@theme inline` 块，映射新的 Tailwind token
- [ ] **Step 3:** 引入 Inter 字体 `@import`
- [ ] **Step 4:** 重写全局基础样式（body, selection, scrollbar）
- [ ] **Step 5:** 删除旧的 utility class（`.btn-primary`, `.content-card` 等），用 Tailwind 直接类替代
- [ ] **Step 6:** 新增动效 keyframes（`fadeUp`, `cardHover`, `shimmer`, `countUp`）
- [ ] **Step 7:** 新增 `prefers-reduced-motion` 媒体查询
- [ ] **Step 8:** 验证亮色/暗色模式切换正常

#### Task 2: 重构侧边栏布局 `App.vue`

**文件:** `frontend/src/components/App.vue`

- [ ] **Step 1:** 重新设计侧边栏 — 毛玻璃效果背景，图标 + 文字，悬浮态微动效
- [ ] **Step 2:** 侧边栏收起时只显示图标，展开时显示图标+文字，过渡动画流畅
- [ ] **Step 3:** 导航项使用 Lucide 图标，active 态用左侧竖条指示器 + 背景色
- [ ] **Step 4:** 顶部添加应用 Logo + 名称区域
- [ ] **Step 5:** 底部放置主题切换和上下文切换
- [ ] **Step 6:** 主内容区域添加 `max-w-[1400px] mx-auto` 居中约束
- [ ] **Step 7:** 重构进度条为顶部细线（2px），更精致
- [ ] **Step 8:** 上下文切换弹窗重新设计为卡片式下拉
- [ ] **Step 9:** 验证所有导航和 KeepAlive 功能正常

---

### 阶段二：UI 基础组件升级

#### Task 3: 升级 Card 组件

**文件:** `frontend/src/components/ui/Card.vue`

- [ ] **Step 1:** 增加悬浮态变体 `interactive`（hover 时阴影提升 + 微上移）
- [ ] **Step 2:** Card 默认圆角 16px，内边距 24px
- [ ] **Step 3:** CardHeader 添加底部细线分隔
- [ ] **Step 4:** 新增 `CardMeta` 子组件（图标 + 标题 + 描述的组合）
- [ ] **Step 5:** 验证所有使用 Card 的页面正常

#### Task 4: 升级 Button 组件

**文件:** `frontend/src/components/ui/Button.vue`

- [ ] **Step 1:** 重新设计变体：`primary`（accent 填充）、`secondary`（surface 填充）、`outline`（边框）、`ghost`（透明）、`danger`（红色填充）
- [ ] **Step 2:** 新增 `size: xs` 变体（28px 高度，12px 字号）
- [ ] **Step 3:** 悬浮态：primary 变暗 10%，secondary 添加边框，ghost 添加背景
- [ ] **Step 4:** 加载态使用 Lucide `Loader2` 旋转图标
- [ ] **Step 5:** 圆角统一 10px，内边距调整
- [ ] **Step 6:** 验证所有按钮功能正常

#### Task 5: 升级 Dialog 组件

**文件:** `frontend/src/components/ui/Dialog.vue`

- [ ] **Step 1:** 遮罩层改为 `backdrop-blur-xl bg-black/30`（毛玻璃遮罩）
- [ ] **Step 2:** 内容面板圆角 20px，阴影加深，入场动画 scale(0.95)→1 + fade
- [ ] **Step 3:** 关闭按钮移至右上角，圆形 ghost 按钮
- [ ] **Step 4:** DialogHeader 添加底部细线分隔
- [ ] **Step 5:** DialogFooter 按钮右对齐，主按钮在右
- [ ] **Step 6:** 验证所有弹窗功能正常

#### Task 6: 升级 Input / Textarea / Select 组件

**文件:** `frontend/src/components/ui/Input.vue`, `Textarea.vue`, `Select.vue`

- [ ] **Step 1:** Input 圆角 10px，focus 态用 accent 色边框 + 微弱光环
- [ ] **Step 2:** 添加悬浮态边框色变化
- [ ] **Step 3:** Select 下拉面板添加阴影和圆角，item 悬浮用 accent-soft 背景
- [ ] **Step 4:** Textarea 同步 Input 样式
- [ ] **Step 5:** 验证表单交互正常

#### Task 7: 升级 Table 组件

**文件:** `frontend/src/components/ui/Table.vue`

- [ ] **Step 1:** 表头背景 `surface` 色，文字 `text-secondary`，字重 500
- [ ] **Step 2:** 行悬浮态 `bg-accent-soft/50`
- [ ] **Step 3:** 行间用 `border-subtle` 分割，去掉竖线
- [ ] **Step 4:** 新增 `TableEmpty` 子组件（空状态图标 + 文字）
- [ ] **Step 5:** 验证所有表格功能正常

#### Task 8: 升级 Badge / Switch / Tabs / Tooltip 组件

**文件:** `frontend/src/components/ui/Badge.vue`, `Switch.vue`, `Tabs.vue`, `Tooltip.vue`

- [ ] **Step 1:** Badge 重新设计：圆角 8px，小号字体，语义色背景（淡色底+深色字）
- [ ] **Step 2:** Switch 重新设计：轨道更宽(40px)，滑块更大，过渡更流畅
- [ ] **Step 3:** Tabs 重新设计：底部滑动指示条动画，活跃态文字加粗
- [ ] **Step 4:** Tooltip 圆角 8px，阴影加深，延迟 300ms 显示
- [ ] **Step 5:** 验证所有组件功能正常

---

### 阶段三：核心视图重构

#### Task 9: 重构 DashboardView（仪表盘）

**文件:** `frontend/src/components/DashboardView.vue`

- [ ] **Step 1:** 顶部实例选择器改为横向滚动卡片列表，选中卡片有 accent 左边框
- [ ] **Step 2:** 4 个统计卡片重新设计：大号数值 + 小号标签 + 右侧图标，悬浮阴影提升
- [ ] **Step 3:** 时间范围选择器改为胶囊按钮组
- [ ] **Step 4:** 图表卡片统一风格：标题 + 操作区 + 图表区域，圆角 16px
- [ ] **Step 5:** 进程列表表格使用新 Table 样式
- [ ] **Step 6:** 自动刷新倒计时改为圆形进度环
- [ ] **Step 7:** 添加页面入场动画（卡片依次 fadeUp）
- [ ] **Step 8:** 验证所有仪表盘功能正常（图表、轮询、实例切换）

#### Task 10: 重构 ManagementView（连接管理）

**文件:** `frontend/src/components/ManagementView.vue`

- [ ] **Step 1:** 统计卡片改为横向排列的 3 个简洁指标卡
- [ ] **Step 2:** 筛选栏重新设计：搜索框 + 类型胶囊筛选 + 操作按钮
- [ ] **Step 3:** 连接列表改为卡片网格布局（每行 2-3 个），替代表格
- [ ] **Step 4:** 每个连接卡片显示：名称、类型徽章、状态点、主机信息、快捷操作
- [ ] **Step 5:** 卡片悬浮态：阴影提升 + 显示操作按钮
- [ ] **Step 6:** 展开详情改为侧滑面板（slide-over），替代行内展开
- [ ] **Step 7:** 验证所有管理功能正常（CRUD、搜索、分页）

#### Task 11: 重构 DataManageView（数据管理）

**文件:** `frontend/src/components/DataManageView.vue`

- [ ] **Step 1:** 实例选择改为顶部下拉选择器 + 状态点
- [ ] **Step 2:** 面包屑导航重新设计：大号可点击文字 + 分隔符
- [ ] **Step 3:** 数据库/表列表改为图标网格卡片
- [ ] **Step 4:** 数据表格使用新 Table 样式，行操作改为悬浮显示
- [ ] **Step 5:** SQL 控制台改为可折叠底部面板，代码编辑器风格
- [ ] **Step 6:** Redis 键列表改为虚拟滚动列表
- [ ] **Step 7:** 验证所有数据管理功能正常（浏览、编辑、SQL执行）

#### Task 12: 重构 BackupView（备份管理）

**文件:** `frontend/src/components/BackupView.vue`

- [ ] **Step 1:** 顶部操作栏重新设计：标题 + 操作按钮组
- [ ] **Step 2:** Tab 切换使用新 Tabs 组件
- [ ] **Step 3:** 备份记录表格使用新 Table 样式
- [ ] **Step 4:** 筛选器改为紧凑的行内组件
- [ ] **Step 5:** 定时备份列表改为卡片列表
- [ ] **Step 6:** 批量操作栏改为底部浮动操作条
- [ ] **Step 7:** 验证所有备份功能正常

#### Task 13: 重构 LogsView（日志中心）

**文件:** `frontend/src/components/LogsView.vue`

- [ ] **Step 1:** 日志查看器重新设计：等宽字体 + 行号 + 语法高亮色
- [ ] **Step 2:** 筛选栏改为紧凑行内布局
- [ ] **Step 3:** 日志级别徽章使用新 Badge 样式
- [ ] **Step 4:** 添加日志行悬浮高亮
- [ ] **Step 5:** 验证所有日志功能正常

#### Task 14: 重构 SettingsView（设置）

**文件:** `frontend/src/components/SettingsView.vue`

- [ ] **Step 1:** 设置页改为左侧 Tab 导航 + 右侧内容区布局
- [ ] **Step 2:** 每个设置项改为卡片行：标签 + 描述 + 控件
- [ ] **Step 3:** 实例管理子页面使用新卡片样式
- [ ] **Step 4:** 配置编辑器使用代码编辑器风格
- [ ] **Step 5:** 验证所有设置功能正常

#### Task 15: 重构 RemoteServerView（远程服务器）

**文件:** `frontend/src/components/RemoteServerView.vue`

- [ ] **Step 1:** 服务器列表改为卡片布局
- [ ] **Step 2:** 每个卡片显示：名称、类型、地址、状态、操作
- [ ] **Step 3:** 空状态设计：大图标 + 引导文字 + 操作按钮
- [ ] **Step 4:** 验证所有远程服务器功能正常

---

### 阶段四：弹窗组件升级

#### Task 16: 升级所有 Dialog 组件

**文件:** `InstanceDialog.vue`, `DatabaseDialog.vue`, `DetectDialog.vue`, `ConnectionDialog.vue`, `BackupDialog.vue`, `RemoteServerDialog.vue`

- [ ] **Step 1:** 所有弹窗使用新 Dialog 基础组件
- [ ] **Step 2:** 表单布局统一：标签在上、输入在下，间距 16px
- [ ] **Step 3:** 按钮组统一：取消(ghost) + 确认(primary)，右对齐
- [ ] **Step 4:** 类型选择器改为分段控件样式
- [ ] **Step 5:** 验证所有弹窗功能正常

---

### 阶段五：动效 & 交互打磨

#### Task 17: 全局动效系统

**文件:** `frontend/src/assets/index.css`, 各视图组件

- [ ] **Step 1:** 页面切换动画：内容 fadeUp 入场
- [ ] **Step 2:** 卡片列表 stagger 动画（依次入场）
- [ ] **Step 3:** 数值变化 count-up 动画
- [ ] **Step 4:** 骨架屏 shimmer 动画替代 loading spinner
- [ ] **Step 5:** 侧边栏导航切换滑动指示器
- [ ] **Step 6:** 表格行悬浮微动效
- [ ] **Step 7:** 按钮点击涟漪效果
- [ ] **Step 8:** 验证 `prefers-reduced-motion` 降级

#### Task 18: 交互细节打磨

**文件:** 各组件

- [ ] **Step 1:** 所有可点击元素添加 `cursor-pointer`
- [ ] **Step 2:** 所有交互元素添加 focus-visible 环
- [ ] **Step 3:** 空状态统一设计（图标 + 文字 + 操作按钮）
- [ ] **Step 4:** 加载状态统一使用骨架屏
- [ ] **Step 5:** 错误状态统一设计（图标 + 错误信息 + 重试按钮）
- [ ] **Step 6:** Toast 通知重新设计：左侧色条 + 图标 + 文字

---

### 阶段六：响应式 & 最终验证

#### Task 19: 响应式适配

**文件:** `App.vue`, 各视图组件

- [ ] **Step 1:** 侧边栏在 768px 以下自动收起为图标模式
- [ ] **Step 2:** 卡片网格在窄屏自动变为单列
- [ ] **Step 3:** 表格在窄屏添加横向滚动
- [ ] **Step 4:** 弹窗在窄屏全屏展示
- [ ] **Step 5:** 验证 375px / 768px / 1024px / 1440px 断点

#### Task 20: 最终验证 & 清理

**文件:** 全项目

- [ ] **Step 1:** 亮色/暗色模式全页面验证
- [ ] **Step 2:** 删除未使用的 CSS class 和变量
- [ ] **Step 3:** 删除未使用的依赖
- [ ] **Step 4:** 构建验证 `npm run build`
- [ ] **Step 5:** 全功能回归测试

---

## 执行原则

1. **每完成一个 Task 提交一次**，commit 信息格式：`refactor(ui): Task N - 描述`
2. **每个 Task 完成后验证**：确保功能不回归
3. **按阶段顺序执行**，阶段内可并行
4. **保留所有现有功能**，只改 UI/UX 不改逻辑
5. **暗色模式同步更新**，每个样式变更都需考虑双模式
