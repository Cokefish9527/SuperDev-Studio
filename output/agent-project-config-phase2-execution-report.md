# agent-project-config-phase2 执行报告

## 1. 执行批次

- change_id：`agent-project-config-phase2`
- 标题：补齐项目级 Agent 配置与运行参数接入
- 执行日期：2026-03-08
- 流程：执行任务 → super-dev 流水线执行 → 执行报告落盘 → Git 提交

## 2. 本阶段完成内容

### 后端

- 为 `Project` 增加 `default_agent_name` / `default_agent_mode` 字段，并完成存储迁移与读写。
- 增加项目级 Agent Bundle 查询能力：`GET /api/projects/{projectID}/agent-bundle`。
- 为启动 Pipeline 与 advance / retry 流程增加 agent/mode 选择解析与校验。
- 将 `step_by_step` Agent Runtime 改为使用项目默认值或运行时覆盖值，而不再硬编码默认 agent/mode。

### 前端

- 项目设置页增加默认 Agent / Agent Mode 配置。
- Pipeline 运行页增加 `step_by_step Agent Strategy` 配置卡，支持按项目默认值预填并在运行时覆盖。
- 补充前端类型、API Client 与页面测试。

### 推进治理

- 补充 `docs/AGENT_CORE_ROLLOUT_BOARD.md`，作为 Agent Core 分阶段推进总表。
- 本阶段 change 已完成并归档到 `.super-dev/archive/agent-project-config-phase2/`。

## 3. 验证结果

### 代码验证

- 后端：`go test ./internal/store ./internal/agentconfig ./internal/pipeline ./internal/api`
- 前端：`npm test`
- 构建：`npm run build`

结果：全部通过。

### super-dev 流水线

- `super-dev task run agent-project-config-phase2`
  - 结果：10/10 任务完成
- `super-dev quality --type all`
  - 结果：80/100，通过门禁
- `super-dev spec archive agent-project-config-phase2`
  - 结果：归档完成

## 4. 影响范围

- `backend/internal/store/models.go`
- `backend/internal/store/store.go`
- `backend/internal/agentconfig/loader.go`
- `backend/internal/api/agent.go`
- `backend/internal/api/server.go`
- `backend/internal/pipeline/manager.go`
- `backend/internal/pipeline/manager_agent_helpers.go`
- `frontend/src/types.ts`
- `frontend/src/api/client.ts`
- `frontend/src/pages/ProjectSettingsPage.tsx`
- `frontend/src/pages/PipelinePage.tsx`
- `frontend/src/pages/PipelinePage.test.tsx`
- `docs/AGENT_CORE_ROLLOUT_BOARD.md`

## 5. 当前整体完成情况

- Phase 0：已完成
- Phase 1：已完成
- Phase 2：已完成
- Phase 3：未开始
- Phase 4：未开始

## 6. 下一阶段建议

建议下一批次进入 `Phase 3：Evaluator / need_human / need_context`，优先完成：

1. 将 `need_human` 从直接失败升级为可恢复的人工接管状态。
2. 将 `need_context` 升级为显式检索补强分支与重试闭环。
3. 增加前端人工接管/恢复视图与对应 API。
