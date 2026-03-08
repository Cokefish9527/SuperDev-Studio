import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Card, Empty, Form, Input, InputNumber, Select, Space, Switch, Typography, message } from 'antd';
import { useEffect, useMemo } from 'react';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';

function buildOptions(values: Array<string | undefined>) {
  return Array.from(new Set(values.map((item) => (item ?? '').trim()).filter(Boolean))).map((value) => ({ value, label: value }));
}

export default function ProjectSettingsPage() {
  const { activeProjectId } = useProjectState();
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const projectQuery = useQuery({
    queryKey: ['project', activeProjectId],
    queryFn: () => apiClient.getProject(activeProjectId),
    enabled: !!activeProjectId,
  });

  const bundleQuery = useQuery({
    queryKey: ['project-agent-bundle', activeProjectId],
    queryFn: () => apiClient.getProjectAgentBundle(activeProjectId),
    enabled: !!activeProjectId,
    retry: false,
  });

  useEffect(() => {
    if (projectQuery.data) {
      form.setFieldsValue(projectQuery.data);
    }
  }, [form, projectQuery.data]);

  const agentOptions = useMemo(
    () => buildOptions([...(bundleQuery.data?.agents ?? []).map((item) => item.name), projectQuery.data?.default_agent_name]),
    [bundleQuery.data, projectQuery.data],
  );
  const modeOptions = useMemo(
    () => buildOptions([...(bundleQuery.data?.modes ?? []).map((item) => item.name), projectQuery.data?.default_agent_mode]),
    [bundleQuery.data, projectQuery.data],
  );

  const updateMutation = useMutation({
    mutationFn: (values: Record<string, unknown>) => apiClient.updateProject(activeProjectId, values),
    onSuccess: () => {
      message.success('项目设置已更新');
      void queryClient.invalidateQueries({ queryKey: ['projects'] });
      void queryClient.invalidateQueries({ queryKey: ['project', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['project-agent-bundle', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message),
  });

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        项目设置
      </Typography.Title>

      <Card title="执行配置来源">
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          这里定义项目级默认技术栈、上下文策略与 Agent 策略；“交付运行”页会优先使用这些配置，避免每次手动重复填写。
        </Typography.Paragraph>
      </Card>

      <Card title="默认配置">
        {!activeProjectId ? (
          <Empty description="请先选择项目" />
        ) : (
          <Form form={form} layout="vertical" onFinish={(values) => updateMutation.mutate(values)}>
            <Form.Item name="name" label="项目名" rules={[{ required: true, message: '请输入项目名' }]}>
              <Input />
            </Form.Item>
            <Form.Item name="description" label="项目描述">
              <Input.TextArea rows={3} />
            </Form.Item>
            <Form.Item name="repo_path" label="仓库路径">
              <Input placeholder="D:/Work/agent-demo/SuperDev-Studio" />
            </Form.Item>
            <Form.Item name="status" label="项目状态">
              <Select options={[{ value: 'active' }, { value: 'paused' }, { value: 'archived' }]} />
            </Form.Item>
            <Space style={{ width: '100%' }} size="large" align="start">
              <Form.Item name="default_platform" label="默认平台">
                <Select options={[{ value: 'web' }, { value: 'mobile' }, { value: 'desktop' }]} style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="default_frontend" label="默认前端">
                <Select options={[{ value: 'react' }, { value: 'vue' }, { value: 'angular' }, { value: 'svelte' }]} style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="default_backend" label="默认后端">
                <Select options={[{ value: 'go' }, { value: 'node' }, { value: 'python' }, { value: 'java' }]} style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="default_domain" label="默认领域">
                <Input style={{ width: 160 }} placeholder="saas / content" />
              </Form.Item>
            </Space>
            <Card size="small" title="默认 Agent 策略" style={{ marginBottom: 16, borderRadius: 16 }}>
              <Space style={{ width: '100%' }} size="large" align="start">
                <Form.Item name="default_agent_name" label="默认 Agent" style={{ minWidth: 220 }}>
                  <Select options={agentOptions} placeholder="delivery-agent" />
                </Form.Item>
                <Form.Item name="default_agent_mode" label="默认 Agent Mode" style={{ minWidth: 220 }}>
                  <Select options={modeOptions} placeholder="step_by_step" />
                </Form.Item>
              </Space>
              <Typography.Text type="secondary">
                选项来自项目目录下 `.studio-agent` 配置；若未定义，则回退到内置默认 Bundle。
              </Typography.Text>
            </Card>
            <Space style={{ width: '100%' }} size="large" align="start">
              <Form.Item name="default_context_mode" label="默认上下文模式">
                <Select options={[{ value: 'off' }, { value: 'auto' }, { value: 'manual' }]} style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="default_context_token_budget" label="默认 Token 预算">
                <InputNumber min={200} max={8000} step={100} style={{ width: 180 }} />
              </Form.Item>
              <Form.Item name="default_context_max_items" label="默认条目数">
                <InputNumber min={2} max={20} step={1} style={{ width: 160 }} />
              </Form.Item>
            </Space>
            <Space size="large">
              <Form.Item name="default_context_dynamic" label="按阶段动态召回" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
              <Form.Item name="default_memory_writeback" label="运行结束回写记忆" valuePropName="checked">
                <Switch checkedChildren="开启" unCheckedChildren="关闭" />
              </Form.Item>
            </Space>
            <div>
              <Button type="primary" htmlType="submit" loading={updateMutation.isPending}>
                保存项目设置
              </Button>
            </div>
          </Form>
        )}
      </Card>
    </Space>
  );
}