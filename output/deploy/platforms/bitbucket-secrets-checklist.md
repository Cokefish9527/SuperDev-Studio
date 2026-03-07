# Deploy Remediation Checklist

- Platform: `bitbucket`
- only_missing: `true`

## Environment Variables

| Name | Status | Description | Template |
|:---|:---:|:---|:---|
| `REGISTRY_URL` | `missing` | 镜像仓库地址 | `REGISTRY_URL="<value>"` |
| `KUBE_CONFIG_DEV` | `missing` | 开发环境 Kubernetes kubeconfig | `KUBE_CONFIG_DEV="<value>"` |
| `KUBE_CONFIG_PROD` | `missing` | 生产环境 Kubernetes kubeconfig | `KUBE_CONFIG_PROD="<value>"` |

## Platform Guidance

- 在 Bitbucket Repository variables 中配置密钥。

## Manual Requirements

- No manual requirements.
