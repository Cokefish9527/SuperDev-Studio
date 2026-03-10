# Agent Confirmed Delivery Loop - 执行报告（2026-03-09）

## 本轮目标

完成“用户输入简单需求 -> 系统生成需求草案 -> 用户确认 -> 自动启动 super-dev 交付 -> 页面可查看运行结果”的第一阶段闭环。

## 已完成内容

### 1. 需求确认链路

- 新增并打通 `requirement_sessions`、`requirement_doc_versions`、`requirement_confirmations` 的完整后端读写流程。
- `RequirementSession` 现在持久化记录：
  - `latest_change_batch_id`
  - `latest_run_id`
- 需求确认后，系统会自动按项目默认模式触发一次项目推进：
  - `default_agent_mode=full_cycle` -> `full_cycle`
  - 其他情况 -> `step_by_step`

### 2. LLM 需求草案生成

- API 层现在可复用现有 `LLMAdvisor` 为需求输入生成结构化草案。
- 要求模型输出严格 JSON：
  - `summary`
  - `prd`
  - `plan`
  - `risks`
- 当 LLM 不可用或返回不可解析内容时，自动回退到本地兜底草案生成逻辑。

### 3. 简单交付入口

已将普通用户主流程收敛到 `frontend/src/pages/SimpleDeliveryPage.tsx`：

- 输入一句需求
- 查看系统理解后的摘要 / PRD / 计划 / 风险
- 点击确认并启动交付
- 在同页查看运行状态 / 进度 / 预览入口
- 可一键跳转到 `PipelinePage` 查看完整 Agent / Pipeline 过程

### 4. 自动结果联动

- 确认成功后，页面可直接看到：
  - 自动创建的 `change_batch`
  - 自动启动的 `pipeline run`
  - 运行状态、阶段、进度
  - 如果已生成预览，可直接打开预览
- 若自动启动失败，会保留确认结果并向前端返回 `delivery_error`。

## 验证结果

### 后端

执行：

- `go test ./internal/store ./internal/api -run Requirement`
- `go test ./internal/app ./internal/pipeline -run TestDoesNotExist`

结果：通过。

### 前端

执行：

- `npm run build`

结果：通过。

## 本轮涉及文件

- `backend/internal/api/server.go`
- `backend/internal/api/server_test.go`
- `backend/internal/app/app.go`
- `backend/internal/store/models.go`
- `backend/internal/store/store.go`
- `frontend/src/api/client.ts`
- `frontend/src/pages/SimpleDeliveryPage.tsx`
- `frontend/src/types.ts`
- `.super-dev/changes/agent-confirmed-delivery-loop/tasks.md`

## 当前仍未完成的重点能力

### 1. LLM + super-dev 深度编排闭环

当前已实现“确认后自动启动一次交付”，但还未完全实现：

- LLM 根据阶段结果持续评估 `verdict / next_action`
- 自动提取残留项并形成可持续跟踪的 backlog
- 根据质量门禁/预览结果继续派发下一轮 `super-dev task run`

### 2. 残留问题跟踪

设计里规划的以下实体尚未落地：

- `residual_items`
- `approval_gates`
- `preview_sessions`

### 3. 运行过程文档化增强

当前用户可看到运行结果和预览入口，但尚未形成完整的：

- 过程文档时间线
- 分阶段执行报告自动归档
- 面向普通用户的阶段总结页

## 建议下一步

优先进入第二阶段：

1. 落地 `residual_items` 存储与 API
2. 将 Agent 评估结果自动转换为待修复项
3. 在 `PipelinePage` / 简单入口展示“未完成项 / 已解决项 / 需人工确认项”
4. 将 `super-dev task run` 与 `quality --type all` 串成可重复修复闭环

---

# Agent Confirmed Delivery Loop - 执行报告（2026-03-10）

## 本轮目标

完成“LLM 评估 -> 残留项/审批闸口持久化 -> PipelinePage 可追踪与人工闭环”的第二阶段落地。

