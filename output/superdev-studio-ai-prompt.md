# superdev-studio - AI 开发提示词

> 由 Super Dev 自动生成
> 生成时间: 2026-03-07 01:34:38

---

## 项目概述

**项目名称**: superdev-studio
**项目描述**: Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.
**目标平台**: WEB
**技术栈**:
- 前端: react
- 后端: go

---

## 你的任务

请根据以下规范和文档，实现 superdev-studio 的所有功能。

**重要**:
1. **严格按照任务列表顺序实现**
2. **每完成一个任务，标记为 [x]**
3. **遵循规范中的所有要求**
4. **参考架构文档中的技术选型**

---

## 核心文档

### 1. PRD (产品需求文档)

# superdev-studio - 产品需求文档 (PRD)

> **生成时间**: 2026-03-07 01:34
> **版本**: v2.0.1
> **状态**: 草稿

---

## 文档信息

| 项目 | 内容 |
|:---|:---|
| **项目名称** | superdev-studio |
| **项目描述** | Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.。请结合以下上下文实现：本地知识参考: PRODUCT_DESIGN - # SuperDev Studio 产品设计说明；外部最佳实践: Refactor an Existing Codebase using Prompt Driven Development - DEV Community - January 13, 2026 -The API exposes endpoints for managing products and categories, including batch operations. Before refactoring, the API is executed locally and exercised through its endpoints to confirm current behavio；外部最佳实践: Supercharge Developer Workflows with GitHub Copilot Workspace Extensions | by David Minkovski | Medium - February 14, 2025 -Use
...

### 2. 架构设计文档

# superdev-studio - 架构设计文档

> **生成时间**: 2026-03-07 01:34
> **版本**: v2.0.1
> **架构师**: Super Dev ARCHITECT 专家

---

## 1. 架构概述

### 1.1 系统目标

- **可扩展性**: 支持水平扩展，应对业务增长
- **可用性**: 99.9% 系统可用性
- **性能**: 低延迟、高吞吐
- **安全性**: 端到端安全防护

### 1.2 架构原则

1. **服务拆分**: 按业务领域拆分微服务
2. **数据库分离**: 读写分离、缓存层
3. **异步处理**: 消息队列解耦
4. **监控运维**: 全链路追踪、实时告警

---

## 2. 技术栈

### 2.1 前端技术栈

| 层级 | 技术选型 | 说明 |
|:---|:---|:---|
| **框架** | React | 组件化开发 |
| **状态管理** | Redux Toolkit / Zustand | 全局状态管理 |
| **UI 框架** | Ant Design / Chakra UI | 组件库 |
| **构建工具** | Vite | 打包构建 |
| **HTTP 客户端** | Axios | API 请求 |
| **路由** | React Router | 页面路由 |

### 2.2 后端技术栈

| 层级 | 技术选型 | 说明 |
|:---|:---|:---|
| **运行时** | Go | 服务器运行时 |
| **框架** | Gin / Echo | Web 框架 |
| **API 规范** | RESTful | 接口设计 |
| **认证** | JWT | Token 认证 |
| **ORM** | GORM | 数据库 ORM |
| **验证** | Joi/Zod | 数据验证 |



### 2.3 数据存储

| 存储 | 技术选型 | 用途 |
|:---|:---|:---|
| **主数据库** | PostgreSQL 14+ | 持久化存储 |
| **缓存** | Redis | 缓存层 |
| **文件存储** | AWS S3 / 阿里云 OSS | 文件/图片 |
| 
...

### 3. UI/UX 设计文档

# superdev-studio - UI/UX 设计文档

> **生成时间**: 2026-03-07 01:34
> **版本**: v2.0.1
> **设计师**: Super Dev UI/UX 专家

---

## 0. 设计分析

### 0.1 项目特征

基于需求描述，AI 分析出以下项目特征：

| 特征 | 分析结果 | 说明 |
|:---|:---|:---|
| **产品类型** | General | 通用产品 |
| **行业领域** | General | 通用行业 |
| **风格倾向** | Professional | 专业风格，商务、正式 |
| **技术栈** | REACT | 前端框架 |

### 0.2 设计推荐摘要

AI 基于项目特征，从设计数据库中为您推荐：



---

## 1. 设计概述

### 1.1 设计理念

- **简洁**: 去除不必要的元素
- **一致**: 统一的视觉语言
- **高效**: 快速完成任务
- **愉悦**: 细节打磨体验

### 1.2 设计原则

1. **用户中心**: 以用户需求为出发点
2. **数据驱动**: 基于数据迭代设计
3. **移动优先**: 响应式设计
4. **无障碍**: 符合 WCAG 2.1 AA

---

## 2. 设计系统

### 2.1 色彩规范


