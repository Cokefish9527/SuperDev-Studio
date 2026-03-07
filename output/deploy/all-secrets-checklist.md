# Deploy Remediation Checklist

- Platform: `all`
- only_missing: `true`

## Environment Variables

| Name | Status | Description | Template |
|:---|:---:|:---|:---|
| `DOCKER_USERNAME` | `missing` | Docker 镜像仓库用户名 | `DOCKER_USERNAME="<value>"` |
| `DOCKER_PASSWORD` | `missing` | Docker 镜像仓库密码/Token | `DOCKER_PASSWORD="<value>"` |
| `KUBE_CONFIG_DEV` | `missing` | 开发环境 Kubernetes kubeconfig | `KUBE_CONFIG_DEV="<value>"` |
| `KUBE_CONFIG_PROD` | `missing` | 生产环境 Kubernetes kubeconfig | `KUBE_CONFIG_PROD="<value>"` |
| `CI_REGISTRY_USER` | `missing` | GitLab Registry 用户名 | `CI_REGISTRY_USER="<value>"` |
| `CI_REGISTRY_PASSWORD` | `missing` | GitLab Registry 密码/Token | `CI_REGISTRY_PASSWORD="<value>"` |
| `KUBE_CONTEXT_DEV` | `missing` | 开发环境 K8s 上下文 | `KUBE_CONTEXT_DEV="<value>"` |
| `KUBE_CONTEXT_PROD` | `missing` | 生产环境 K8s 上下文 | `KUBE_CONTEXT_PROD="<value>"` |
| `AZURE_ACR_SERVICE_CONNECTION` | `missing` | Azure ACR 服务连接标识 | `AZURE_ACR_SERVICE_CONNECTION="<value>"` |
| `AZURE_DEV_K8S_CONNECTION` | `missing` | 开发环境 AKS 服务连接标识 | `AZURE_DEV_K8S_CONNECTION="<value>"` |
| `AZURE_PROD_K8S_CONNECTION` | `missing` | 生产环境 AKS 服务连接标识 | `AZURE_PROD_K8S_CONNECTION="<value>"` |
| `REGISTRY_URL` | `missing` | 镜像仓库地址 | `REGISTRY_URL="<value>"` |

## Platform Guidance

- 在 GitHub Settings > Secrets and variables > Actions 配置变量。
- 按 dev/prod 环境拆分敏感变量。
- 在 GitLab Settings > CI/CD > Variables 中配置变量并启用 Masked。
- 在 Jenkins Credentials 中创建与流水线一致的凭据 ID。
- 在 Azure DevOps 配置 Service Connection 和 Variable Group。
- 在 Bitbucket Repository variables 中配置密钥。

## Manual Requirements

- Jenkins Credentials: docker-credentials
- Jenkins Credentials: kubeconfig-dev
- Jenkins Credentials: kubeconfig-prod
