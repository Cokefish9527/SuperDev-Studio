import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  Progress,
  Row,
  Skeleton,
  Space,
  Statistic,
  Tag,
  Typography,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';
import type { RequirementDocVersion, RequirementSessionBundle } from '../types';

function DocPreview({ title, content }: { title: string; content?: string }) {
  if (!content) return null;
  return (
    <Card size="small" title={title} style={{ borderRadius: 14 }}>
      <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', margin: 0 }}>{content}</pre>
    </Card>
  );
}

function pickLatestDoc(docVersions: RequirementDocVersion[], type: string) {
  return docVersions
    .filter((item) => item.type === type)
    .sort((left, right) => right.version - left.version)[0];
}

function buildPreviewHref(apiBase: string, previewUrl?: string) {
  if (!previewUrl) return '';
  if (/^https?:\/\//.test(previewUrl)) return previewUrl;
  return `${apiBase}${previewUrl}`;
}

function runStatusColor(status?: string) {
  switch (status) {
    case 'completed':
      return 'green';
    case 'failed':
      return 'red';
    case 'awaiting_human':
      return 'orange';
    default:
      return 'blue';
  }
}

export default function SimpleDeliveryPage() {
  const navigate = useNavigate();
  const { activeProjectId, setActiveChangeBatchId } = useProjectState();
  const [form] = Form.useForm<{ title?: string; raw_input: string }>();
  const [sessionBundle, setSessionBundle] = useState<RequirementSessionBundle | null>(null);
  const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

  const projectQuery = useQuery({
    queryKey: ['project', activeProjectId],
    queryFn: () => apiClient.getProject(activeProjectId),
    enabled: !!activeProjectId,
  });

  const latest = sessionBundle?.session;
  const latestRunId = latest?.latest_run_id || sessionBundle?.run?.id || '';

  const runQuery = useQuery({
    queryKey: ['simple-run', latestRunId],
    queryFn: () => apiClient.getRun(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 2500 : false,
  });

  const completionQuery = useQuery({
    queryKey: ['simple-run-completion', latestRunId],
    queryFn: () => apiClient.getRunCompletion(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 5000 : false,
  });

  const fetchSession = useMutation({
    mutationFn: async (sessionId: string) => {
      if (!activeProjectId) throw new Error('缺少工作区');
      return apiClient.getRequirementSession(activeProjectId, sessionId);
    },
    onSuccess: (data) => setSessionBundle((previous) => ({ ...previous, ...data })),
    onError: (err: Error) => message.error(err.message),
  });

  const createSession = useMutation({
    mutationFn: async (values: { title?: string; raw_input: string }) => {
      if (!activeProjectId) throw new Error('缺少工作区');
      return apiClient.createRequirementSession(activeProjectId, values);
    },
    onSuccess: (data) => {
      message.success('已生成需求草案，等待确认');
      setSessionBundle(data);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const reviseSession = useMutation({
    mutationFn: async (payload: { title?: string; raw_input?: string }) => {
      const sid = sessionBundle?.session.id;
      if (!activeProjectId || !sid) throw new Error('缺少会话');
      return apiClient.reviseRequirementSession(activeProjectId, sid, payload);
    },
    onSuccess: (data) => {
      message.success('已更新需求草案，请重新确认');
      setSessionBundle(data);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const confirmSession = useMutation({
    mutationFn: async (payload: { note?: string }) => {
      const sid = sessionBundle?.session.id;
      if (!activeProjectId || !sid) throw new Error('缺少会话');
      return apiClient.confirmRequirementSession(activeProjectId, sid, payload);
    },
    onSuccess: (data) => {
      setSessionBundle(data);
      if (data.change_batch?.id) {
        setActiveChangeBatchId(data.change_batch.id);
      }
      if (data.delivery_error) {
        message.warning('需求已确认，但自动交付启动失败，请转到交付运行页继续处理');
        return;
      }
      message.success('需求已确认，已自动启动交付流程');
    },
    onError: (err: Error) => message.error(err.message),
  });

  const docVersions = sessionBundle?.doc_versions ?? [];
  const summaryDoc = useMemo(() => pickLatestDoc(docVersions, 'summary'), [docVersions]);
  const prdDoc = useMemo(() => pickLatestDoc(docVersions, 'prd'), [docVersions]);
  const planDoc = useMemo(() => pickLatestDoc(docVersions, 'plan'), [docVersions]);
  const riskDoc = useMemo(() => pickLatestDoc(docVersions, 'risks'), [docVersions]);
  const run = runQuery.data ?? sessionBundle?.run;
  const previewHref = buildPreviewHref(apiBase, completionQuery.data?.preview_url);
  const autoModeLabel =
    projectQuery.data?.default_agent_mode === 'full_cycle' ? '全流程自动交付' : '逐步推进交付';

  const statusTag = (status?: string) => {
    switch (status) {
      case 'confirmed':
        return <Tag color="green">已确认</Tag>;
      case 'awaiting_confirm':
        return <Tag color="orange">待确认</Tag>;
      default:
        return <Tag>草稿</Tag>;
    }
  };

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        简单交付入口
      </Typography.Title>

      {!activeProjectId ? (
        <Alert type="warning" showIcon message="请先在左侧选择工作区" />
      ) : (
        <Card title="需求输入" style={{ borderRadius: 16 }}>
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Alert
              type="info"
              showIcon
              message="默认执行方式"
              description={`确认后将按当前项目默认模式自动进入${autoModeLabel}。高级参数请在项目设置中统一维护。`}
            />
            <Form
              layout="vertical"
              form={form}
              initialValues={{ raw_input: '' }}
              onFinish={(values) => createSession.mutate(values)}
            >
              <Form.Item label="需求标题（可选）" name="title">
                <Input placeholder="如：时间线 + 知识图谱记事本" allowClear />
              </Form.Item>
              <Form.Item
                label="需求描述"
                name="raw_input"
                rules={[{ required: true, message: '请输入需求描述' }]}
              >
                <Input.TextArea rows={4} placeholder="一句话或一段话描述需求" allowClear />
              </Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={createSession.isPending}
                  disabled={!activeProjectId}
                >
                  生成需求草案
                </Button>
                {latest ? (
                  <Button onClick={() => fetchSession.mutate(latest.id)} loading={fetchSession.isPending}>
                    刷新会话
                  </Button>
                ) : null}
              </Space>
            </Form>
          </Space>
        </Card>
      )}

      {createSession.isPending || fetchSession.isPending ? <Skeleton active /> : null}

      {latest ? (
        <Card
          title={
            <Space>
              <Typography.Text strong>{latest.title}</Typography.Text>
              {statusTag(latest.status)}
              <Tag>{latest.id.slice(0, 8)}</Tag>
            </Space>
          }
          extra={
            <Space>
              <Statistic
                title="创建时间"
                value={dayjs(latest.created_at).format('MM-DD HH:mm')}
                valueStyle={{ fontSize: 14 }}
              />
              <Statistic
                title="更新"
                value={dayjs(latest.updated_at).format('MM-DD HH:mm')}
                valueStyle={{ fontSize: 14 }}
              />
            </Space>
          }
          style={{ borderRadius: 16 }}
        >
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <DocPreview title="需求摘要" content={summaryDoc?.content || latest.latest_summary} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title="风险与开放问题" content={riskDoc?.content || latest.latest_risks} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title="PRD 草案" content={prdDoc?.content || latest.latest_prd} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title="开发计划草案" content={planDoc?.content || latest.latest_plan} />
            </Col>
          </Row>

          <Divider />
          <Space wrap>
            <Button
              type="primary"
              onClick={() => confirmSession.mutate({})}
              loading={confirmSession.isPending}
              disabled={latest.status === 'confirmed'}
            >
              确认并启动交付
            </Button>
            <Button
              onClick={() =>
                reviseSession.mutate({
                  title: latest.title,
                  raw_input: latest.raw_input,
                })
              }
              loading={reviseSession.isPending}
            >
              重新生成草案
            </Button>
          </Space>

          {confirmSession.isPending || reviseSession.isPending ? (
            <Skeleton active style={{ marginTop: 12 }} />
          ) : null}
          {sessionBundle?.confirmation ? (
            <Alert
              type="success"
              showIcon
              style={{ marginTop: 12 }}
              message="需求已确认"
              description={dayjs(sessionBundle.confirmation.created_at).format('YYYY-MM-DD HH:mm:ss')}
            />
          ) : null}
          {sessionBundle?.delivery_error ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 12 }}
              message="自动交付未成功启动"
              description={sessionBundle.delivery_error}
            />
          ) : null}
        </Card>
      ) : null}

      {latestRunId ? (
        <Card
          title="交付结果"
          style={{ borderRadius: 16 }}
          extra={
            <Space>
              <Button onClick={() => navigate('/pipeline')}>打开交付看板</Button>
              {previewHref ? (
                <Button type="primary" onClick={() => window.open(previewHref, '_blank', 'noopener,noreferrer')}>
                  打开预览
                </Button>
              ) : null}
            </Space>
          }
        >
          {!run ? (
            <Skeleton active />
          ) : (
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <Space wrap>
                <Tag color={runStatusColor(run.status)}>{run.status}</Tag>
                <Tag>{run.stage}</Tag>
                {run.full_cycle ? <Tag color="cyan">full-cycle</Tag> : null}
                {run.step_by_step ? <Tag color="blue">step-by-step</Tag> : null}
                {latest?.latest_change_batch_id ? <Tag>{latest.latest_change_batch_id.slice(0, 8)}</Tag> : null}
              </Space>
              <Typography.Text type="secondary">{run.prompt}</Typography.Text>
              <Progress percent={run.progress} strokeColor={{ from: '#0ea5e9', to: '#7c3aed' }} />
              <Row gutter={[16, 16]}>
                <Col xs={24} md={8}>
                  <Statistic title="运行 ID" value={run.id.slice(0, 8)} valueStyle={{ fontSize: 16 }} />
                </Col>
                <Col xs={24} md={8}>
                  <Statistic title="更新时间" value={dayjs(run.updated_at).format('MM-DD HH:mm:ss')} valueStyle={{ fontSize: 16 }} />
                </Col>
                <Col xs={24} md={8}>
                  <Statistic
                    title="预览状态"
                    value={completionQuery.data?.preview_url ? '已生成' : '生成中'}
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
              </Row>
              {completionQuery.data?.preview_url ? (
                <Alert
                  type="success"
                  showIcon
                  message="已生成可预览产物"
                  description="你可以直接打开预览，也可以进入交付看板查看阶段详情、Agent 评估和剩余问题。"
                />
              ) : (
                <Alert
                  type="info"
                  showIcon
                  message="正在等待交付产物"
                  description="系统会持续轮询运行状态；如果需要更完整的阶段视图，请打开交付看板。"
                />
              )}
            </Space>
          )}
        </Card>
      ) : null}

      <Card title="如何使用" size="small" style={{ borderRadius: 12 }}>
        <Space direction="vertical">
          <Typography.Text>1) 输入一句需求，系统先生成摘要、PRD、计划和风险。</Typography.Text>
          <Typography.Text>2) 你只需确认理解是否正确；高风险问题会在后续流程中单独提示。</Typography.Text>
          <Typography.Text>3) 确认后系统自动进入标准 super-dev 交付流程，并在本页持续展示运行结果。</Typography.Text>
        </Space>
      </Card>
    </Space>
  );
}
