# Agent Core 推进表

## 总体状态

- Phase 0：方案与规格冻结 — 已完成
- Phase 1：Runtime Foundation — 已完成
- Phase 2：项目级 Agent 配置接入 — 已完成
- Phase 3：Evaluator / need_human / need_context — 已完成
- Phase 4：full_cycle Agent 化与治理收口 — 待执行

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
- 状态：待执行
- 目标：
  - `full_cycle` Agent 化接管
  - 高风险工具确认机制
  - 安全/性能/覆盖率专项收口
  - 最终验收与发布说明
