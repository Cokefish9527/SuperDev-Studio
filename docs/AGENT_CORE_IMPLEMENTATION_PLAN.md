# SuperDev Studio Agent Core 落地方案

## 1. 文档目标

本文给出一份可落地的 Agent 升级方案，用于将 `SuperDev Studio` 从“LLM 局部增强的交付工作台”升级为“具备自主规划、工具调用、上下文检索、结果评估与迭代修复能力的系统性 Agent 平台”。

本文聚焦三件事：

1. 解释为什么当前版本“接了 LLM 但不够智能”。
2. 对比三条路径：
   - 继续基于现有设计做增强
   - 引入 `LangChain / LangGraph`
   - 仿造 `OpenCode / Oh-My-OpenCode` 风格插件体系
3. 给出一个能够在现有代码基础上分阶段实施的技术方案。

## 2. 执行摘要

### 2.1 结论

建议采用以下总体路线：

> 保留现有 `Project -> ChangeBatch -> PipelineRun -> Context Hub` 产品骨架，新增独立的 `Agent Core` 编排层；
> 核心自治与状态编排使用 `LangGraph`；
> 插件/技能/模式层借鉴 `OpenCode / Oh-My-OpenCode` 的可扩展设计；
> 现有 Go 后端继续承担控制面、持久化、事件流与 UI API。

这意味着：

- **不是** 推倒重做现有 Studio。
- **不是** 仅仅替换一个模型供应商。
- **不是** 先做插件市场再补智能核心。
- **而是** 先补 Agent 大脑，再补插件层和体验层。

### 2.2 推荐顺序

推荐实施顺序如下：

1. 建立 `Agent Core` 最小闭环（优先覆盖 `step_by_step` 模式）
2. 升级检索与上下文链路（Embedding / Rerank / Evidence）
3. 接入评估与自修复（Evaluator / Critic / Retry Policy）
4. 抽象 OpenCode 风格的 `agents / modes / skills / hooks / commands`
5. 再考虑可视化插件装配与团队模板市场

## 3. 当前版本的真实问题

## 3.1 当前 LLM 的角色仍是“建议生成器”

当前实现中，LLM 主要通过 `VolcengineAdvisor` 被挂载到流水线管理器，用于：

- 任务拆分
- 任务执行建议
- 任务完成判定
- 迭代修复建议
- 验收总结
- 构思稿 / 设计稿 / 复盘稿生成

这些能力是有价值的，但它们都属于 **point enhancement**，即“在固定流程某些节点调用一次模型”。

因此当前系统的主执行逻辑仍然是：

- 由 Studio 决定流程
- 由 `super-dev` CLI 执行命令
- 由 LLM 在局部提供建议文本

换句话说，目前 LLM 没有真正承担：

- 任务规划者
- 工具路由器
- 阶段判断者
- 失败恢复策略制定者
- 自我评估者
- 长程状态维护者

### 3.2 当前智能度不足的根因

当前“没感觉到智能”的核心原因，不在模型本身，而在系统结构：

1. **主链路是固定命令流，不是 Agent 决策流**
2. **上下文构建是规则式召回，不是证据驱动检索**
3. **没有计划-执行-观察-评估-重试的闭环**
4. **没有把工具调用与模型推理统一进一个状态机**
5. **没有细粒度步骤级观测，难以调优 Agent 行为**

### 3.3 当前版本真正值得保留的部分

当前产品设计并不失败，相反，它已经具备很好的“控制面”雏形：

- `Project`：稳定配置中心
- `ChangeBatch`：交付目标与边界
- `PipelineRun`：执行记录与追踪闭环
- `Context Hub`：记忆 / 知识 / Context Pack 聚合入口

这套设计非常适合挂载一个新的 Agent Core。

因此本次方案的原则不是替换控制面，而是增强执行面。

## 4. 三条路线对比

## 4.1 方案 A：继续基于现有设计增强

### 方案描述

保持当前 `Go + super-dev + SQLite` 架构不变，在 `pipeline.Manager` 内继续追加：

