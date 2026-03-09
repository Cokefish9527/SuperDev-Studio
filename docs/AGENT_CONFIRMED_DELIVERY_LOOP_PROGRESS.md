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