#### 主色调

| 颜色 | 用途 | Hex | RGB |
|:---|:---|:---|:---|
| **Primary** | 主要操作、强调 | #2563EB | rgb(37, 99, 235) |
| **Secondary** | 次要操作 | #64748B | rgb(100, 116, 139) |
| **Success** | 成功状态 | #10B981 | rgb(16, 185, 129) |
| **Warning** | 警告状态 | #F59E0B | rgb(245, 158, 11) |
| **Error** | 错误状态 | #EF4444 | rgb(239, 68, 68) |

---


### 2.2 字体规范

#### 字体家族

```css
font-family: -apple-system, 
...

### 4. 执行路线图

# superdev-studio - 执行路线图

> **生成时间**: 2026-03-07 01:34
> **场景**: 1-N+1
> **策略**: 先前端可视化，再系统能力闭环

---

## 1. 需求范围

| 模块 | 需求 | 说明 |
|:---|:---|:---|
| core | business-core-flow | 系统应完整支持以下业务目标：Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.。请结合以下上下文实现：本地知识参考: PRODUCT_DESIGN - # SuperDev Studio 产品设计说明；外部最佳实践: Refactor an Existing Codebase using Prompt Driven Development - DEV Community - January 13, 2026 -The API exposes endpoints for managing products and categories, including batch operations. Before refactoring, the API is executed locally and exercised through its endpoints to confirm current behavio；外部最佳实践: Supercharge Developer Workflows with GitHub Copilot Workspace Extensions | by David Minkovski | Medium - Febru
...

### 5. 前端蓝图

# superdev-studio - 前端蓝图

> **生成时间**: 2026-03-07 01:34
> **前端框架**: react
> **设计重点**: 先可视化业务流程，再补齐系统深度能力

---

## 1. 体验目标

- 一次进入即可理解产品价值和关键流程。
- 文档和执行状态可追踪，避免信息散落。
- 关键任务路径操作链路最短、反馈明确。

## 2. 模块拆分

### 1. 需求总览面板

**目标**: 集中展示需求摘要、优先级和执行状态。

**核心元素**:
- 需求卡片
- 优先级标签
- 状态标识

### 2. 文档工作台

**目标**: 统一管理 PRD、架构和 UIUX 文档入口。

**核心元素**:
- 文档卡片
- 版本信息
- 快速跳转

### 3. 执行路线图

**目标**: 可视化 0-1 / 1-N+1 阶段任务。

**核心元素**:
- 阶段时间线
- 里程碑
- 风险提示

### 4. core 模块

**目标**: 系统应完整支持以下业务目标：Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.。请结合以下上下文实现：本地知识参考: PRODUCT_DESIGN - # SuperDev Studio 产品设计说明；外部最佳实践: Refactor an Existing Codebase using Prompt Driven Development - DEV Community - January 13, 2026 -The API exposes endpoints for managing product
...

---

## 任务列表


### 变更: superdev-studio

**描述**: Superdev Studio

**状态**: proposed

#### 任务列表

[ ] **1.1: 梳理增量变更影响范围**
  - 明确兼容性、风险点和回滚策略。
  - 规范引用: core::*

[ ] **1.2: 确定灰度与开关策略**
  - 对增量功能设计灰度发布与快速回滚方案。
  - 规范引用: core::*

[ ] **2.1: 实现 core 前端模块**
  - 先交付 core 的页面框架、核心组件与交互流程。
  - 规范引用: core::*

[ ] **2.2: 实现 profile 前端模块**
  - 先交付 profile 的页面框架、核心组件与交互流程。
  - 规范引用: profile::*

[ ] **3.1: 实现 core 后端能力**
  - 完成 core 的 API、数据模型和权限控制。
  - 规范引用: core::*

[ ] **3.2: 实现 profile 后端能力**
  - 完成 profile 的 API、数据模型和权限控制。
  - 规范引用: profile::*

[ ] **4.1: 完成端到端联调**
  - 对关键链路进行联调，修复接口与交互偏差。
  - 规范引用: core::*

[ ] **4.2: 执行质量门禁前检查**
  - 完成安全、性能、可用性预检查并修复阻塞项。
  - 规范引用: core::*

[ ] **5.1: 测试 core 功能**
  - 编写并执行 core 的单元、集成与回归测试。
  - 规范引用: core::*

[ ] **5.2: 测试 profile 功能**
  - 编写并执行 profile 的单元、集成与回归测试。
  - 规范引用: profile::*

#### 规范要求

**core** (added)
- business-core-flow: 系统应完整支持以下业务目标：Refactor SuperDev Studio into a change-driven delivery workspace. Unify the product IA around workspace, change center, delivery runs, context hub, and project settings. Persist project default execution profile. Add change batch and run traceability metadata. Upgrade the React plus Go app pages and APIs accordingly.。请结合以下上下文实现：本地知识参考: PRODUCT_DESIGN - # SuperDev Studio 产品设计说明；外部最佳实践: Refactor an Existing Codebase using Prompt Driven Development - DEV Community - January 13, 2026 -The API exposes endpoints for managing products and categories, including batch operations. Before refactoring, the API is executed locally and exercised through its endpoints to confirm current behavio；外部最佳实践: Supercharge Developer Workflows with GitHub Copilot Workspace Extensions | by David Minkovski | Medium - February 14, 2025 -Use the same UI/UX, but with our own AI logic.；外部最佳实践: 4 steps to connect change management and DevOps | CIO - December 1, 2023 -Businesses must emphasize the importance ofclear processes for requesting, reviewing, approving, and implementing changes, rigorous testing and validation, and continuous improvementto ensure successful
  - 按业务路径完成主要操作: 系统成功返回结果并展示下一步引导

**profile** (added)
- profile-management: 用户应可查看和更新个人资料与偏好设置。
  - 在个人中心提交更新: 资料变更被持久化并反馈成功状态


---

## 开发规范

### 代码规范

1. **遵循项目代码风格**
   - 使用 Prettier 格式化
   - 使用 ESLint 检查
   - 遵循现有命名规范

2. **提交规范**
   - Conventional Commits
   - 一个功能一个 commit
   - Commit message 清晰描述变更

3. **测试规范**
   - 单元测试覆盖率 > 80%
   - 每个功能点都有测试
   - 使用 pytest / jest

### 图标使用规范

**严格禁止**:
- ❌ **禁止使用 emoji 表情作为图标**
  - 不允许使用 emoji 来代替图标（如 💾 保存、🔍 搜索、⚙️ 设置）
  - emoji 在不同平台显示不一致
  - 可访问性差（屏幕阅读器支持不佳）
  - 不够专业

**图标使用标准**（按优先级）:
1. ✅ **首选**: UI 框架自带图标库
   - Vue: Element Plus、Naive UI、Vuetify 自带图标
   - React: Ant Design、Material-UI、Chakra UI 图标
   - 其他: 使用项目选择的 UI 库官方图标

2. ✅ **专业图标库**:
   - [Lucide Icons](https://lucide.dev/) - 推荐，轻量且现代
   - [Heroicons](https://heroicons.com/) - Tailwind CSS 官方
   - [Tabler Icons](https://tabler-icons.io/) - 开源免费
   - [Phosphor Icons](https://phosphoricons.com/) - 精美免费

3. ✅ **自定义 SVG**:
   - 如果需要自定义图标，使用 SVG 格式
   - 确保遵循无障碍标准（添加 aria-label）

**代码示例**:
```typescript
// ✅ 正确：使用图标库
import { Save, Search, Settings } from 'lucide-react';
<button><Save size={20} />保存</button>

