import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  List,
  Progress,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Timeline,
  Typography,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { apiClient } from '../api/client';
import type { PipelineCompletion, PipelineRun, RunEvent } from '../types';
import { useProjectState } from '../state/project-context';

export default function PipelinePage() {
  const { activeProjectId } = useProjectState();
  const queryClient = useQueryClient();
  const [manualSelectedRunId, setManualSelectedRunId] = useState<string>('');
  const [previewRunId, setPreviewRunId] = useState('');
  const [form] = Form.useForm();
  const contextMode = Form.useWatch('context_mode', form) as 'off' | 'auto' | 'manual' | undefined;
  const fullCycle = Form.useWatch('full_cycle', form) as boolean | undefined;
  const stepByStep = Form.useWatch('step_by_step', form) as boolean | undefined;
  const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

  const runsQuery = useQuery({
    queryKey: ['runs', activeProjectId],
    queryFn: () => apiClient.listRuns(activeProjectId),
    enabled: !!activeProjectId,
    refetchInterval: 2500,
  });
  const runs = runsQuery.data ?? [];
  const selectedRunId = runs.some((item) => item.id === manualSelectedRunId)
    ? manualSelectedRunId
    : runs[0]?.id ?? '';

  const runQuery = useQuery({
    queryKey: ['run', selectedRunId],
    queryFn: () => apiClient.getRun(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: (query) => {
      const status = (query.state.data as PipelineRun | undefined)?.status;
      if (status === 'running' || status === 'queued') {
        return 2000;
      }
      return false;
    },
  });

  const eventsQuery = useQuery({
    queryKey: ['run-events', selectedRunId],
    queryFn: () => apiClient.listRunEvents(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: (query) => {
      const runStatus = runQuery.data?.status;
      if (runStatus === 'running' || runStatus === 'queued') {
        return 1500;
      }
      if (query.state.data && (query.state.data as RunEvent[]).length < 2) {
        return 1500;
      }
      return false;
    },
  });

  const completionQuery = useQuery({
    queryKey: ['run-completion', selectedRunId],
    queryFn: () => apiClient.getRunCompletion(selectedRunId),
    enabled:
      !!selectedRunId &&
      !!runQuery.data &&
      (runQuery.data.status === 'completed' || runQuery.data.status === 'failed'),
  });

  const startMutation = useMutation({
    mutationFn: apiClient.startPipeline,
    onSuccess: (run) => {
      message.success('流水线已启动');
      setManualSelectedRunId(run.id);
      form.resetFields();
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
    },
    onError: (error: Error) => {
      message.error(error.message || '启动失败');
    },
  });

  const retryMutation = useMutation({
    mutationFn: apiClient.retryPipeline,
    onSuccess: (run) => {
      message.success('已创建重试运行');
      setManualSelectedRunId(run.id);
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['run', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-events', run.id] });
    },
    onError: (error: Error) => {
      message.error(error.message || '重试失败');
    },
  });

  const completionData = completionQuery.data as PipelineCompletion | undefined;
  const previewVisible = !!selectedRunId && previewRunId === selectedRunId;

  const runColumns = useMemo(
    () => [
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => (
          <Tag color={status === 'completed' ? 'green' : status === 'failed' ? 'red' : 'blue'}>{status}</Tag>
        ),
      },
      {
        title: '阶段',
        dataIndex: 'stage',
        key: 'stage',
      },
      {
        title: '进度',
        dataIndex: 'progress',
        key: 'progress',
        render: (value: number) => `${value}%`,
      },
      {
        title: '需求',
        dataIndex: 'prompt',
        key: 'prompt',
      },
      {
        title: '开始时间',
        dataIndex: 'created_at',
        key: 'created_at',
        render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
      },
    ],
    [],
  );

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        super-dev 流水线控制台
      </Typography.Title>

      <Card title="启动新运行">
        {!activeProjectId ? (
          <Empty description="请先选择项目" />
        ) : (
          <Form
            layout="vertical"
            form={form}
            onFinish={(
              values: {
                prompt: string;
                simulate: boolean;
                full_cycle?: boolean;
                step_by_step?: boolean;
                iteration_limit?: number;
                project_dir?: string;
                context_mode?: 'off' | 'auto' | 'manual';
                context_query?: string;
                context_token_budget?: number;
                context_max_items?: number;
                context_dynamic?: boolean;
                memory_writeback?: boolean;
              },
            ) => {
              startMutation.mutate({
                project_id: activeProjectId,
                prompt: values.prompt,
                simulate: values.full_cycle || values.step_by_step ? false : (values.simulate ?? true),
                full_cycle: values.full_cycle,
                step_by_step: values.step_by_step,
                iteration_limit: values.iteration_limit,
                project_dir: values.project_dir,
                platform: 'web',
                frontend: 'react',
                backend: 'go',
                context_mode: values.context_mode ?? 'off',
                context_query: values.context_query,
                context_token_budget: values.context_token_budget,
                context_max_items: values.context_max_items,
                context_dynamic: values.context_dynamic,
                memory_writeback: values.memory_writeback,
              });
            }}
            initialValues={{
              simulate: true,
              full_cycle: false,
              step_by_step: false,
              iteration_limit: 3,
              context_mode: 'auto',
              context_token_budget: 1200,
              context_max_items: 8,
              context_dynamic: true,
              memory_writeback: true,
            }}
          >
            <Row gutter={16}>
              <Col xs={24} lg={16}>
                <Form.Item name="prompt" label="需求描述" rules={[{ required: true, message: '请输入需求描述' }]}> 
                  <Input.TextArea rows={3} placeholder="例如：实现一个支持知识库检索和项目任务管理的开发协作平台" />
                </Form.Item>
              </Col>
              <Col xs={24} lg={8}>
                <Form.Item name="project_dir" label="目标项目目录（可选）">
                  <Input placeholder="D:/Work/target-project" />
                </Form.Item>
                <Form.Item name="simulate" label="模拟模式" valuePropName="checked">
                  <Switch
                    checkedChildren="模拟"
                    unCheckedChildren="真实 super-dev"
                    disabled={Boolean(fullCycle || stepByStep)}
                  />
                </Form.Item>
                <Form.Item name="full_cycle" label="一键全流程交付" valuePropName="checked">
                  <Switch checkedChildren="开启" unCheckedChildren="关闭" disabled={Boolean(stepByStep)} />
                </Form.Item>
                <Form.Item name="step_by_step" label="按 super-dev 原生步骤执行" valuePropName="checked">
                  <Switch checkedChildren="开启" unCheckedChildren="关闭" disabled={Boolean(fullCycle)} />
                </Form.Item>
                {fullCycle ? (
                  <>
                    <Form.Item name="iteration_limit" label="开发-单测-修复迭代次数">
                      <InputNumber min={1} max={8} step={1} style={{ width: '100%' }} />
                    </Form.Item>
                    <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
                      开启后将自动执行：设计 → 开发迭代 → 质量测试 → 验收总结 → 上线准备（真实模式）
                    </Typography.Paragraph>
                  </>
                ) : null}
                {stepByStep ? (
                  <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
                    开启后将自动执行：create → task run → quality → preview → deploy（真实模式）
                  </Typography.Paragraph>
                ) : null}
                <Form.Item name="context_mode" label="上下文注入策略">
                  <Select
                    options={[
                      { value: 'off', label: '关闭' },
                      { value: 'auto', label: '自动（按需求召回）' },
                      { value: 'manual', label: '手动（按自定义查询）' },
                    ]}
                  />
                </Form.Item>
                {contextMode === 'manual' && (
                  <Form.Item
                    name="context_query"
                    label="上下文查询"
                    rules={[{ required: true, message: 'manual 模式需要输入查询' }]}
                  >
                    <Input placeholder="例如：订单接口兼容性 + 回滚策略" />
                  </Form.Item>
                )}
                {contextMode !== 'off' && (
                  <>
                    <Form.Item name="context_token_budget" label="上下文 Token 预算">
                      <InputNumber min={200} max={8000} step={100} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="context_max_items" label="最大上下文条目数">
                      <InputNumber min={2} max={20} step={1} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="context_dynamic" label="按阶段动态召回" valuePropName="checked">
                      <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                    </Form.Item>
                  </>
                )}
                <Form.Item name="memory_writeback" label="运行结束回写记忆" valuePropName="checked">
                  <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                </Form.Item>
                <Button type="primary" htmlType="submit" loading={startMutation.isPending}>
                  启动流水线
                </Button>
              </Col>
            </Row>
          </Form>
        )}
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={11}>
          <Card title="运行列表">
            <Table<PipelineRun>
              rowKey="id"
              dataSource={runs}
              columns={runColumns}
              loading={runsQuery.isLoading}
              pagination={{ pageSize: 6 }}
              rowSelection={{
                type: 'radio',
                selectedRowKeys: selectedRunId ? [selectedRunId] : [],
                onChange: (keys) => setManualSelectedRunId(String(keys[0] ?? '')),
              }}
              onRow={(record) => ({ onClick: () => setManualSelectedRunId(record.id) })}
            />
          </Card>
        </Col>

        <Col xs={24} lg={13}>
          <Card title="运行详情">
            {!selectedRunId || !runQuery.data ? (
              <Empty description="请选择一条运行记录" />
            ) : (
              <Space orientation="vertical" style={{ width: '100%' }}>
                <Progress
                  percent={runQuery.data.progress}
                  status={runQuery.data.status === 'failed' ? 'exception' : 'active'}
                />
                <Descriptions column={1} size="small" bordered>
                  <Descriptions.Item label="运行 ID">{runQuery.data.id}</Descriptions.Item>
                  {runQuery.data.retry_of ? (
                    <Descriptions.Item label="重试来源">{runQuery.data.retry_of}</Descriptions.Item>
                  ) : null}
                  <Descriptions.Item label="阶段">{runQuery.data.stage}</Descriptions.Item>
                  <Descriptions.Item label="状态">{runQuery.data.status}</Descriptions.Item>
                  <Descriptions.Item label="需求">{runQuery.data.prompt}</Descriptions.Item>
                </Descriptions>
                {runQuery.data.status === 'failed' ? (
                  <Button
                    danger
                    onClick={() => retryMutation.mutate(runQuery.data.id)}
                    loading={retryMutation.isPending}
                  >
                    重试失败运行
                  </Button>
                ) : null}
                <Card size="small" title="完成清单" loading={completionQuery.isLoading}>
                  {!completionData ? (
                    <Empty description="运行完成后可查看产物清单" />
                  ) : (
                    <Space orientation="vertical" style={{ width: '100%' }} size="middle">
                      <Typography.Text type="secondary">输出目录：{completionData.output_dir}</Typography.Text>
                      <List
                        size="small"
                        dataSource={completionData.checklist}
                        renderItem={(item) => (
                          <List.Item>
                            <Space>
                              <Tag
                                color={
                                  item.status === 'completed'
                                    ? 'green'
                                    : item.status === 'failed'
                                      ? 'red'
                                      : item.status === 'in_progress'
                                        ? 'blue'
                                        : 'orange'
                                }
                              >
                                {item.status}
                              </Tag>
                              <Typography.Text>{item.title}</Typography.Text>
                              {item.note ? <Typography.Text type="secondary">{item.note}</Typography.Text> : null}
                            </Space>
                          </List.Item>
                        )}
                      />
                      <Typography.Text strong>产物列表</Typography.Text>
                      <List
                        size="small"
                        dataSource={completionData.artifacts}
                        renderItem={(artifact) => (
                          <List.Item>
                            <Space orientation="vertical" size={0}>
                              <Typography.Text>{artifact.name}</Typography.Text>
                              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                                {artifact.path}
                              </Typography.Text>
                            </Space>
                          </List.Item>
                        )}
                      />
                      {completionData.preview_url ? (
                        <>
                          <Button
                            onClick={() =>
                              setPreviewRunId((current) => (current === selectedRunId ? '' : selectedRunId))
                            }
                          >
                            {previewVisible ? '隐藏预览' : '预览页面'}
                          </Button>
                          {previewVisible ? (
                            <iframe
                              title="pipeline-preview"
                              src={`${apiBase}${completionData.preview_url}`}
                              style={{ width: '100%', height: 460, border: '1px solid #f0f0f0' }}
                            />
                          ) : null}
                        </>
                      ) : null}
                    </Space>
                  )}
                </Card>
                <Timeline
                  items={(eventsQuery.data ?? []).map((event) => ({
                    color:
                      event.status === 'failed' ? 'red' : event.status === 'completed' ? 'green' : 'blue',
                    children: (
                      <Space orientation="vertical" size={2}>
                        <Typography.Text strong>
                          [{event.stage}] {event.status}
                        </Typography.Text>
                        <Typography.Text>{event.message}</Typography.Text>
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          {dayjs(event.created_at).format('YYYY-MM-DD HH:mm:ss')}
                        </Typography.Text>
                      </Space>
                    ),
                  }))}
                />
              </Space>
            )}
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
