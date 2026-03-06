# 测试报告

## 1. 测试环境

- 日期：2026-03-05
- OS：Windows (PowerShell)
- Go：`go1.23.2`
- Node：`v20.19.2`
- npm：`10.8.2`

## 2. 后端测试

执行命令：

```bash
cd backend
go test ./...
```

结果：通过。

通过包：

- `internal/api`
- `internal/contextopt`
- `internal/llm`
- `internal/pipeline`
- `internal/store`

关键覆盖场景：

1. `internal/api/server_test.go`
- `POST /api/pipeline/runs` 支持 `full_cycle` + `iteration_limit`，并在 `full_cycle=true` 时强制 `simulate=false`
- `POST /api/pipeline/runs` 在 `full_cycle=true` 且未传 `iteration_limit` 时默认使用 `3`
- `POST /api/pipeline/runs/{runID}/retry` 支持失败运行重试并保留关键配置（含 `full_cycle`）
- `GET /api/pipeline/runs/{runID}/completion` 返回完成清单与产物
- `GET /api/pipeline/runs/{runID}/preview/*` 支持预览页面访问

2. `internal/pipeline/manager_test.go`
- 一键全流程在质量门禁首次失败后可继续迭代，并在后续通过后完成交付
- 一键全流程在迭代次数耗尽仍未通过质量门禁时，正确失败并终止发布阶段
- 上下文注入（通用 + 阶段动态）与运行结束记忆回写

3. `internal/llm/volcengine_test.go`
- 火山引擎客户端启用条件校验
- Ark `/chat/completions` 请求结构与鉴权头校验
- 字符串与分片数组两种响应内容解析
- API 错误透传与空 prompt 校验

## 3. 前端测试

执行命令：

```bash
cd frontend
npm run test -- --run
```

结果：通过（6/6）。

通过用例：

- `DashboardPage.test.tsx`：1 项
- `ContextPage.test.tsx`：1 项
- `PipelinePage.test.tsx`：4 项
  - 动态上下文与记忆回写参数透传
  - 一键全流程参数透传（`full_cycle`、`iteration_limit`）与真实模式强制
  - 失败运行重试
  - 完成清单与预览入口展示

说明：测试输出含 Ant Design `List` 组件弃用告警，不影响功能正确性。

## 4. 前端静态检查

执行命令：

```bash
cd frontend
npm run lint
```

结果：通过。

## 5. 前端构建测试

执行命令：

```bash
cd frontend
npm run build
```

结果：通过，成功产出 `dist/`。

说明：

- 构建提示主包体积较大（`~1.19MB`），建议后续增加路由懒加载和手动分包。

## 6. 端口与访问验证

执行命令：

```bash
Invoke-WebRequest http://127.0.0.1:5273
```

结果：`HTTP_STATUS=200`。

补充：再次执行 `npm run dev -- --host 127.0.0.1 --port 5273` 时提示端口占用，说明 `5273` 已有前端服务在运行。

## 7. 结论

当前版本已实现并验证：

- React + Go + SQLite 架构稳定可运行
- super-dev 一键全流程交付（设计 -> 开发/单测/修复迭代 -> 测试 -> 验收 -> 准备上线）
- 火山引擎 Ark LLM 可用于迭代修复建议与验收总结（配置后生效）
- 记忆模块、知识库、上下文优化与流水线动态联动
- 失败运行重试、完成清单、产物列表、页面预览能力可用

未在本次自动化中执行：

- 使用真实 `SUPER_DEV_CMD` 对外部项目仓库进行长链路集成运行
- 使用真实 `VOLCENGINE_ARK_API_KEY` 发起线上模型调用

以上两项需要在具备外部依赖与凭据的环境进行端到端验收。