## 已完成内容

### 1. 残留项与审批闸口持久化

- 新增 `ResidualItem` 与 `ApprovalGate` 数据模型，并补齐数据库迁移能力。
- 新增 `backend/internal/store/followups.go`，提供：
  - 残留项 upsert / list / status update
  - 审批闸口 upsert / list / auto resolve
- Run 级别的 follow-up 状态不再只靠前端推断，而是落库存储，便于持续跟踪。

### 2. 基于运行状态自动同步 follow-ups

- 新增 `backend/internal/api/followups.go`，提供以下 API：
  - `GET /api/projects/{projectID}/residual-items`
  - `GET /api/projects/{projectID}/approval-gates`
  - `GET /api/pipeline/runs/{runID}/residual-items`
  - `GET /api/pipeline/runs/{runID}/approval-gates`
  - `PATCH /api/residual-items/{itemID}`
- 同步逻辑会根据当前 run / agent evaluation / tool approval 自动生成或更新：
  - `need_context` -> requirement 类残留项
  - `need_human` -> 高优先级残留项
  - `awaiting_approval` 高风险工具调用 -> approval gate
- `AgentEvaluation` 结构现已扩展为：
  - `verdict`
  - `reason`
  - `next_action`
  - `missing_items[]`
  - `acceptance_delta`
- `missing_items[]` 会进一步拆解为具体 residual items，避免“知道有问题但不知道差什么”。
- 同步生成的旧残留项/旧 gate 在不再活跃时会自动转为已解决，避免页面长期残留脏状态。

### 3. PipelinePage 跟进视图

- `frontend/src/types.ts` 与 `frontend/src/api/client.ts` 已补齐 follow-up 类型与接口。
- `frontend/src/pages/PipelinePage.tsx` 已接入：
  - run residual items 查询
  - run approval gates 查询
  - 手动将 residual item 标记为 `resolved`
- 页面现在可以直接看到：
  - 当前未完成残留项
  - 待人工确认的高风险闸口
  - 人工关闭残留项后的即时刷新结果

### 4. 覆盖测试

- 新增 API 测试，覆盖：
  - `need_context` 运行后可拉取 residual items
  - 高风险 deploy 审批等待时可拉取 approval gates
  - `PATCH /api/residual-items/{id}` 可将残留项标记为 resolved
- 新增 Store 测试，覆盖 `missing_items` / `acceptance_delta` 的持久化与读取。

## 验证结果

### 后端

执行：

- `go test ./internal/store ./internal/api -run 'Residual|Approval|NeedContext|NeedHuman'`

结果：通过。

补充执行：

- `go test ./...`

结果：本轮改动相关包通过；但仓库内既有 `backend/src/main_test.go` 仍因 `searchHandler` / `analyticsHandler` / `notificationHandler` 缺失而失败，此问题与本轮 follow-up 改动无关。

### 前端

执行：

- `npm run build`

结果：通过。

## 本轮涉及文件

- `backend/internal/api/followups.go`
- `backend/internal/api/server.go`
- `backend/internal/api/server_test.go`
- `backend/internal/store/followups.go`
- `backend/internal/store/models.go`
- `backend/internal/store/store.go`
- `frontend/src/api/client.ts`
- `frontend/src/pages/PipelinePage.tsx`
- `frontend/src/types.ts`
- `.super-dev/changes/agent-confirmed-delivery-loop/tasks.md`

## 当前仍未完成的重点能力

### 1. LLM + super-dev 的自动派工闭环

当前已具备“评估 -> 残留跟踪 -> 人工关闭/审批”的基础能力，但仍未完全实现：

- LLM 根据 `verdict / next_action / acceptance_delta` 决定下一条 `super-dev` 命令
- 自动 repair loop（例如继续 `task run` / `quality` / `preview`）
- 质量门禁失败后的残留聚合与再次派工

### 2. 预览与验收闭环

以下能力仍待补齐：

