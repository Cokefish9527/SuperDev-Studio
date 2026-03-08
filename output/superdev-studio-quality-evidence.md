# superdev-studio - 质量证据刷新摘要

- 刷新时间：2026-03-08 16:59:16Z
- 覆盖率口径：core-backend-packages
- 核心后端覆盖率：63.2%
- 覆盖率回退原因：全量 go test ./... 被非稳定包阻塞，已回退到核心后端包集合。
- 红队总分：58/100
- 红队状态：未通过（当前以中低风险改进项为主）
- 质量门总分：93/100
- 质量门状态：未通过

## 关键结论

- 安全 high/critical：0
- 性能 high/critical：0
- 架构 high/critical：0
- 当前质量门可读取覆盖率 XML，并使用红队结果评估安全/性能维度。

## 产物路径

- 覆盖率 XML：`coverage\cobertura-coverage.xml`
- 红队报告：`output\superdev-studio-redteam.md`
- 质量门禁：`output\superdev-studio-quality-gate.md`

## 建议

- 修复: Spec 任务闭环状态
- 建议: 覆盖率报告
