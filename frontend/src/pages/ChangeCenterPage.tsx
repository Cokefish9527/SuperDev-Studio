import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Card, Empty, Form, Input, Modal, Space, Table, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import { useMemo, useState } from 'react';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';
import type { ChangeBatch } from '../types';

export default function ChangeCenterPage() {
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { activeProjectId, activeChangeBatchId, setActiveChangeBatchId } = useProjectState();

  const batchesQuery = useQuery({
    queryKey: ['change-batches', activeProjectId],
    queryFn: () => apiClient.listChangeBatches(activeProjectId),
    enabled: !!activeProjectId,
  });

  const createMutation = useMutation({
    mutationFn: (payload: { title: string; goal?: string; mode?: string }) =>
      apiClient.createChangeBatch(activeProjectId, payload),
    onSuccess: (batch) => {
      setOpen(false);
      form.resetFields();
      setActiveChangeBatchId(batch.id);
      message.success('变更批次已创建');
      void queryClient.invalidateQueries({ queryKey: ['change-batches', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message),
  });

  const columns = useMemo(
    () => [
      {
        title: '批次标题',
        dataIndex: 'title',
        key: 'title',
        render: (value: string, record: ChangeBatch) => (
          <Button type={record.id === activeChangeBatchId ? 'primary' : 'link'} onClick={() => setActiveChangeBatchId(record.id)}>
            {value}
          </Button>
        ),
      },
      {
        title: '模式',
        dataIndex: 'mode',
        key: 'mode',
        render: (value: string) => <Tag color={value === 'full_cycle' ? 'purple' : 'blue'}>{value}</Tag>,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (value: string) => <Tag color={value === 'completed' ? 'green' : value === 'failed' ? 'red' : 'gold'}>{value}</Tag>,
      },
      {
        title: '外部 change_id',
        dataIndex: 'external_change_id',
        key: 'external_change_id',
        render: (value?: string) => value || '-',
      },
      {
        title: '最近运行',
        dataIndex: 'latest_run_id',
        key: 'latest_run_id',
        render: (value?: string) => value || '-',
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        key: 'updated_at',
        render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
      },
    ],
    [activeChangeBatchId, setActiveChangeBatchId],
  );

  const selectedBatch = (batchesQuery.data ?? []).find((item) => item.id === activeChangeBatchId);

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
          变更中心
        </Typography.Title>
        <Button type="primary" disabled={!activeProjectId} onClick={() => setOpen(true)}>
          新建变更批次
        </Button>
      </Space>

      <Card title="改版思路">
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          以 change batch 为交付单元：先明确目标与范围，再在“交付运行”页基于当前批次启动 super-dev 流程，运行结果会自动回写外部
          change_id 与最近运行链路。
        </Typography.Paragraph>
      </Card>

      <Card title="批次列表">
        {!activeProjectId ? (
          <Empty description="请先选择项目" />
        ) : (
          <Table<ChangeBatch>
            rowKey="id"
            columns={columns}
            dataSource={batchesQuery.data ?? []}
            loading={batchesQuery.isLoading}
            pagination={{ pageSize: 6 }}
          />
        )}
      </Card>

      <Card title="当前选中批次">
        {!selectedBatch ? (
          <Empty description="请选择或创建一个变更批次" />
        ) : (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Text strong>{selectedBatch.title}</Typography.Text>
            <Typography.Text type="secondary">目标：{selectedBatch.goal || '未填写'}</Typography.Text>
            <Typography.Text type="secondary">
              交付建议：前往“交付运行”页，以当前批次为上下文启动本轮 super-dev 执行。
            </Typography.Text>
          </Space>
        )}
      </Card>

      <Modal
        open={open}
        title="新建变更批次"
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{ mode: 'step_by_step' }}
          onFinish={(values) => createMutation.mutate(values)}
        >
          <Form.Item name="title" label="批次标题" rules={[{ required: true, message: '请输入批次标题' }]}>
            <Input placeholder="例如：工作台 IA 与数据模型改版" />
          </Form.Item>
          <Form.Item name="goal" label="目标说明">
            <Input.TextArea rows={4} placeholder="本批次要解决的问题、改动范围与验收预期" />
          </Form.Item>
          <Form.Item name="mode" label="执行模式">
            <Input placeholder="step_by_step / full_cycle" />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
