# SuperDev Studio

SuperDev Studio 是基于 `super-dev` 流水线思想构建的可视化开发辅助工具，提供：

- 项目与任务管理
- AI 流水线可视化运行控制
- 记忆模块（Memory）
- 知识库（Knowledge Base）
- 上下文优化器（Context Pack）

技术栈：

- Frontend: React + TypeScript + Vite + Ant Design + TanStack Query
- Backend: Go + Chi
- DB: SQLite

## 目录结构

```text
super-dev-studio/
  backend/                # Go API
  frontend/               # React UI
  docs/
    PRODUCT_DESIGN.md     # 产品设计与调研
    TEST_REPORT.md        # 测试报告
```

## 快速启动

## 1) 启动后端

```bash
cd backend
go mod tidy
go run ./cmd/server
```

默认地址：`http://localhost:8080`

可选环境变量（见 `backend/.env.example`）：

- `SUPERDEV_STUDIO_ADDR`：服务监听地址，默认 `:8080`
- `SUPERDEV_STUDIO_DB`：SQLite 路径，默认 `./data/superdev_studio.db`
- `SUPER_DEV_CMD`：真实 super-dev 调用命令，默认 `python -m super_dev.cli`
- `SUPER_DEV_WORKDIR`：super-dev 工作目录（当前版本保留）
- `VOLCENGINE_ARK_API_KEY`：火山引擎方舟 API Key（启用 LLM 迭代建议）
- `VOLCENGINE_ARK_MODEL`：火山引擎模型/Endpoint ID
- `VOLCENGINE_ARK_BASE_URL`：方舟兼容接口地址，默认 `https://ark.cn-beijing.volces.com/api/v3`

后端启动时会自动加载 `.env`：

- 优先读取当前工作目录下的 `.env`
- 同时兼容读取 `backend/.env`（从仓库根目录启动时）
- 已存在且非空的系统环境变量优先，不会被 `.env` 覆盖

## 2) 启动前端

```bash
cd frontend
npm install
npm run dev
```

默认地址：`http://localhost:5273`

如后端地址不同，可配置：

```bash
VITE_API_BASE_URL=http://localhost:8080
```

## 3) 构建与测试

后端测试：

```bash
cd backend
go test ./...
```

前端测试：

```bash
cd frontend
npm run test
```

前端构建：

```bash
cd frontend
npm run build
```

## 主要页面

- `Dashboard`：项目运营概览、最近流水线运行
- `项目管理`：项目列表、任务看板
- `流水线`：启动运行（模拟/真实 super-dev）、查看进度和事件
- `记忆模块`：写入/查看记忆条目
- `知识库`：导入文档、切片、检索
- `上下文优化`：根据 query 在 token 预算内打包最佳上下文

流水线支持上下文动态注入：

- `off`：不注入上下文，直接执行原始需求
- `auto`：基于需求自动构建 Context Pack 并注入执行 prompt
- `manual`：基于手动 query 构建 Context Pack 并注入执行 prompt
- 可选开启“按阶段动态召回”，针对 discovery/implementation/redteam/qa/delivery 追加阶段上下文
- 可选开启“运行结束回写记忆”，将本次运行总结自动写入记忆模块
- 失败运行支持一键重试（复用原运行配置并建立 `retry_of` 关联）
- 可选开启“一键全流程交付（full_cycle）”
  - 设计 -> 开发-单测-修复迭代 -> 质量测试 -> 验收总结 -> 上线准备
  - 支持迭代次数 `iteration_limit`
  - 开启后自动走真实 super-dev 执行（非模拟）
- 可选开启“逐步开发模式（step_by_step）”
  - 按 super-dev 原生命令逐步执行：`create -> spec validate -> task status -> task run -> quality -> preview -> deploy`
  - 自动从 `create` 日志解析 `change_id`，失败可在事件流中定位
  - 流水线会读取初始化文档并自动写入「项目任务」模块（任务看板可见）
  - 后续以项目任务为基准推进：按任务进入 `in_progress -> done` 状态并驱动 super-dev 执行
  - 若已配置火山引擎 API（`VOLCENGINE_ARK_*`），LLM 智能体会基于文档/任务/质量结果生成每轮推进建议并写入事件流
  - 开启后自动走真实 super-dev 执行（非模拟），并与 `full_cycle` 互斥

一键全流程执行序列（后端编排）：

1. `lifecycle-design`
- `super-dev pipeline ... --skip-quality-gate --skip-redteam --skip-scaffold`
2. `lifecycle-iteration-*`
- LLM 生成修复动作清单后执行
- `super-dev pipeline ... --skip-quality-gate --skip-redteam`
3. `lifecycle-quality-*`
- `super-dev quality --type all`
  - 自动同步 `super-dev` 配置中的项目名，避免质量检查按错误前缀误判文档缺失
  - 对非 Python 后端，若仅剩 `Python 语法检查` 单项失败且分数达标，将记录 soft-pass 并继续流程
4. `lifecycle-acceptance`
- 生成验收总结
5. `lifecycle-release` / `lifecycle-preview`
- `super-dev deploy --docker --cicd all`
- `super-dev preview --output output/preview.html`

逐步开发模式执行序列（后端编排）：