- 更多 prompt 模板
- 更多阶段判断逻辑
- 更多 LLM 辅助函数
- 更细的 fallback 规则

### 优点

- 改动小
- 接入成本低
- 对现有前后端影响最小

### 缺点

- 很容易继续把“智能”写成大量 if/else + prompt 调用
- `manager.go` 会继续膨胀
- 难以形成真正的 Agent 状态机
- 难以支持后续多 agent、技能、插件和人机协作

### 适用场景

- 仅希望补一两个智能能力
- 不追求真正自主运行
- 更重视短期交付而非中长期平台化

### 判断

不建议作为主路线，只适合作为过渡方案。

## 4.2 方案 B：接入 LangChain / LangGraph

### 方案描述

引入独立 `Agent Core` 服务，将以下能力从 Go 流水线层抽离：

- 任务规划
- 工具调用
- 检索链路
- 阶段状态机
- 评估与反思
- 自修复与重试策略

建议实际采用：

- `LangGraph`：负责编排与状态图
- `LangChain`：负责模型、工具、结构化输出、检索组件集成

### 优点

- 能直接实现真正的 Agent 闭环
- 更适合多步骤任务、长时运行和错误恢复
- 更容易加上 evaluator、critic、human-in-the-loop
- 更容易接入向量检索、rerank、structured output
- 更容易做步骤级 tracing 和行为回放

### 缺点

- 需要引入新的运行时，建议是 Python sidecar
- 增加跨服务通信复杂度
- 需要明确 Go 与 Python 的职责边界

### 适用场景

- 追求真正的自主运行
- 后续准备做多 agent / 复杂工作流 / 自我反思
- 希望后面可以扩展为平台能力

### 判断

这是本项目的推荐主路线。

## 4.3 方案 C：仿造 OpenCode / Oh-My-OpenCode 插件体系

### 方案描述

引入 OpenCode 风格的扩展抽象，为 Studio 增加：

- `agents`
- `modes`
- `skills`
- `hooks`
- `commands`
- `plugins`

让不同项目、不同团队、不同变更类型可以选择不同 Agent 行为模板。

### 优点

- 用户感知会更强，产品“智能味道”提升明显
- 便于做行业化、团队化定制
- 适合在 Studio 中做“Agent 角色市场”与“执行策略模板”

### 缺点

- 插件体系不等于自治能力
- 如果没有 Agent Core，插件只是包装 prompt 和命令
- 容易出现“看起来能配很多，但真正不会自主推进”的问题

### 适用场景

- 已经有稳定 Agent Core，希望增强可配置性和生态扩展性
- 需要产品层差异化与团队复用能力

### 判断

应作为第二层建设，而不是第一优先级。

## 4.4 结论对比表

| 维度 | 现有设计增强 | LangChain / LangGraph | 仿造 OpenCode / Oh-My-OpenCode |
| --- | --- | --- | --- |
| 短期改造成本 | 低 | 中高 | 中 |
| 自主运行能力 | 低 | 高 | 中 |
| 复杂流程编排 | 低 | 高 | 中 |
| 检索与工具统一建模 | 低 | 高 | 低-中 |
| 可观测性 | 低 | 高 | 中 |
| 可扩展性 | 中 | 高 | 高 |
| 产品感知 | 低-中 | 中 | 高 |
| 长期平台价值 | 低 | 高 | 中高 |

### 最终建议

采用组合式路线：

1. **底座**：保留现有 Studio 控制面
2. **核心**：引入 `LangGraph Agent Core`
3. **扩展**：借鉴 `OpenCode / Oh-My-OpenCode` 做插件层

## 5. 目标架构

## 5.1 总体架构原则

### 原则一：控制面与执行面分离

- `SuperDev Studio Go Backend` 负责控制面
- `Agent Core` 负责执行面

### 原则二：保留现有数据模型主干

不推翻以下实体：

- `Project`
- `ChangeBatch`
- `PipelineRun`
- `RunEvent`
- `Memory`
- `KnowledgeDocument`
- `KnowledgeChunk`