- `preview_sessions` 持久化
- 预览 URL / 预览快照的历史追踪
- 最终验收页与“预上线版本”交付确认

### 3. 面向普通用户的极简主流程收敛

`SimpleDeliveryPage` 已完成第一阶段，但还可以继续收敛成更明确的三段式：

- 输入需求
- 确认系统理解
- 查看交付结果 / 预览 / 剩余问题

## 建议下一步

下一阶段建议直接推进：

1. 将 follow-up 结果回灌到 LLM evaluator，形成 `next_command` 决策
2. 接入 `super-dev task run` / `super-dev quality --type all` 的自动 repair loop
3. 新增 `preview_sessions` 与最终验收记录
4. 在简单交付页合并“需求确认 + 交付进展 + 验收结果”主链路

---

# Preview Acceptance Loop - 执行报告（2026-03-10）

## 本轮目标

完成“预览会话持久化 -> 预览历史可追踪 -> 人工验收确认”的闭环，补齐交付末端的预览与验收能力。

## 已完成内容

### 1. 预览会话持久化

- 新增 `preview_sessions` 数据模型与数据库表。
- 新增 `backend/internal/store/preview_sessions.go`，支持：
  - preview session upsert
  - project/run 维度列表查询
  - 验收状态更新（`generated / accepted / rejected`）
- 系统会基于 `pipeline completion` 自动提取主预览 URL 并写入 preview session。

### 2. 预览与验收 API

- 新增 API：
  - `GET /api/projects/{projectID}/preview-sessions`
  - `GET /api/pipeline/runs/{runID}/preview-sessions`
  - `PATCH /api/preview-sessions/{sessionID}`
- run 级接口会在返回前自动同步 preview session，保证页面拿到的是最新预览状态。

### 3. PipelinePage 验收入口

- `PipelinePage` 新增“预览与验收”卡片。
- 支持直接看到：
  - 当前 run 的 preview session 列表
  - 预览状态
  - 预览更新时间
  - 评审备注
- 支持操作：
  - 打开预览
  - 验收通过
  - 退回修改

### 4. 覆盖测试

- 新增 API 测试，覆盖：
  - completed run 可自动同步 preview session
  - `PATCH /api/preview-sessions/{id}` 可将预览标记为 accepted

## 验证结果

### 后端

执行：

- `go test ./internal/store ./internal/api -run 'Preview|Completion'`

结果：通过。

### 前端

执行：

- `npm run build`

结果：通过。

## 本轮涉及文件

- `backend/internal/api/preview_sessions.go`
- `backend/internal/api/server.go`
- `backend/internal/api/server_test.go`
- `backend/internal/store/models.go`
- `backend/internal/store/preview_sessions.go`
- `backend/internal/store/store.go`
- `frontend/src/api/client.ts`
- `frontend/src/pages/PipelinePage.tsx`
- `frontend/src/pages/PipelinePage.test.tsx`
- `frontend/src/types.ts`
- `.super-dev/changes/preview-acceptance-loop/tasks.md`

## 当前仍未完成的重点能力

### 1. 自动派工与 repair loop

当前系统已经能：

- 生成需求草案
- 启动交付
- 跟踪 residual / approval gate / preview acceptance

但还未完全做到：

- LLM 根据残留项自动决定下一条 `super-dev` 命令
- 自动触发下一轮 `task run / quality / preview`
- 直到产品完成前持续推进而非主要依赖人工点击

### 2. 极简主流程进一步收敛

目前普通用户的主路径还可继续收敛为：

- 输入一句需求
- 确认系统理解
- 查看结果 / 预览 / 剩余问题 / 验收状态

## 建议下一步

建议下一阶段优先推进：

1. 将 `residual_items + approval_gates + preview_sessions` 汇总为统一 delivery checklist
2. 让 LLM evaluator 输出 `next_command` 并自动驱动下一轮 `super-dev`
3. 在 `SimpleDeliveryPage` 合并“交付结果 + 预览验收 + 剩余问题”三段式主链路
