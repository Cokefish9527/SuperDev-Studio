# SuperDev Studio 产品设计说明

## 1. 背景与目标
`super-dev` 已经具备完整的 AI 研发流水线编排能力，但主要入口是 CLI。目标是构建一个图形化开发辅助工具 `SuperDev Studio`，将其编排思想扩展为可视化协作产品，并加入：

- 记忆模块（对话/决策沉淀）
- 上下文优化（Token 预算内上下文打包）
- 知识库（文档入库与检索）
- 项目管理（项目/任务/流水线运行）

技术选型：

- 前端：React + TypeScript + Vite + Ant Design + TanStack Query
- 后端：Golang（Chi）
- 数据库：SQLite（含 FTS 检索兜底）

## 2. super-dev 代码分析与复用映射

### 2.1 核心发现
通过阅读 `super-dev` 仓库（`README`、CLI、编排引擎、Web API）可抽象出三类核心模式：

1. **阶段化流水线模型**
- `super_dev/orchestrator/engine.py` 中明确了分阶段执行、阶段结果、质量门禁、运行报告。

2. **运行状态持久化模型**
- `super_dev/web/api.py` 中维护 run 状态、阶段结果、事件和工件下载。

3. **CLI 与 API 双入口模型**
- `super_dev/cli.py` 负责命令路由，`web/api.py` 提供可视化能力底座。

### 2.2 本产品映射
`SuperDev Studio` 直接采用上述思想：

- 后端 `pipeline.Manager` 采用 12 阶段事件流（并支持真实 `super-dev` 命令执行）
- 数据层保存 `pipeline_runs` + `run_events`
- 前端提供 Pipeline 控制台（启动、进度、事件时间线）

## 3. 外部调研结论（AI 开发辅助工具）
调研重点为“当前主流 AI 工具在开发流程中的能力方向”：

1. **可异步执行的 Agent 工作流**
- Cursor 官方文档强调 Background Agents，可将任务异步执行并回传状态。

2. **对代码库上下文理解与自动化任务分派**
- GitHub Copilot 官方文档强调在仓库上下文中执行探索、建议与自动化开发任务。

3. **知识检索和文件检索能力（RAG）**
- OpenAI 官方 File Search 指南强调将文件转为可检索上下文，是稳定提升输出质量的关键。

4. **成本与延迟优化（Prompt Caching）**
- OpenAI Prompt Caching 提供对重复上下文的缓存机制，适合频繁迭代场景。

5. **项目管理与跟踪能力**
- GitHub Projects 支持规划、看板与任务跟踪，适合和 AI 产物联动。

## 4. 信息架构与功能设计

### 4.1 页面结构
- 概览 Dashboard
- 项目管理 Projects
- 流水线 Pipeline
- 记忆模块 Memory
- 知识库 Knowledge
- 上下文优化 Context Optimizer

### 4.2 核心能力
1. 项目管理
- 创建/选择项目
- 任务管理（todo/in_progress/done）

2. 流水线编排
- 启动运行（模拟/真实 super-dev）
- 查看运行状态、阶段进度、事件日志
- 支持上下文动态注入（off/auto/manual）
  - auto：按需求自动召回记忆+知识并注入 prompt
  - manual：按自定义 query 召回并注入 prompt
  - dynamic by phase：按关键阶段追加阶段上下文
  - memory writeback：运行结束自动回写 run-summary 记忆

3. 记忆模块
- 写入结构化记忆（role/content/tags/importance）
- 按项目回看记忆沉淀

4. 知识库
- 文档入库、自动切片
- FTS 检索 + LIKE 兜底检索

5. 上下文优化
- 按 query 汇总记忆与知识片段
- 在 token 预算下进行选择与摘要输出

## 5. 数据模型
SQLite 核心表：

- `projects`
- `tasks`
- `pipeline_runs`
- `run_events`
- `memories`
- `knowledge_documents`
- `knowledge_chunks`
- `knowledge_chunks_fts`（可用时启用）

## 6. 后端接口设计（摘要）

