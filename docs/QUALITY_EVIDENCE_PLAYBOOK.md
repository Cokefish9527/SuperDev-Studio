# 质量证据刷新手册

## 目标

将 `redteam`、`coverage.xml` 与 `quality-gate` 三类证据统一刷新，避免仅执行 `super-dev quality --type all` 时丢失安全/性能上下文。

## 一键命令

```bash
python scripts/refresh_quality_evidence.py
```

## 脚本做了什么

1. 在 `backend/` 下执行 `go test ./... -coverprofile=...`
2. 解析 `go tool cover -func=...` 的总覆盖率
3. 输出 `coverage/cobertura-coverage.xml`
4. 调用 `super_dev.reviewers.RedTeamReviewer`
5. 调用 `super_dev.reviewers.QualityGateChecker.check(redteam_report)`
6. 刷新以下产物：
   - `output/superdev-studio-redteam.md`
   - `output/superdev-studio-quality-gate.md`
   - `output/superdev-studio-quality-evidence.md`
   - `output/superdev-studio-quality-evidence.json`

## 为什么不用单独的 `super-dev quality --type all`

当前独立质量命令不会自动读取已有红队结果，因此安全审查和性能审查会退化为默认 `50/100`。

项目内脚本会显式把红队结果传给质量门，确保：

- 安全维度基于真实红队问题计分
- 性能维度基于真实红队问题计分
- 覆盖率维度基于 `coverage/cobertura-coverage.xml` 计分

## 当前已知说明

- 后端总覆盖率以 Go 测试结果为准
- 红队报告仍可能保留 `medium` 级改进项，这不会阻塞质量门通过，但应持续纳入后续重构计划
- 若需要进一步提升覆盖率分数，可继续补充 `backend/internal/store`、`backend/internal/api` 等模块测试
