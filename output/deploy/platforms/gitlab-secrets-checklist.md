# Deploy Remediation Checklist

- Platform: `gitlab`
- only_missing: `true`

## Environment Variables

| Name | Status | Description | Template |
|:---|:---:|:---|:---|
| `CI_REGISTRY_USER` | `missing` | GitLab Registry 用户名 | `CI_REGISTRY_USER="<value>"` |
| `CI_REGISTRY_PASSWORD` | `missing` | GitLab Registry 密码/Token | `CI_REGISTRY_PASSWORD="<value>"` |
| `KUBE_CONTEXT_DEV` | `missing` | 开发环境 K8s 上下文 | `KUBE_CONTEXT_DEV="<value>"` |
| `KUBE_CONTEXT_PROD` | `missing` | 生产环境 K8s 上下文 | `KUBE_CONTEXT_PROD="<value>"` |

## Platform Guidance

- 在 GitLab Settings > CI/CD > Variables 中配置变量并启用 Masked。

## Manual Requirements

- No manual requirements.