### 原则三：Agent 不直接取代 super-dev

`super-dev` 仍然是关键交付执行器；
Agent 的作用是：

- 决定何时调用哪种 `super-dev` 子命令
- 决定是否先检索 / 先补文档 / 先执行测试 / 先修复
- 根据结果判断下一步动作

### 原则四：先单 Agent，后多 Agent

第一阶段只做一个 `Delivery Agent`，避免过早引入多 agent 协同复杂度。

## 5.2 目标架构分层

```text
Frontend (React)
  └─ Studio UI

Go Backend (Control Plane)
  ├─ Project / ChangeBatch / PipelineRun API
  ├─ Context Hub API
  ├─ Artifact / Event / Preview API
  ├─ Tool Gateway for super-dev / file / context / preview
  └─ Agent Run Orchestrator Adapter

Python Agent Core (Execution Plane)
  ├─ LangGraph State Machine
  ├─ Planner
  ├─ Retriever
  ├─ Tool Router
  ├─ Evaluator / Critic
  ├─ Retry Policy
  └─ Skill / Mode / Hook Runtime

Persistence
  ├─ SQLite (existing)
  ├─ Vector Index (new, can be pgvector / qdrant / sqlite-vss in later phase)
  └─ Trace Store / Step Logs (can start in SQLite)
```

## 5.3 关键组件定义

### A. Studio Control Plane

保留现有 Go 后端，继续负责：

- 项目与变更管理
- 运行生命周期持久化
- 事件流展示
- 工件索引与预览
- UI API 与权限控制

### B. Agent Core

新增 Python 服务，负责：

- 接收一个 `run intent`
- 构建计划
- 检索上下文
- 选择工具并执行
- 观察执行结果
- 判断是否通过 / 是否重试 / 是否转人工
- 写回步骤级结构化轨迹

### C. Tool Gateway

建议由 Go 后端统一暴露工具能力，供 Agent Core 调用。工具应至少包括：

1. `search_context`
2. `read_run_history`
3. `read_artifact`
4. `write_memory`
5. `create_change_batch`
6. `run_superdev_create`
7. `run_superdev_spec_validate`
8. `run_superdev_task_status`
9. `run_superdev_task_run`
10. `run_superdev_quality`
11. `run_superdev_preview`
12. `run_superdev_deploy`
13. `collect_quality_summary`
14. `list_project_tasks`
15. `update_project_task_status`
16. `emit_run_event`

### D. Retrieval Layer

从当前 `FTS + LIKE` 升级为两层检索：

1. **初筛层**：关键词检索 / metadata filter
2. **重排层**：embedding similarity + rerank

最终返回给 Agent 的不是“拼接后的大文本”，而是：

- evidence 列表
- 来源标识
- relevance score
- 摘要与可引用片段

### E. Evaluator / Critic

用于回答四个问题：

1. 本轮动作是否达成当前子目标？
2. 当前结果是否满足进入下一阶段的条件？
3. 失败属于可自动修复还是需人工介入？
4. 还需要追加哪些检索与工具调用？

## 6. 推荐的实施形态

## 6.1 服务形态建议

建议新增目录：

```text
agent-core/
  app/
  graph/
  tools/
  retrievers/
  evaluators/
  skills/
  tests/
```

建议采用：

- Python 3.11+
- `langgraph`
- `langchain`
- 官方兼容 OpenAI API 的模型客户端
- HTTP 或 gRPC 与 Go 后端通信

### 为什么不直接把 Agent 写回 Go

原因不是 Go 不行，而是：

- 生态上，Agent 编排、structured output、retrieval、evaluation 的成熟工具链更多在 Python
- 后续做实验、prompt/version、graph 调参会更快
- 可以把高频变化的 Agent 逻辑与稳定的业务控制面解耦

## 6.2 与现有后端的边界

### Go 后端继续负责

- 数据存储
- 运行记录
- 工件管理
- 安全与权限
- 项目配置
- UI 查询接口

