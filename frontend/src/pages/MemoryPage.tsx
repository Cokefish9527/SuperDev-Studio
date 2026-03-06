import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Card, Empty, Form, Input, InputNumber, Space, Table, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { apiClient } from '../api/client';
import type { Memory } from '../types';
import { useProjectState } from '../state/project-context';

export default function MemoryPage() {
  const { activeProjectId } = useProjectState();
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const memoriesQuery = useQuery({
    queryKey: ['memories', activeProjectId],
    queryFn: () => apiClient.listMemories(activeProjectId),
    enabled: !!activeProjectId,
  });

  const createMutation = useMutation({
    mutationFn: (payload: Partial<Memory>) => apiClient.createMemory(activeProjectId, payload),
    onSuccess: () => {
      form.resetFields();
      message.success('记忆已写入');
      void queryClient.invalidateQueries({ queryKey: ['memories', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message),
  });

  const columns = [
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => <Tag color="cyan">{role}</Tag>,
    },
    {
      title: '内容',
      dataIndex: 'content',
      key: 'content',
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      render: (tags: string[]) => (
        <Space wrap>
          {tags?.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '重要度',
      dataIndex: 'importance',
      key: 'importance',
    },
    {
      title: '写入时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        记忆模块
      </Typography.Title>
      <Card title="新增记忆">
        {!activeProjectId ? (
          <Empty description="请先选择项目" />
        ) : (
          <Form
            form={form}
            layout="vertical"
            initialValues={{ role: 'note', importance: 0.7 }}
            onFinish={(values: { role: string; content: string; tags?: string; importance: number }) => {
              createMutation.mutate({
                role: values.role,
                content: values.content,
                importance: values.importance,
                tags: (values.tags || '')
                  .split(',')
                  .map((item) => item.trim())
                  .filter(Boolean),
              });
            }}
          >
            <Form.Item name="role" label="角色" rules={[{ required: true }]}> 
              <Input placeholder="user / assistant / note" />
            </Form.Item>
            <Form.Item name="content" label="内容" rules={[{ required: true }]}> 
              <Input.TextArea rows={4} />
            </Form.Item>
            <Form.Item name="tags" label="标签（逗号分隔）">
              <Input placeholder="架构, 风险, 接口" />
            </Form.Item>
            <Form.Item name="importance" label="重要度（0-1）">
              <InputNumber min={0} max={1} step={0.1} style={{ width: 180 }} />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={createMutation.isPending}>
              写入记忆
            </Button>
          </Form>
        )}
      </Card>

      <Card title="记忆列表">
        <Table<Memory>
          rowKey="id"
          columns={columns}
          dataSource={memoriesQuery.data ?? []}
          loading={memoriesQuery.isLoading}
        />
      </Card>
    </Space>
  );
}