- `GET /api/health`
- `GET/POST/PUT/DELETE /api/projects`
- `GET/POST /api/projects/{id}/tasks`
- `PATCH /api/tasks/{id}`
- `POST /api/pipeline/runs`
- `GET /api/pipeline/runs/{id}`
- `GET /api/pipeline/runs/{id}/events`
- `GET/POST /api/projects/{id}/memories`
- `GET/POST /api/projects/{id}/knowledge/documents`
- `GET /api/projects/{id}/knowledge/search`
- `POST /api/projects/{id}/context-pack`
- `GET /api/dashboard`

## 7. UI 设计原则

- 侧边导航 + 顶部项目上下文切换
- 以“阶段进度 + 事件时间线”展示流水线
- 记忆与知识在上下文优化器中可视化组合
- 使用统一设计变量（色彩、字体、圆角）保证一致性

## 8. 可扩展路线

1. 接入向量数据库（如 pgvector/milvus）替换纯文本检索
2. 引入多模型路由与 Agent 执行队列
3. 与 GitHub Projects/Jira 双向同步任务
4. 增加 Prompt 缓存与上下文增量更新

## 9. 参考资料
- super-dev 仓库: https://github.com/Cokefish9527/super-dev
- Cursor Background Agents: https://docs.cursor.com/en/background-agents/overview
- GitHub Copilot 开发流程文档: https://docs.github.com/copilot/tutorials/use-copilot-to-explore-projects
- OpenAI File Search: https://platform.openai.com/docs/guides/tools-file-search
- OpenAI Prompt Caching: https://platform.openai.com/docs/guides/prompt-caching
- GitHub Projects: https://docs.github.com/issues/planning-and-tracking-with-projects
- TanStack Query (React): https://tanstack.com/query/latest/docs/framework/react/overview

## 10. 当前落地状态（2026-03-05）

三大模块与流水线联动：

- 记忆模块：已落地
  - 页面：新增/列表
  - 接口：`GET/POST /api/projects/{id}/memories`
  - 流水线联动：可召回；运行结束可自动回写 `run-summary`
- 知识库：已落地
  - 页面：文档入库、检索、列表
  - 接口：`GET/POST /api/projects/{id}/knowledge/documents`、`GET /api/projects/{id}/knowledge/search`
  - 流水线联动：可召回并参与 Context Pack
- 上下文优化：已落地
  - 页面：按 query 构建并展示 context pack
  - 接口：`POST /api/projects/{id}/context-pack`
  - 流水线联动：`off/auto/manual` + 可选阶段动态召回

与规划存在的差距（非阻塞）：

- 记忆/知识当前未提供编辑与删除 UI
- 检索仍以 SQLite FTS + LIKE 为主，未接入向量检索与重排
- 普通（非 `full_cycle`）模式仍为单次 `super-dev pipeline` 执行；一键全流程模式已支持多阶段命令编排与迭代

## 11. 一键全流程交付（火山引擎 LLM + super-dev）

目标：在一次运行中串联 `设计 -> [开发-单测-修复]迭代 -> 测试 -> 验收 -> 准备上线`。

已落地编排（后端 `pipeline.Manager`）：

1. 设计阶段（`lifecycle-design`）
- 执行 `super-dev pipeline ... --skip-quality-gate --skip-redteam --skip-scaffold`
- 输出需求/架构/规格文档，建立迭代输入基线

2. 开发-单测-修复迭代（`lifecycle-iteration-*`）
- 每轮先通过 LLM 生成修复动作清单（火山引擎未配置时使用内置兜底建议）
- 执行 `super-dev pipeline ... --skip-quality-gate --skip-redteam`（保留实现骨架与任务执行）

3. 测试阶段（`lifecycle-quality-*`）
- 执行 `super-dev quality --type all`
- 未通过时进入下一轮迭代，直到达到 `iteration_limit`

4. 验收阶段（`lifecycle-acceptance`）
- LLM 生成 3-5 条上线前验收总结（功能、测试、发布/回滚）

5. 准备上线阶段（`lifecycle-release` / `lifecycle-preview`）
- 执行 `super-dev deploy --docker --cicd all`
- 执行 `super-dev preview --output output/preview.html`

火山引擎接入：

- 环境变量：`VOLCENGINE_ARK_API_KEY`、`VOLCENGINE_ARK_MODEL`、`VOLCENGINE_ARK_BASE_URL`
- 协议：Ark OpenAI 兼容 `/chat/completions`
- 用途：迭代修复建议 + 验收总结生成
