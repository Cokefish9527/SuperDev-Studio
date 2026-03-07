# SuperDev Studio Eino + OpenCode 详细设计

## 1. 设计目标

本文面向实施，描述 `SuperDev Studio` 如何在现有 Go 服务端内引入 `Eino Agent Runtime`，并以 `OpenCode` 风格抽象扩展 Agent 行为。

本文重点回答：

1. 代码层如何组织
2. 运行时如何编排
3. 工具、检索、评估如何接入
4. 配置与扩展层如何设计
5. UI 和存储如何配合展示 Agent 运行过程

## 2. 模块分层

建议在后端新增如下目录：

```text
backend/internal/
  agentruntime/
    runtime.go
    types.go
    registry.go
    eino/
      runtime.go
      graph.go
      state.go
      planner.go
      evaluator.go
      hooks.go
      checkpoint.go
  tools/
    registry.go
    gateway.go
    superdev.go
    context.go
    artifacts.go
    project.go
  retrieval/
    service.go
    lexical.go
    vector.go
    rerank.go
  agenttrace/
    service.go
    store.go
  agentconfig/
    loader.go
    schema.go
    validators.go
```

## 3. 运行时接口设计

## 3.1 `AgentRuntime`

建议定义统一接口：

```go
type AgentRuntime interface {
    Start(ctx context.Context, req StartAgentRunRequest) (AgentRun, error)
    Resume(ctx context.Context, runID string, input ResumeInput) error
    Cancel(ctx context.Context, runID string) error
    GetState(ctx context.Context, runID string) (AgentRunState, error)
}
```

设计意图：

- 让 `pipeline.Manager` 不直接依赖 `Eino`
- 后续如果需要补充 `LangGraph` 或测试实现，可替换 runtime

## 3.2 `StartAgentRunRequest`

建议字段：

- `PipelineRunID`
- `ProjectID`
- `ChangeBatchID`
- `Prompt`
- `Mode`
- `ProjectDefaults`
- `ContextOptions`
- `SelectedAgent`
- `SelectedSkills`
- `SelectedHooks`

## 4. Eino Runtime 设计

## 4.1 状态对象

建议状态对象包含：

- `Goal`
- `CurrentStage`
- `CurrentTask`
- `Plan`
- `Evidence`
- `ToolHistory`
- `QualitySummary`
- `RetryCount`
- `HumanGate`
- `FinalDecision`

## 4.2 图节点设计

建议第一阶段图节点如下：

```text
Init
  -> LoadProjectProfile
  -> LoadAgentConfig
  -> RetrieveEvidence
  -> PlanNextStep
  -> ExecuteTool
  -> ObserveToolResult
  -> EvaluateStep
      -> NeedRepair      -> BuildRepairPlan -> ExecuteTool
      -> NeedMoreContext -> RetrieveEvidence
      -> NeedHuman       -> Interrupt
      -> StepDone        -> AdvanceStage
  -> FinalizeRun
```

### 节点说明

#### `LoadProjectProfile`
读取：

- 项目默认技术栈
- 当前 `ChangeBatch`
- 历史 `PipelineRun`
- 最近任务状态

#### `LoadAgentConfig`
加载：

- `agent`
- `mode`
- `skills`
- `hooks`
- `commands`

#### `RetrieveEvidence`
检索来源：

- Memory
- Knowledge
- 历史运行总结
- 项目任务
- 工件摘要

返回结果必须结构化，而不是仅拼接文本。

#### `PlanNextStep`
输出结构必须至少包含：

- `objective`
- `selected_tool`
- `tool_args`
- `expected_outcome`
- `fallback_strategy`

#### `EvaluateStep`
产出至少包括：

- `verdict`: pass / retry / need_context / need_human / fail
- `reason`
- `next_action`

## 4.3 中断与恢复

优先支持以下三类中断：

1. 高风险动作前中断
2. 重试次数超过阈值时中断
3. 工具结果不确定时中断

恢复时应允许：

- 用户补充上下文
- 用户修改 Agent 指令
- 用户强制跳过当前阶段

## 5. Tool Gateway 设计

## 5.1 工具分类

### A. `super-dev` 工具

- `run_superdev_create`
- `run_superdev_spec_validate`
- `run_superdev_task_status`
- `run_superdev_task_run`
- `run_superdev_quality`
- `run_superdev_preview`
- `run_superdev_deploy`

### B. 上下文工具

- `search_context`
- `list_project_memories`
- `search_knowledge`
- `read_recent_runs`
- `build_context_pack`

### C. 项目工具

- `list_project_tasks`
- `update_task_status`
- `create_change_batch`
- `append_run_event`

### D. 工件工具

- `read_artifact`
- `list_artifacts`
- `summarize_output`

## 5.2 工具执行约束

每个工具必须定义：

- 输入 schema
- 输出 schema
- 是否幂等
- 是否高风险
- 是否允许自动重试
- 默认超时

## 5.3 工具注册结构

建议统一注册：

