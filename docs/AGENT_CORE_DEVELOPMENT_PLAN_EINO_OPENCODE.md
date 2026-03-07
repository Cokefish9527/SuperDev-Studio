# SuperDev Studio Eino + OpenCode 开发计划

## 1. 计划目标

本文将 `Eino + OpenCode` 方案拆解为可按 `super-dev` 变更流程执行的分阶段开发计划。

目标不是一次性重构所有 Agent 逻辑，而是以最短路径完成：

1. Agent Runtime 基础设施
2. `step_by_step` Agent MVP
3. 检索与评估增强
4. OpenCode 风格扩展层
5. 前端可观测性与治理能力

## 2. 总体里程碑

### Milestone 0：设计冻结

目标：

- 文档落盘
- 变更提案建立
- 规格与任务完成拆解

### Milestone 1：Runtime Foundation

目标：

- 建立 `AgentRuntime` 抽象
- 引入 `Eino` 基础运行时
- 具备最小 AgentRun / Step 追踪模型

### Milestone 2：step_by_step MVP

目标：

- Agent 可在 `step_by_step` 中执行计划、调用工具、做阶段评估

### Milestone 3：Retrieval + Evaluator

目标：

- 检索从规则式 Context Pack 升级为 evidence-driven
- 增加 repair 与人工接管机制

### Milestone 4：OpenCode-style Extension Layer

目标：

- 支持 `agents / modes / skills / hooks / commands`

### Milestone 5：UI + Governance + Rollout

目标：

- Pipeline 页面可观察 Agent 轨迹
- 完成质量门禁、文档和回归验证

## 3. 分阶段任务

## Phase 0：方案固化与接口冻结

### 目标

完成文档、提案、规格和实现边界定义。

### 任务

1. 新增 Eino 方案文档
2. 新增详细设计文档
3. 新增开发计划文档
4. 创建 `super-dev` 变更提案
5. 添加核心规格要求
6. 拆解阶段任务

### 验收标准

- 文档齐全
- change proposal 可查看
- specs 可 validate
- tasks 可进入执行状态

## Phase 1：Runtime Foundation

### 目标

在 Go 后端内引入 Agent Runtime 基础骨架。

### 任务

1. 新建 `backend/internal/agentruntime/`
2. 定义 `AgentRuntime` 接口与类型
3. 新建 `backend/internal/agentruntime/eino/`
4. 新增 `agent_runs` / `agent_steps` / `agent_tool_calls` 迁移
5. 增加 runtime registry 与 feature flag
6. 将 `PipelineRun` 与 `AgentRun` 关联

### 验收标准

- 能创建一条 AgentRun 记录
- 能记录至少一个 Step
- 不影响现有非 Agent 流程

## Phase 2：Tool Gateway

### 目标

将 Agent 所需外部能力统一收口为工具接口。

### 任务

1. 新建 `backend/internal/tools/registry.go`
2. 封装 `super-dev` create/spec/task/quality/preview/deploy 工具
3. 封装 context / artifact / project 工具
4. 为工具定义输入输出 schema
5. 为工具增加日志和超时控制

### 验收标准

- Agent 能通过统一 gateway 调用工具
- 工具结果可结构化持久化

## Phase 3：step_by_step Agent MVP

### 目标

让 Agent 真正接管 `step_by_step` 主链路。

### 任务

1. 建立 Eino graph 节点
2. 实现 `load profile -> retrieve -> plan -> execute -> evaluate`
3. 打通 `task run` / `quality` 的 repair loop
4. 增加 basic interrupt/resume
5. 将 agent 结果写回 RunEvent

### 验收标准

- Agent 能自主执行一轮 step-by-step
- 失败时能进入 repair loop
- UI 可看到 Agent 当前步骤

## Phase 4：Retrieval + Evaluator

### 目标

让 Agent 决策建立在 evidence 和 evaluator 之上。

### 任务

1. 新建 `backend/internal/retrieval/`
2. 实现 FTS + embedding + rerank 混合检索
3. 新建 `backend/internal/evaluator/`
4. 实现 `StepOutcomeEvaluator`
5. 实现 `QualityFailureClassifier`
6. 实现 `need_context / need_human` 分支

### 验收标准

- 决策可解释
- evidence 可回放
- 失败分类准确可用

## Phase 5：OpenCode-style Extension Layer

### 目标

让 Agent 行为从硬编码变成可配置。

### 任务

1. 新建 `backend/internal/agentconfig/`
2. 支持读取 `.studio-agent/agents/*.yaml`
3. 支持读取 `.studio-agent/modes/*.yaml`
4. 支持读取 `.studio-agent/skills/*.yaml`
5. 支持 hooks / commands 注册
6. 项目设置页增加默认 agent/mode 选择

### 验收标准

- 不改代码即可切换不同 agent 策略
- 项目级 agent 配置可生效

## Phase 6：UI + 治理 + 收口

### 目标

补齐产品可观测性与上线前治理能力。

### 任务

1. Pipeline 页面增加 Agent Timeline
2. 增加 Tool Calls 展示
3. 增加 Evidence 面板
4. 增加 Evaluator 面板
5. 增加高风险工具人工确认
6. 完成测试、文档、质量门禁和回归验证

### 验收标准

- 前端可清晰解释 Agent 行为
- 质量门禁通过
- 核心路径有自动化验证

## 4. 建议任务编组

为减少并行冲突，建议按以下顺序推进：

1. `foundation` 组
2. `tools` 组
3. `runtime-mvp` 组
4. `retrieval-eval` 组
5. `extension-layer` 组
6. `ui-observability` 组
7. `validation-rollout` 组

## 5. 建议变更拆分

如果单个 change 过大，建议再拆成三个连续 change：

1. `agent-runtime-foundation`
2. `agent-step-mvp`
3. `agent-extensibility-and-ui`

本轮先以 `eino-opencode-agent-core` 作为总规划 change。

## 6. 风险控制

### 风险一：Eino 接入成本高于预期

缓解：

- 保留 `AgentRuntime` 接口
- 先用最小 graph 骨架，不一次引入多 agent

### 风险二：检索质量不稳定

缓解：

- 先做 evidence trace
- 先做 FTS + rerank，再逐步增强 embedding

### 风险三：UI 太复杂

缓解：

- 先展示步骤和结论，再逐步开放详细输入输出

## 7. 本轮完成定义

本轮“规划完成”应满足：

- 文档已落盘
- `super-dev` change 已创建
- 核心规格已补齐
- 分阶段任务已拆解
- 可进入 `task status / task run` 阶段

## 8. 下一步建议

按照 `super-dev` 最短路径，接下来优先推进：

1. `Phase 1: Runtime Foundation`
2. `Phase 2: Tool Gateway`
3. `Phase 3: step_by_step Agent MVP`

在未冻结接口前，不建议直接跳到 UI 或多 Agent 设计。