### Agent Core 负责

- 状态图执行
- 工具选择
- 动态规划
- 上下文选择
- 结果评估
- 重试决策

## 7. 最小可行版本（MVP）

## 7.1 MVP 目标

MVP 不追求一次性替代全部运行模式，只完成：

- 覆盖 `step_by_step` 模式
- 支持单个 `Delivery Agent`
- 支持关键工具调用
- 支持基本检索与评估闭环
- 支持运行轨迹回放

## 7.2 MVP 范围

### 输入

- `project_id`
- `change_batch_id`
- `prompt`
- `project defaults`
- `context mode`

### 输出

- 结构化计划
- 每步工具调用记录
- 关键证据引用
- 阶段决策说明
- 失败原因
- 下一步建议
- 最终运行总结

### 必需工具

- `search_context`
- `run_superdev_create`
- `run_superdev_task_status`
- `run_superdev_task_run`
- `run_superdev_quality`
- `read_artifact`
- `emit_run_event`

### 暂不纳入 MVP

- 多 agent 协同
- 自动部署上线
- 插件市场 UI
- 跨项目知识图谱
- 复杂权限隔离

## 7.3 MVP 状态机

建议实现如下 LangGraph 状态机：

```text
ReceiveRun
  -> BuildGoal
  -> RetrieveContext
  -> PlanNextAction
  -> ExecuteTool
  -> ObserveResult
  -> EvaluateOutcome
      -> Pass? -> AdvanceStage
      -> Retryable? -> RepairPlan
      -> NeedsHuman? -> Escalate
  -> Finish
```

### 关键说明

- `PlanNextAction` 必须输出结构化结果，而不是自由文本
- `EvaluateOutcome` 必须是独立节点，不能混在执行节点里
- 每一步必须写 trace，避免“黑盒智能”

## 8. 数据模型增量设计

在不破坏现有主模型的前提下，建议新增以下表：

## 8.1 `agent_runs`

记录一次 Agent 编排实例。

建议字段：

- `id`
- `pipeline_run_id`
- `project_id`
- `change_batch_id`
- `agent_name`
- `mode_name`
- `status`
- `current_node`
- `started_at`
- `finished_at`

## 8.2 `agent_steps`

记录状态图中的每一步。

建议字段：

- `id`
- `agent_run_id`
- `step_index`
- `node_name`
- `input_json`
- `output_json`
- `decision_summary`
- `status`
- `started_at`
- `finished_at`

## 8.3 `agent_tool_calls`

记录工具调用行为。

建议字段：

- `id`
- `agent_step_id`
- `tool_name`
- `request_json`
- `response_json`
- `success`
- `latency_ms`

## 8.4 `agent_retrieval_traces`

记录本次检索拿到了哪些证据。

建议字段：

- `id`
- `agent_step_id`
- `source_type`
- `source_id`
- `score`
- `snippet`
- `metadata_json`

## 8.5 `agent_evaluations`

记录 evaluator 的判断结果。

建议字段：

- `id`
- `agent_step_id`
- `evaluation_type`
- `verdict`
- `reason`
- `followup_action`

## 9. 接口设计建议

## 9.1 Studio -> Agent Core

新增内部接口：

### `POST /internal/agent/runs`

请求：

```json
{
  "pipeline_run_id": "run_xxx",
  "project_id": "proj_xxx",
  "change_batch_id": "cb_xxx",
  "prompt": "实现一个可追踪的交付 Agent",
  "mode": "step_by_step",
  "project_defaults": {
    "platform": "web",
    "frontend": "react",
    "backend": "go",
    "domain": "developer-tools"
  },
  "context": {
    "mode": "auto",
    "token_budget": 1600,
    "max_items": 10,
    "dynamic": true
  }
}
```

### `POST /internal/agent/runs/{id}/resume`

用于人工介入后继续执行。

### `POST /internal/agent/runs/{id}/cancel`

用于强制终止 Agent。