1. `step-create`
- `super-dev create "<prompt>" --platform ... --frontend ... --backend ... --domain ...`
2. `step-spec-validate` / `step-task-status-init`
- `super-dev spec validate <change_id>`
- `super-dev task status <change_id>`
3. `step-task-run-*` / `step-task-status-*` / `step-quality-*`（迭代）
- `super-dev task run <change_id> --max-retries N --platform ... --frontend ... --backend ...`
- `super-dev task status <change_id>`
- `super-dev quality --type all`
- 每个项目任务都会进入任务看板并作为执行基线
- 质量未通过时，调用火山引擎 LLM 生成“下一轮修复动作”并继续迭代，直到当前任务完成
4. `step-preview`
- `super-dev preview --output output/preview.html`
5. `step-release`
- `super-dev deploy --docker --cicd all`

## 流水线输出位置与预览

- 真实 `super-dev` 模式下，默认输出目录为：`<project_dir>/output`
- 若启动时未填写 `project_dir`，则输出到后端进程当前工作目录下的 `output`（通常是 `backend/output`）
- 常见产物：
  - 文档：`<change-id>-prd.md`、`<change-id>-architecture.md`、`<change-id>-quality-gate.md` 等
  - 前端可预览页面：`output/frontend/index.html`（及 `styles.css`、`app.js`）
  - 一键全流程预览页：`output/preview.html`
- 模拟模式仅演示阶段事件，不保证生成真实产物文件

流水线详情页新增能力：

- 完成清单：展示运行状态、文档产物、前端产物是否完成
- 产物列表：展示检测到的输出文件路径与基础信息
- 预览功能：若检测到 `output/frontend/index.html` 或 `output/preview.html`，可直接点击“预览页面”内嵌查看

## 三大模块在流水线中的动态应用

推荐使用顺序：

1. 在「记忆模块」写入团队约束、历史决策、风险点。
2. 在「知识库」导入 PRD、接口规范、部署手册并执行检索验证。
3. 在「流水线」选择 `context_mode=auto`，开启「按阶段动态召回」与「运行结束回写记忆」。
4. 运行后在「记忆模块」查看 `role=run-summary` 自动回写条目，用于下次检索。

运行时行为：

- 启动时先构建一次通用 Context Pack 注入执行 Prompt。
- 若开启 `context_dynamic`，系统会按关键阶段构建阶段上下文摘要并追加到 Prompt。
- 若开启 `memory_writeback`，运行结束会自动沉淀 `run-summary` 记忆（包含 run_id、状态、阶段、原始需求等）。

核心接口：

- 记忆：`GET/POST /api/projects/{id}/memories`
- 知识库：`GET/POST /api/projects/{id}/knowledge/documents`、`GET /api/projects/{id}/knowledge/search`
- 上下文优化：`POST /api/projects/{id}/context-pack`
- 流水线启动：`POST /api/pipeline/runs`（支持 `context_mode/context_query/context_dynamic/memory_writeback/full_cycle/step_by_step/iteration_limit`）
- 流水线重试：`POST /api/pipeline/runs/{runID}/retry`（仅允许重试 `failed` 运行）
- 运行完成清单：`GET /api/pipeline/runs/{runID}/completion`
- 运行预览页面：`GET /api/pipeline/runs/{runID}/preview/*`

## 功能覆盖检查（当前版本）

已具备：

- 记忆模块：写入、列表展示、运行结束自动回写
- 知识库：文档导入、切片存储、关键词检索
- 上下文优化：按 query + token 预算 + max_items 构建上下文包并摘要
- 流水线动态应用：`off/auto/manual` 注入、关键阶段动态召回、事件日志可观测

当前未覆盖（规划中的可扩展项）：

- 记忆与知识的编辑/删除权限流
- 向量数据库检索与重排
- 普通模式（非 `full_cycle`）仍为单次 `super-dev pipeline`；如需多轮自动迭代请开启 `full_cycle`

## super-dev 集成说明

流水线页面支持两种模式：

1. 模拟模式（默认）
- 快速演示 12 阶段运行链路

2. 真实模式
- 调用 `SUPER_DEV_CMD` 执行 super-dev CLI
- 运行日志写入事件流

建议先模拟模式验证流程，再切换真实模式。

## 一键全流程 API 示例

```bash
curl -X POST http://localhost:8080/api/pipeline/runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "YOUR_PROJECT_ID",
    "prompt": "实现一个React+Go+SQLite的任务协同系统",
    "simulate": false,
    "full_cycle": true,
    "iteration_limit": 3,
    "project_dir": "D:/Work/target-project",
    "platform": "web",
    "frontend": "react",
    "backend": "go",
    "context_mode": "auto",
    "context_dynamic": true,
    "memory_writeback": true
  }'
```

## 逐步开发模式 API 示例

```bash
curl -X POST http://localhost:8080/api/pipeline/runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "YOUR_PROJECT_ID",
    "prompt": "实现一个React+Go+SQLite的任务协同系统",
    "simulate": false,
    "step_by_step": true,
    "project_dir": "D:/Work/target-project",
    "platform": "web",
    "frontend": "react",
    "backend": "go",
    "context_mode": "auto",
    "context_dynamic": true,
    "memory_writeback": true
  }'
```

## 文档

- 产品设计：`docs/PRODUCT_DESIGN.md`
- 测试报告：`docs/TEST_REPORT.md`
