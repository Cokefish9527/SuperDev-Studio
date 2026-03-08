# Agent Core 推进表

## 总体状态

- Phase 0：方案与规格冻结 — 已完成
- Phase 1：Runtime Foundation — 已完成
- Phase 2：项目级 Agent 配置接入 — 已完成
- Phase 3：Evaluator / need_human / need_context — 已完成
- Phase 4：full_cycle Agent 化与治理收口 — 已完成

## 分阶段推进

### Phase 0：方案与规格冻结
- 状态：已完成
- 产物：Eino 方案、详细设计、开发计划、初始 specs
- 证据：`docs/AGENT_CORE_IMPLEMENTATION_PLAN_EINO_OPENCODE.md`

### Phase 1：Runtime Foundation
- 状态：已完成
- 范围：AgentRuntime、Eino Runtime、持久化模型、Retrieval MVP、Tool Gateway MVP、Pipeline Agent Observability
- 证据：`output/eino-agent-runtime-foundation-execution-report.md`

### Phase 2：项目级 Agent 配置接入
- 状态：已完成
- change_id：`agent-project-config-phase2`（已归档）
- 目标：
  - 增加项目级 `default_agent_name` / `default_agent_mode`
  - 提供项目级 Agent Bundle 查询接口
  - 在项目设置页和 Pipeline 运行页接入 agent/mode 选择
  - 让 step_by_step Agent Runtime 使用所选 agent/mode
- 完成标准：
  - 项目能保存默认 agent/mode
  - 运行页能覆盖选择 agent/mode
  - 启动 step_by_step 后 AgentRun 记录正确 agent/mode
  - `super-dev` 规格、任务、质量门通过
- 执行结果：
  - 后端完成 Project 默认 agent/mode 存储、Agent Bundle 查询与运行参数解析
  - 前端完成项目设置页与 Pipeline 运行页的 agent/mode 选择接入
  - 自动化验证通过：`go test ./internal/store ./internal/agentconfig ./internal/pipeline ./internal/api`、`npm test`、`npm run build`
  - 流水线门禁通过：`super-dev task run agent-project-config-phase2`、`super-dev quality --type all`
- 证据：`output/agent-project-config-phase2-execution-report.md`

### Phase 3：Evaluator 与人工分支
- 状态：已完成
- change_id：`agent-evaluator-human-phase3`（已归档）
- 目标：
  - 独立 `StepOutcomeEvaluator`
  - `QualityFailureClassifier`
  - `need_human` / `need_context` 分支闭环
  - 前端人工接管视图
- 完成标准：
  - `need_human` 不再直接终止为不可恢复失败，而是进入可恢复人工接管状态
  - `need_context` 具备显式上下文补强记录，并可在既有 step loop 中继续重试
  - 后端 API 输出最新 Evaluator 结果，前端可展示并驱动恢复操作
  - `super-dev` 规格、任务、质量门通过并完成归档
- 执行结果：
  - 后端完成 `need_human` / `need_context` 分支闭环、恢复执行入口与最新评估结果透出
  - 前端完成人工接管提示、恢复执行、补强上下文提示，以及运行详情弹窗化整理
  - 自动化验证通过：`go test ./internal/store ./internal/agentconfig ./internal/pipeline ./internal/api`、`npm test`、`npm run build`
  - 流水线门禁通过：`super-dev task run agent-evaluator-human-phase3`、`super-dev quality --type all`、`super-dev spec archive agent-evaluator-human-phase3`
- 证据：`output/agent-evaluator-human-phase3-execution-report.md`

### Phase 4：full_cycle 与治理收口
- 状态：已完成
- change_id：`agent-fullcycle-governance-phase4`（已归档）
- 目标：
  - `full_cycle` Agent 化接管
  - 高风险工具确认机制
  - 同运行审批后继续执行
  - 最终验收与发布说明收口
- 完成标准：
  - `full_cycle` 运行全过程生成 AgentRun / Step / Tool / Evaluation 轨迹
  - 高风险 `deploy` 动作进入待确认状态，而不是直接执行
  - 人工批准后在同一运行中继续，不新建 retry run
  - 前端能展示待确认提示、风险级别和继续执行入口
  - `super-dev` 规格、任务、质量门通过并完成归档
- 执行结果：
  - 后端完成 `full_cycle` 生命周期 Agent 化接管与高风险 deploy 审批暂停点
  - 新增 `approve-tool` API，并支持在同一 PipelineRun / AgentRun 中继续执行发布阶段
  - 前端完成 full-cycle 轨迹展示、高风险动作确认入口、运行摘要提示和交互测试
  - 自动化验证通过：`go test ./internal/store ./internal/agentconfig ./internal/pipeline ./internal/api`、`npx vitest run`、`npm run build`
  - 流水线门禁通过：`super-dev task run agent-fullcycle-governance-phase4`、`super-dev quality --type all`、`super-dev spec archive agent-fullcycle-governance-phase4`
- 证据：`output/agent-fullcycle-governance-phase4-execution-report.md`

## 收尾结论

- Agent Core 0-4 阶段全部完成。
- 当前已具备 step-by-step 与 full-cycle 两类 Agent 执行模式，以及人工接管、上下文补强、高风险工具审批等关键治理能力。
- 后续若继续推进，建议转入体验优化、工具风险分级扩展、覆盖率专项和发布运营能力建设。