## 9.2 Agent Core -> Studio Tool Gateway

建议采用统一工具网关协议，而不是让 Agent 直接访问数据库。

例如：

### `POST /internal/tools/execute`

```json
{
  "tool_name": "run_superdev_quality",
  "arguments": {
    "project_dir": "D:/Work/target-project",
    "change_id": "agent-core-poc"
  }
}
```

返回：

```json
{
  "success": true,
  "summary": "quality passed with score 86",
  "artifacts": ["output/agent-core-poc-quality-gate.md"],
  "raw": {
    "stdout": ["..."],
    "exit_code": 0
  }
}
```

## 10. OpenCode 风格扩展层设计

## 10.1 设计目标

在 Agent Core 稳定后，再增加可配置扩展层，使不同项目可选择不同执行人格与策略。

## 10.2 建议抽象

### `agents`

定义角色，例如：

- `delivery-agent`
- `refactor-agent`
- `qa-agent`
- `release-agent`

### `modes`

定义运行模式，例如：

- `step_by_step`
- `full_cycle`
- `analysis_only`
- `repair_only`

### `skills`

定义专门能力包，例如：

- `super-dev-delivery`
- `golang-backend`
- `react-ui-polish`
- `quality-gate-remediation`

### `hooks`

定义节点前后钩子，例如：

- `before_plan`
- `before_tool_call`
- `after_quality`
- `on_failure`
- `on_finish`

### `commands`

定义面向 UI 或 CLI 的高阶动作，例如：

- `run_delivery_agent`
- `retry_failed_quality`
- `generate_release_brief`

## 10.3 配置落地形式

建议后续支持项目级配置目录：

```text
.studio-agent/
  agents/
  skills/
  hooks/
  modes/
  commands/
```

也可以后续合并进 Studio 的项目设置页进行可视化配置。

## 11. 分阶段实施路线图

## Phase 0：解耦准备（1 周）

### 目标

为引入 Agent Core 做结构准备，但不改变前端使用方式。

### 任务

1. 将现有 `pipeline.Manager` 中与 LLM 强相关的逻辑抽成独立模块接口
2. 抽出工具执行适配层，避免 `manager.go` 继续膨胀
3. 定义 Agent Run / Step / Tool Call 的持久化模型
4. 补充运行事件与步骤事件的映射规范

### 产出

- `backend/internal/agentbridge/`
- `backend/internal/tools/`
- 新增 SQLite migration

## Phase 1：Agent Core MVP（2-3 周）

### 目标

在 `step_by_step` 模式下，Agent 可以自主推进一轮完整的“检索 -> create -> task -> quality -> repair -> finish”流程。

### 任务

1. 新建 `agent-core/` Python 服务
2. 实现 LangGraph 最小状态机
3. 实现 6-8 个关键工具
4. 将 `PipelineRun` 与 `AgentRun` 关联
5. 在 UI 中展示 Agent 步骤轨迹

### 验收标准

- 能完成一次真实 `step_by_step` 运行
- 每一步有结构化 trace
- 失败时能自动进入 repair 分支
- 质量通过后能自动收敛结束

## Phase 2：检索升级（1-2 周）

### 目标

提升上下文质量，使 Agent 的动作有证据基础。

### 任务

1. 引入 Embedding 生成链路
2. 建立向量索引或向量存储
3. 实现混合检索（FTS + vector）
4. 增加 rerank
5. 返回 evidence trace 到 Agent 与前端

### 验收标准

- 检索结果可解释
- 同类需求的上下文命中率提升
- Agent 决策可以展示“依据哪些记忆/知识/历史运行”

## Phase 3：Evaluator 与自修复（1-2 周）

### 目标

让系统不止会执行，还会判断结果是否足够好。

### 任务

1. 增加 `outcome_evaluator`
2. 增加 `quality_failure_classifier`
3. 增加 `retry_policy`
4. 区分“自动修复”和“需人工介入”

### 验收标准

- 失败场景不会无脑重试
- 可区分可恢复与不可恢复错误
- 自动修复成功率可量化