// ❌ 错误：使用 emoji
<button>💾 保存</button>
```

### 安全规范

1. **输入验证**: 所有用户输入必须验证
2. **SQL 注入**: 使用参数化查询
3. **XSS**: 输出转义
4. **认证**: JWT Token 认证

---

## 文件结构

请按照以下结构组织代码：

```
project-root/
├── frontend/          # 前端代码
│   ├── src/
│   │   ├── components/  # 组件
│   │   ├── pages/       # 页面
│   │   ├── services/    # API 服务
│   │   └── utils/       # 工具函数
│   ├── package.json
│   └── vite.config.js
│
├── backend/           # 后端代码
│   ├── src/
│   │   ├── controllers/ # 控制器
│   │   ├── models/      # 数据模型
│   │   ├── services/    # 业务逻辑
│   │   ├── routes/      # 路由
│   │   └── utils/       # 工具函数
│   ├── package.json
│   └── tsconfig.json
│
└── shared/            # 共享代码
    ├── types/         # 类型定义
    └── constants/     # 常量
```

---

## 开始实现

请从任务 1.1 开始，按顺序实现所有任务。

**每完成一个任务**:
1. 更新 `.super-dev/changes/superdev-studio/tasks.md`
2. 将任务标记为 [x] 完成状态
3. 提交代码 (可选)
4. 继续下一个任务

---

## 遇到问题？

如果遇到不清楚的地方：
1. 优先查看架构文档
2. 参考 PRD 中的需求说明
3. 查看 UI/UX 文档中的设计规范

---

## 完成标准

所有任务完成后：
- [ ] 所有功能正常运行
- [ ] 所有测试通过
- [ ] 代码符合规范
- [ ] 文档已更新

**祝开发顺利！**
