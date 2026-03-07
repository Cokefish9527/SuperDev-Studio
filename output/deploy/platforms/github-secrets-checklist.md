# Deploy Remediation Checklist

- Platform: `github`
- only_missing: `true`

## Environment Variables

| Name | Status | Description | Template |
|:---|:---:|:---|:---|
| `DOCKER_USERNAME` | `missing` | Docker 镜像仓库用户名 | `DOCKER_USERNAME="<value>"` |
| `DOCKER_PASSWORD` | `missing` | Docker 镜像仓库密码/Token | `DOCKER_PASSWORD="<value>"` |
| `KUBE_CONFIG_DEV` | `missing` | 开发环境 Kubernetes kubeconfig | `KUBE_CONFIG_DEV="<value>"` |
| `KUBE_CONFIG_PROD` | `missing` | 生产环境 Kubernetes kubeconfig | `KUBE_CONFIG_PROD="<value>"` |

## Platform Guidance

- 在 GitHub Settings > Secrets and variables > Actions 配置变量。
- 按 dev/prod 环境拆分敏感变量。

## Manual Requirements

- No manual requirements.