## Phase 4：OpenCode 风格扩展层（2 周）

### 目标

支持项目级智能模板与技能复用。

### 任务

1. 定义 `agents / modes / skills / hooks / commands` 配置结构
2. 允许项目设置选择默认 Agent
3. 允许不同 `ChangeBatch` 绑定不同模式
4. 支持导入团队技能模板

### 验收标准

- 同一 Studio 中可运行多个风格不同的 Agent
- 项目智能行为可配置而非硬编码

## 12. 对现有代码的建议改造点

## 12.1 后端

建议新增或重构：

- `backend/internal/agentbridge/`：Go 与 Agent Core 通信
- `backend/internal/tools/`：统一工具注册与执行
- `backend/internal/trace/`：步骤级 trace 聚合
- `backend/internal/eval/`：评估结果持久化与对外展示

建议逐步瘦身：

- `backend/internal/pipeline/manager.go`
- `backend/internal/store/store.go`
- `backend/internal/api/server.go`

## 12.2 前端

新增 UI 区块：

1. `Agent Timeline`
   - 展示节点执行顺序
2. `Tool Calls`
   - 展示调用了哪些工具、输入输出是什么
3. `Evidence Panel`
   - 展示当前决策参考了哪些记忆、知识、历史运行
4. `Evaluation Panel`
   - 展示为什么通过、为什么重试、为什么转人工

## 13. 风险与缓解

## 13.1 风险：跨服务复杂度上升

### 缓解

- 先做内部 HTTP，不上复杂消息总线
- 先实现单 Agent、单项目串行执行

## 13.2 风险：Agent 行为不可控

### 缓解

- 工具白名单
- 结构化输出校验
- 最大步骤数限制
- 重试次数限制
- 人工接管开关

## 13.3 风险：上下文成本过高

### 缓解

- 缓存检索摘要
- 分层召回
- 限制 evidence 条数与 token budget

## 13.4 风险：前期投入后体验不明显

### 缓解

- MVP 阶段先把 Agent 轨迹、证据、评估显示出来
- 让用户明确看到系统“为什么这么做”

## 14. 成功标准

落地后至少要达到以下指标：

1. 用户能明显看出系统在“自主决定下一步”，而非只输出建议文本
2. `step_by_step` 模式下，Agent 可自动推进至少 70% 的标准流程
3. 每次运行都能回放：
   - 计划
   - 工具调用
   - 证据来源
   - 评估结果
4. 失败时系统能区分：
   - 自动修复
   - 需要补上下文
   - 需要人工介入
5. 智能行为可以通过配置迭代，而不是只能改代码

## 15. 本方案最终建议

### 最终路线

> **保留现有 Studio 控制面 + 引入 LangGraph Agent Core + 在第二阶段借鉴 OpenCode / Oh-My-OpenCode 做扩展层。**

### 不建议的路线

- 不建议只继续堆 prompt 与 fallback 逻辑
- 不建议一开始就做插件生态而忽略核心自治
- 不建议直接推翻现有 Go 后端重写为单体 Agent 应用

## 16. 建议的下一步变更批次

建议拆成三个连续变更批次：

1. `agent-core-foundation`
   - 建立 Agent Core 基础设施、数据模型和工具网关
2. `agent-step-by-step-mvp`
   - 完成 `step_by_step` 模式 Agent MVP
3. `agent-retrieval-and-skills`
   - 升级检索与技能层

## 17. 参考资料

以下资料适合作为后续实施时的一手参考：

- LangChain Overview: `https://docs.langchain.com/oss/python/langchain/overview`
- LangGraph Workflows and Agents: `https://docs.langchain.com/oss/python/langgraph/workflows-agents`
- OpenCode Agents: `https://opencode.ai/docs/agents`
- OpenCode Plugins: `https://opencode.ai/docs/plugins`
- OpenCode Skills: `https://opencode.ai/docs/skills`
- Oh-My-OpenCode: `https://ohmyopencode.org/`