```text
ToolDefinition
  ├─ Name
  ├─ Description
  ├─ InputSchema
  ├─ OutputSchema
  ├─ RiskLevel
  ├─ RetryPolicy
  └─ Execute()
```

## 6. 检索设计

## 6.1 第一阶段检索策略

第一阶段不必马上引入独立向量数据库，可采用分层策略：

1. SQLite FTS 初筛
2. Embedding 相似度补充召回
3. Rerank 得到最终 evidence 列表

## 6.2 Evidence 数据结构

建议 Agent 消费以下结构：

```json
{
  "source_type": "memory|knowledge|run|artifact|task",
  "source_id": "...",
  "title": "...",
  "snippet": "...",
  "score": 0.92,
  "metadata": {}
}
```

## 6.3 检索原则

- 不把所有上下文直接灌给模型
- 先检索，再压缩，再让 Agent 决策
- 所有决策都应能回溯到 evidence

## 7. 评估器设计

## 7.1 评估器角色

评估器不直接负责执行，它负责判断：

1. 这一步是否达成目标
2. 当前结果是否可接受
3. 是否需要 repair
4. 是否需要人工介入

## 7.2 评估器分类

建议最少实现三类：

- `StepOutcomeEvaluator`
- `QualityFailureClassifier`
- `CompletionEvaluator`

## 7.3 判定结果

统一返回：

- `pass`
- `retry`
- `need_context`
- `need_human`
- `fail`

## 8. OpenCode 风格扩展层设计

## 8.1 配置目录

建议支持项目级目录：

```text
.studio-agent/
  agents/
  modes/
  skills/
  hooks/
  commands/
```

## 8.2 Agent 配置

示例字段：

- `name`
- `description`
- `default_model`
- `planner_prompt`
- `evaluator_prompt`
- `allowed_tools`
- `default_skills`
- `max_steps`

## 8.3 Mode 配置

例如：

- `step_by_step`
- `full_cycle`
- `analysis_only`
- `repair_only`

Mode 负责控制：

- 默认节点路径
- 是否允许 deploy
- 最大重试次数
- 是否强制人工确认

## 8.4 Skill 配置

每个 skill 包含：

- 适用场景
- 提示词片段
- 工具白名单
- 检索偏好
- 输出约束

## 8.5 Hook 配置

支持：

- `before_plan`
- `after_plan`
- `before_tool`
- `after_tool`
- `after_quality`
- `on_failure`
- `on_finish`

## 8.6 Command 配置

面向高阶业务动作，例如：

- `run_delivery_agent`
- `retry_quality_failures`
- `generate_release_summary`

## 9. 数据模型设计

建议新增如下表：

### `agent_runs`

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

### `agent_steps`

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

### `agent_tool_calls`

- `id`
- `agent_step_id`
- `tool_name`
- `request_json`
- `response_json`
- `success`
- `latency_ms`

### `agent_evidence`

- `id`
- `agent_step_id`
- `source_type`
- `source_id`
- `score`
- `snippet`
- `metadata_json`

### `agent_evaluations`

- `id`
- `agent_step_id`
- `evaluation_type`
- `verdict`
- `reason`
- `next_action`

## 10. API 设计

## 10.1 后端内部接口

- `POST /internal/agent/runs`
- `POST /internal/agent/runs/{id}/resume`
- `POST /internal/agent/runs/{id}/cancel`
- `GET /internal/agent/runs/{id}`
- `GET /internal/agent/runs/{id}/steps`
- `GET /internal/agent/runs/{id}/tool-calls`
- `GET /internal/agent/runs/{id}/evidence`

## 10.2 前端展示接口

- `GET /api/pipeline/runs/{runID}/agent`
- `GET /api/pipeline/runs/{runID}/agent/steps`
- `GET /api/pipeline/runs/{runID}/agent/tool-calls`
- `GET /api/pipeline/runs/{runID}/agent/evidence`

## 11. 前端页面设计

在现有 Pipeline 详情页增加四个面板：

1. `Agent Timeline`
2. `Tool Calls`
3. `Evidence`
4. `Evaluator`

展示原则：

- 默认简洁
- 点击展开查看输入输出
- 强调“为什么这样做”

## 12. 安全与治理设计

## 12.1 工具白名单

不同 agent / mode / skill 对应不同工具白名单。

## 12.2 风险级别

工具按风险级别划分：

- `low`
- `medium`
- `high`

高风险工具默认需要人工确认。

## 12.3 资源限制

- 最大步骤数
- 最大重试次数
- 单步超时
- 单运行 token 预算

## 13. 迁移策略

### Phase A
只接入 `step_by_step`，不替换原 `full_cycle`

### Phase B
让 `step_by_step` 支持 Agent 化 repair loop

### Phase C
再接管 `full_cycle` 的阶段决策与重试策略

## 14. 参考资料

- `docs/AGENT_CORE_IMPLEMENTATION_PLAN.md`
- `docs/AGENT_CORE_IMPLEMENTATION_PLAN_EINO_OPENCODE.md`
- Eino 官方文档与 OpenCode 官方文档
