# Deploy Remediation Checklist

- Platform: `azure`
- only_missing: `true`

## Environment Variables

| Name | Status | Description | Template |
|:---|:---:|:---|:---|
| `AZURE_ACR_SERVICE_CONNECTION` | `missing` | Azure ACR 服务连接标识 | `AZURE_ACR_SERVICE_CONNECTION="<value>"` |
| `AZURE_DEV_K8S_CONNECTION` | `missing` | 开发环境 AKS 服务连接标识 | `AZURE_DEV_K8S_CONNECTION="<value>"` |
| `AZURE_PROD_K8S_CONNECTION` | `missing` | 生产环境 AKS 服务连接标识 | `AZURE_PROD_K8S_CONNECTION="<value>"` |

## Platform Guidance

- 在 Azure DevOps 配置 Service Connection 和 Variable Group。

## Manual Requirements

- No manual requirements.
