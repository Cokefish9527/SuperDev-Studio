# agent-evaluator-human-phase3 - Pipeline 布局迭代记录

- 时间: `2026-03-08 04:08:02 +08:00`
- Active Change: `agent-evaluator-human-phase3`
- 迭代主题: `Pipeline 单屏布局平衡 + 运行详情多标签弹窗`

## 本轮改动

- 将 `PipelinePage` 的完整运行详情从主页面收起，改为通过 `查看运行详情` 打开多标签弹窗。
- 主页面保留启动区、运行列表、当前选中运行摘要卡，减少长内容对单屏布局的挤压。
- 详情弹窗拆分为：`概览`、`阶段产物`、`产物预览`、`执行轨迹`、`Agent 轨迹`。
- 时间线卡片补齐分页摘要、返回顶部与稳定测试选择器。
- Pipeline 页面回归测试改为围绕“摘要卡 -> 弹窗 -> 标签页”路径验证。

## 涉及文件

- `frontend/src/pages/PipelinePage.tsx`
- `frontend/src/pages/PipelinePage.test.tsx`
- `frontend/src/components/pipeline/PipelineRunDetailsModal.tsx`
- `frontend/src/components/pipeline/PipelineTimelineCard.tsx`
- `.super-dev/changes/agent-evaluator-human-phase3/proposal.md`
- `.super-dev/changes/agent-evaluator-human-phase3/tasks.md`
- `.super-dev/changes/agent-evaluator-human-phase3/specs/pipeline-workspace/spec.md`

## 验证结果

- `frontend: npm test -- --run PipelinePage` -> 通过
- `frontend: npm test` -> 通过（7 files, 15 tests）
- `frontend: npm run build` -> 通过
- `super-dev spec validate` -> 通过

## 备注

- 当前 active change 仍未整体收口；本轮仅完成 Pipeline 页面布局与交互层优化。
- 后续可继续在同一 change 内补齐 `need_human / need_context / evaluator` 的前后端闭环。
