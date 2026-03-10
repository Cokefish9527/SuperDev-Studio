import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  FloatButton,
  Form,
  Input,
  InputNumber,
  Progress,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { apiClient } from '../api/client';
import PipelineArtifactPreviewPanel from '../components/pipeline/PipelineArtifactPreviewPanel';
import DeliveryHandoffCard from '../components/pipeline/DeliveryHandoffCard';
import PipelineChecklistCard from '../components/pipeline/PipelineChecklistCard';
import PipelineStageBoardCard from '../components/pipeline/PipelineStageBoardCard';
import PipelineTimelineCard from '../components/pipeline/PipelineTimelineCard';
import PipelineRunDetailsModal from '../components/pipeline/PipelineRunDetailsModal';
import type {
  AgentEvaluation,
  AgentEvidence,
  AgentStep,
  AgentToolCall,
  ApprovalGate,
  ChangeBatch,
  PipelineArtifact,
  PipelineAutoAdvanceResult,
  PipelineCompletion,
  PipelineRunAgent,
  PipelineRun,
  PipelineStage,
  PreviewSession,
  Project,
  ResidualItem,
} from '../types';
import { useProjectState } from '../state/project-context';

type PipelineFormValues = {
  prompt: string;
  llm_enhanced_loop?: boolean;
  multimodal_assets_text?: string;
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
  agent_name?: string;
  agent_mode?: string;
};

const STAGE_FALLBACK: PipelineStage[] = [
  { key: 'idea', title: '构思', status: 'pending', artifacts: [] },
  { key: 'design', title: '设计', status: 'pending', artifacts: [] },
  { key: 'superdev', title: 'super-dev', status: 'pending', artifacts: [] },
  { key: 'output', title: '产出', status: 'pending', artifacts: [] },
  { key: 'rethink', title: '再构思', status: 'pending', artifacts: [] },
];

const FULL_CYCLE_RELEASE_APPROVAL_STAGE = 'lifecycle-release-approval';
const FULL_CYCLE_DEPLOY_TOOL = 'run_superdev_deploy';

export default function PipelinePage() {
  const { activeProjectId, activeChangeBatchId } = useProjectState();
  const queryClient = useQueryClient();
  const [manualSelectedRunId, setManualSelectedRunId] = useState('');
  const [selectedArtifactPath, setSelectedArtifactPath] = useState('');
  const [detailsModalOpen, setDetailsModalOpen] = useState(false);
  const [detailsTabKey, setDetailsTabKey] = useState('overview');
  const [form] = Form.useForm<PipelineFormValues>();
  const contextMode = Form.useWatch('context_mode', form) as PipelineFormValues['context_mode'];
  const fullCycle = Form.useWatch('full_cycle', form) as boolean | undefined;
  const stepByStep = Form.useWatch('step_by_step', form) as boolean | undefined;
  const llmEnhancedLoop = Form.useWatch('llm_enhanced_loop', form) as boolean | undefined;
  const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

  const runsQuery = useQuery({
    queryKey: ['runs', activeProjectId],
    queryFn: () => apiClient.listRuns(activeProjectId),
    enabled: !!activeProjectId,
    refetchInterval: 2500,
  });
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
  const changeBatchesQuery = useQuery({
    queryKey: ['change-batches', activeProjectId],
    queryFn: () => apiClient.listChangeBatches(activeProjectId),
    enabled: !!activeProjectId,
  });

  const runs = runsQuery.data ?? [];
  const selectedBatch = (changeBatchesQuery.data ?? []).find((item: ChangeBatch) => item.id === activeChangeBatchId);
  const selectedRunId = runs.some((item) => item.id === manualSelectedRunId)
    ? manualSelectedRunId
    : runs[0]?.id ?? '';
  const bundleAgentNames = useMemo(
    () => (bundleQuery.data?.agents ?? []).map((item) => item.name.trim()).filter(Boolean),
    [bundleQuery.data],
  );
  const bundleModeNames = useMemo(
    () => (bundleQuery.data?.modes ?? []).map((item) => item.name.trim()).filter(Boolean),
    [bundleQuery.data],
  );
  const projectDefaultAgentName = (projectQuery.data?.default_agent_name ?? '').trim();
  const projectDefaultAgentMode = (projectQuery.data?.default_agent_mode ?? '').trim();
  const bundleFallbackAgentName = firstNonEmptyValue(bundleAgentNames[0], 'delivery-agent');
  const bundleFallbackModeName = firstNonEmptyValue(bundleModeNames[0], 'step_by_step');
  const resolvedProjectAgentName = resolveBundleSelection(projectDefaultAgentName, bundleAgentNames, bundleFallbackAgentName);
  const resolvedProjectAgentMode = resolveBundleSelection(projectDefaultAgentMode, bundleModeNames, bundleFallbackModeName);
  const staleProjectAgentName = Boolean(bundleAgentNames.length && projectDefaultAgentName && !hasNamedOption(bundleAgentNames, projectDefaultAgentName));
  const staleProjectAgentMode = Boolean(bundleModeNames.length && projectDefaultAgentMode && !hasNamedOption(bundleModeNames, projectDefaultAgentMode));
  const agentFallbackNotice = useMemo(() => {
    if (!staleProjectAgentName && !staleProjectAgentMode) {
      return undefined;
    }
    const details: string[] = [];
    if (staleProjectAgentName) {
      details.push(`默认 Agent 已从 ${projectDefaultAgentName} 回退为 ${resolvedProjectAgentName}`);
    }
    if (staleProjectAgentMode) {
      details.push(`默认 Mode 已从 ${projectDefaultAgentMode} 回退为 ${resolvedProjectAgentMode}`);
    }
    return details.join('；');
  }, [projectDefaultAgentName, projectDefaultAgentMode, resolvedProjectAgentName, resolvedProjectAgentMode, staleProjectAgentName, staleProjectAgentMode]);
  const agentOptions = useMemo(
    () => buildSelectOptions([...(bundleQuery.data?.agents ?? []).map((item) => item.name), resolvedProjectAgentName]),
    [bundleQuery.data, resolvedProjectAgentName],
  );
  const agentModeOptions = useMemo(
    () => buildSelectOptions([...(bundleQuery.data?.modes ?? []).map((item) => item.name), resolvedProjectAgentMode]),
    [bundleQuery.data, resolvedProjectAgentMode],
  );

  const runQuery = useQuery({
    queryKey: ['run', selectedRunId],
    queryFn: () => apiClient.getRun(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: (query) => {
      const status = (query.state.data as PipelineRun | undefined)?.status;
      return status === 'running' || status === 'queued' ? 2000 : false;
    },
  });

  const eventsQuery = useQuery({
    queryKey: ['run-events', selectedRunId],
    queryFn: () => apiClient.listRunEvents(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' ? 1500 : false;
    },
  });

  const completionQuery = useQuery({
    queryKey: ['run-completion', selectedRunId],
    queryFn: () => apiClient.getRunCompletion(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' ? 1500 : false;
    },
  });

  const agentEnabled = Boolean(selectedRunId && (runQuery.data?.step_by_step || runQuery.data?.full_cycle));
  const agentRunQuery = useQuery({
    queryKey: ['run-agent', selectedRunId],
    queryFn: () => apiClient.getRunAgent(selectedRunId),
    enabled: agentEnabled,
    retry: false,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' ? 1500 : false;
    },
  });
  const agentStepsQuery = useQuery({
    queryKey: ['run-agent-steps', selectedRunId],
    queryFn: () => apiClient.listRunAgentSteps(selectedRunId),
    enabled: agentEnabled,
    retry: false,
  });
  const agentToolCallsQuery = useQuery({
    queryKey: ['run-agent-tool-calls', selectedRunId],
    queryFn: () => apiClient.listRunAgentToolCalls(selectedRunId),
    enabled: agentEnabled,
    retry: false,
  });
  const agentEvidenceQuery = useQuery({
    queryKey: ['run-agent-evidence', selectedRunId],
    queryFn: () => apiClient.listRunAgentEvidence(selectedRunId),
    enabled: agentEnabled,
    retry: false,
  });
  const agentEvaluationsQuery = useQuery({
    queryKey: ['run-agent-evaluations', selectedRunId],
    queryFn: () => apiClient.listRunAgentEvaluations(selectedRunId),
    enabled: agentEnabled,
    retry: false,
  });
  const residualItemsQuery = useQuery({
    queryKey: ['run-residual-items', selectedRunId],
    queryFn: () => apiClient.listRunResidualItems(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' || status === 'awaiting_human' ? 2000 : false;
    },
  });
  const approvalGatesQuery = useQuery({
    queryKey: ['run-approval-gates', selectedRunId],
    queryFn: () => apiClient.listRunApprovalGates(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' || status === 'awaiting_human' ? 2000 : false;
    },
  });
  const previewSessionsQuery = useQuery({
    queryKey: ['run-preview-sessions', selectedRunId],
    queryFn: () => apiClient.listRunPreviewSessions(selectedRunId),
    enabled: !!selectedRunId,
    refetchInterval: () => {
      const status = runQuery.data?.status;
      return status === 'running' || status === 'queued' || status === 'awaiting_human' || status === 'completed' ? 2000 : false;
    },
  });

  const startMutation = useMutation({
    mutationFn: apiClient.startPipeline,
    onSuccess: (run) => {
      message.success('流水线已启动');
      setManualSelectedRunId(run.id);
      setSelectedArtifactPath('');
      form.resetFields();
      form.setFieldsValue({
        simulate: true,
        full_cycle: false,
        step_by_step: false,
        iteration_limit: 3,
        llm_enhanced_loop: true,
        project_dir: projectQuery.data?.repo_path || undefined,
        context_mode: normalizeContextMode(projectQuery.data?.default_context_mode),
        context_token_budget: projectQuery.data?.default_context_token_budget,
        context_max_items: projectQuery.data?.default_context_max_items,
        context_dynamic: projectQuery.data?.default_context_dynamic,
        memory_writeback: projectQuery.data?.default_memory_writeback,
        agent_name: projectQuery.data?.default_agent_name,
        agent_mode: projectQuery.data?.default_agent_mode,
      });
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message || '启动失败'),
  });

  const retryMutation = useMutation({
    mutationFn: apiClient.retryPipeline,
    onSuccess: (run) => {
      message.success('已创建重试运行');
      setManualSelectedRunId(run.id);
      setSelectedArtifactPath('');
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['run', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-events', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-completion', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-residual-items', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-approval-gates', run.id] });
    },
    onError: (error: Error) => message.error(error.message || '重试失败'),
  });
  const resumeMutation = useMutation({
    mutationFn: apiClient.resumePipeline,
    onSuccess: (run) => {
      message.success('已发起恢复执行');
      setManualSelectedRunId(run.id);
      setSelectedArtifactPath('');
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['run', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-events', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-completion', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent-tool-calls', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent-evaluations', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-residual-items', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-approval-gates', run.id] });
    },
    onError: (error: Error) => message.error(error.message || '恢复失败'),
  });
  const approveToolMutation = useMutation({
    mutationFn: ({ runId, toolName }: { runId: string; toolName?: string }) => apiClient.approvePipelineTool(runId, toolName),
    onSuccess: (run) => {
      message.success('已确认高风险动作，继续执行');
      setManualSelectedRunId(run.id);
      setSelectedArtifactPath('');
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['run', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-events', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-completion', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent-tool-calls', run.id] });
      void queryClient.invalidateQueries({ queryKey: ['run-agent-evaluations', run.id] });
    },
    onError: (error: Error) => message.error(error.message || '确认失败'),
  });
  const autoAdvanceMutation = useMutation({
    mutationFn: apiClient.autoAdvancePipeline,
    onSuccess: (result: PipelineAutoAdvanceResult) => {
      if (result.executed && result.run) {
        message.success('Auto advanced to the next delivery run.');
        setManualSelectedRunId(result.run.id);
        setSelectedArtifactPath('');
        void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
        void queryClient.invalidateQueries({ queryKey: ['run', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-events', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-completion', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-agent', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-agent-tool-calls', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-agent-evaluations', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-residual-items', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-preview-sessions', result.run.id] });
        void queryClient.invalidateQueries({ queryKey: ['run-approval-gates', result.run.id] });
        return;
      }
      message.info(result.reason || 'No further automatic action is available.');
      if (selectedRunId) {
        void queryClient.invalidateQueries({ queryKey: ['run', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-events', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-agent', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-agent-evaluations', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-residual-items', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-preview-sessions', selectedRunId] });
        void queryClient.invalidateQueries({ queryKey: ['run-approval-gates', selectedRunId] });
      }
    },
    onError: (error: Error) => message.error(error.message || 'Auto advance failed'),
  });
  const updateResidualItemMutation = useMutation({
    mutationFn: ({
      itemId,
      status,
      resolutionNote,
    }: {
      itemId: string;
      status: 'open' | 'resolved' | 'waived';
      resolutionNote?: string;
    }) =>
      apiClient.updateResidualItem(itemId, {
        status,
        resolution_note: resolutionNote,
      }),
    onSuccess: (_, variables) => {
      message.success(variables.status === 'resolved' ? '残留项已标记为已解决' : '残留项状态已更新');
      void queryClient.invalidateQueries({ queryKey: ['run-residual-items', selectedRunId] });
      void queryClient.invalidateQueries({ queryKey: ['run-approval-gates', selectedRunId] });
      if (activeProjectId) {
        void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      }
    },
    onError: (error: Error) => message.error(error.message || '更新残留项失败'),
  });
  const updatePreviewSessionMutation = useMutation({
    mutationFn: ({
      sessionId,
      status,
      reviewerNote,
    }: {
      sessionId: string;
      status: 'generated' | 'accepted' | 'rejected';
      reviewerNote?: string;
    }) =>
      apiClient.updatePreviewSession(sessionId, {
        status,
        reviewer_note: reviewerNote,
      }),
    onSuccess: (_, variables) => {
      message.success(variables.status === 'accepted' ? '预览已验收通过' : '预览状态已更新');
      void queryClient.invalidateQueries({ queryKey: ['run-preview-sessions', selectedRunId] });
      if (activeProjectId) {
        void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      }
    },
    onError: (error: Error) => message.error(error.message || '更新预览验收状态失败'),
  });

  const completionData = completionQuery.data as PipelineCompletion | undefined;
  const selectedRun = runQuery.data as PipelineRun | undefined;
  const residualItems = residualItemsQuery.data ?? [];
  const previewSessions = previewSessionsQuery.data ?? [];
  const approvalGates = approvalGatesQuery.data ?? [];
  const latestAgentEvaluation = agentRunQuery.data?.latest_evaluation ?? agentEvaluationsQuery.data?.at(-1);
  const pendingApprovalToolCall = findPendingApprovalToolCall(
    agentToolCallsQuery.data ?? [],
    selectedRun,
    latestAgentEvaluation,
    agentRunQuery.data,
  );

  const openRunDetails = (tabKey: string = 'overview') => {
    setDetailsTabKey(tabKey);
    setDetailsModalOpen(true);
  };

  useEffect(() => {
    const project = projectQuery.data as Project | undefined;
    if (!project) {
      return;
    }
    form.setFieldsValue({
      project_dir: project.repo_path || undefined,
      context_mode: normalizeContextMode(project.default_context_mode),
      context_token_budget: project.default_context_token_budget,
      context_max_items: project.default_context_max_items,
      context_dynamic: project.default_context_dynamic,
      memory_writeback: project.default_memory_writeback,
      agent_name: resolvedProjectAgentName,
      agent_mode: resolvedProjectAgentMode,
      llm_enhanced_loop: true,
    });
  }, [form, projectQuery.data, resolvedProjectAgentName, resolvedProjectAgentMode]);

  useEffect(() => {
    if (!fullCycle) {
      if (stepByStep && form.getFieldValue('agent_mode') === 'full_cycle') {
        form.setFieldValue('agent_mode', resolvedProjectAgentMode || 'step_by_step');
      }
      return;
    }
    if (form.getFieldValue('agent_mode') !== 'full_cycle') {
      form.setFieldValue('agent_mode', 'full_cycle');
    }
  }, [form, fullCycle, stepByStep, resolvedProjectAgentMode]);

  const stageBoard = useMemo(() => buildStageBoard(completionData), [completionData]);
  const allArtifacts = useMemo(() => completionData?.artifacts ?? [], [completionData]);
  const selectedArtifact = useMemo(() => {
    if (selectedArtifactPath) {
      return allArtifacts.find((artifact) => artifact.path === selectedArtifactPath);
    }
    return pickDefaultArtifact(allArtifacts);
  }, [allArtifacts, selectedArtifactPath]);

  const artifactContentQuery = useQuery({
    queryKey: ['artifact-content', selectedRunId, selectedArtifact?.preview_url],
    queryFn: async () => {
      if (!selectedArtifact?.preview_url) {
        return '';
      }
      const response = await fetch(`${apiBase}${selectedArtifact.preview_url}`);
      if (!response.ok) {
        throw new Error('预览内容加载失败');
      }
      return response.text();
    },
    enabled:
      !!selectedArtifact?.preview_url &&
      (selectedArtifact.preview_type === 'markdown' || selectedArtifact.preview_type === 'text'),
  });

  const runColumns = useMemo<ColumnsType<PipelineRun>>(
    () => [
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 96,
        render: (value: string) => <Tag color={statusColor(value)}>{value}</Tag>,
      },
      { title: '阶段', dataIndex: 'stage', key: 'stage', width: 120 },
      { title: '进度', dataIndex: 'progress', key: 'progress', width: 96, render: (value: number) => `${value}%` },
      {
        title: '需求摘要',
        dataIndex: 'prompt',
        key: 'prompt',
        render: (value: string) => (
          <Typography.Paragraph
            ellipsis={{ rows: 2, tooltip: value }}
            style={{
              marginBottom: 0,
              display: '-webkit-box',
              WebkitLineClamp: 2,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
              lineHeight: 1.6,
            }}
          >
            {value}
          </Typography.Paragraph>
        ),
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 168,
        responsive: ['xxl'],
        render: (value: string) => dayjs(value).format('MM-DD HH:mm'),
      },
    ],
    [],
  );

  return (
    <>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        <Card variant="borderless" style={{ background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 100%)', color: '#e2e8f0', borderRadius: 24 }}>
          <Row gutter={[20, 20]} align="middle">
            <Col xs={24} xl={14}>
              <Space orientation="vertical" size={10} style={{ width: '100%' }}>
                <Tag color="gold" style={{ width: 'fit-content', borderRadius: 999 }}>Volcengine 多模态 × super-dev 闭环</Tag>
                <Typography.Title level={2} style={{ margin: 0, color: '#f8fafc', fontFamily: 'var(--heading-font)' }}>
                  构思 → 设计 → super-dev → 产出 → 再构思
                </Typography.Title>
                <Typography.Paragraph style={{ color: '#cbd5e1', marginBottom: 0 }}>
                  阶段产物会在运行过程中持续暴露，支持 Markdown / HTML / 文本内联预览，解决“只能看到第一阶段骨架文档”的问题。
                </Typography.Paragraph>
              </Space>
            </Col>
            <Col xs={24} xl={10}>
              <Row gutter={[12, 12]}>
                <Col span={12}><MetricCard label="当前批次" value={selectedBatch?.title || '项目级运行'} note={selectedBatch?.goal || '未绑定 change batch'} /></Col>
                <Col span={12}><MetricCard label="当前运行" value={selectedRun?.status || '待启动'} note={selectedRun?.stage || 'queued'} /></Col>
                <Col span={12}><MetricCard label="LLM 闭环" value={selectedRun?.llm_enhanced_loop ? '已启用' : '可启用'} note={`${selectedRun?.multimodal_assets?.length ?? 0} 个素材`} /></Col>
                <Col span={12}><MetricCard label="可预览阶段" value={`${stageBoard.filter((item) => item.artifacts.length > 0).length}/5`} note={completionData?.preview_url ? '含 HTML 预览' : '以内联预览为主'} /></Col>
              </Row>
            </Col>
          </Row>
        </Card>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={10}>
            <Card title="启动新运行" style={{ borderRadius: 20 }}>
              {!activeProjectId ? (
                <Empty description="请先选择项目" />
              ) : (
                <Form<PipelineFormValues>
                  layout="vertical"
                  form={form}
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
                    llm_enhanced_loop: true,
                  }}
                  onFinish={(values) => {
                    startMutation.mutate({
                      project_id: activeProjectId,
                      change_batch_id: activeChangeBatchId || undefined,
                      prompt: values.prompt,
                      llm_enhanced_loop: values.llm_enhanced_loop,
                      multimodal_assets: parseMultimodalAssets(values.multimodal_assets_text),
                      simulate: values.full_cycle || values.step_by_step ? false : (values.simulate ?? true),
                      full_cycle: values.full_cycle,
                      step_by_step: values.step_by_step,
                      iteration_limit: values.iteration_limit,
                      project_dir: values.project_dir,
                      platform: projectQuery.data?.default_platform,
                      frontend: projectQuery.data?.default_frontend,
                      backend: projectQuery.data?.default_backend,
                      domain: projectQuery.data?.default_domain,
                      context_mode: values.context_mode ?? 'off',
                      context_query: values.context_query,
                      context_token_budget: values.context_token_budget,
                      context_max_items: values.context_max_items,
                      context_dynamic: values.context_dynamic,
                      memory_writeback: values.memory_writeback,
                      agent_name: resolveBundleSelection(values.agent_name, bundleAgentNames, resolvedProjectAgentName),
                      agent_mode: values.full_cycle
                        ? 'full_cycle'
                        : resolveBundleSelection(values.agent_mode, bundleModeNames, resolvedProjectAgentMode || 'step_by_step'),
                    });
                  }}
                >
                  <Form.Item name="prompt" label="需求描述" rules={[{ required: true, message: '请输入需求描述' }]}>
                    <Input.TextArea rows={4} placeholder="例如：实现一个支持知识库检索和项目任务管理的开发协作平台" />
                  </Form.Item>

                  <Card size="small" style={{ marginBottom: 16, borderRadius: 16, background: '#f8fafc' }}>
                    <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                      <Typography.Text strong>当前变更批次</Typography.Text>
                      <Typography.Text>{selectedBatch?.title || '未选择，运行将只挂到项目层'}</Typography.Text>
                      <Typography.Text type="secondary">
                        {selectedBatch?.goal || '可先在“变更中心”选中 change batch，再回到这里启动。'}
                      </Typography.Text>
                    </Space>
                  </Card>

                  <Row gutter={12}>
                    <Col span={12}>
                      <Form.Item name="simulate" label="模拟模式" valuePropName="checked">
                        <Switch checkedChildren="模拟" unCheckedChildren="真实 super-dev" disabled={Boolean(fullCycle || stepByStep)} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name="llm_enhanced_loop" label="启用多模态 LLM 闭环" valuePropName="checked">
                        <Switch checkedChildren="启用" unCheckedChildren="关闭" />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Row gutter={12}>
                    <Col span={12}>
                      <Form.Item name="full_cycle" label="一键全流程交付" valuePropName="checked">
                        <Switch checkedChildren="开启" unCheckedChildren="关闭" disabled={Boolean(stepByStep)} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item name="step_by_step" label="按 super-dev 原生步骤执行" valuePropName="checked">
                        <Switch checkedChildren="开启" unCheckedChildren="关闭" disabled={Boolean(fullCycle)} />
                      </Form.Item>
                    </Col>
                  </Row>

                  {fullCycle ? (
                    <Form.Item name="iteration_limit" label="开发-单测-修复迭代次数">
                      <InputNumber min={1} max={8} style={{ width: '100%' }} />
                    </Form.Item>
                  ) : null}

                  <Card size="small" title="Agent Strategy" style={{ marginBottom: 16, borderRadius: 16 }}>
                    {agentFallbackNotice ? (
                      <Alert
                        showIcon
                        type="warning"
                        data-testid="pipeline-agent-default-fallback-alert"
                        style={{ marginBottom: 12, borderRadius: 14 }}
                        title="检测到项目默认 Agent 配置已失效"
                        description={agentFallbackNotice}
                      />
                    ) : null}
                    <Row gutter={12}>
                      <Col span={12}>
                        <Form.Item name="agent_name" label="Agent">
                          <Select options={agentOptions} placeholder="delivery-agent" disabled={!(stepByStep || fullCycle)} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="agent_mode" label="Agent Mode">
                          <Select options={agentModeOptions} placeholder="full_cycle / step_by_step" disabled={!(stepByStep || fullCycle)} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Typography.Text type="secondary">
                      Options come from the project `.studio-agent` bundle; AgentRun is created when step_by_step or full_cycle is enabled.
                    </Typography.Text>
                  </Card>

                  <Form.Item name="multimodal_assets_text" label="多模态参考素材 URL（每行一个，可选）">
                    <Input.TextArea rows={4} placeholder={'https://example.com/wireframe.png\nhttps://example.com/brand-board.jpg'} disabled={!llmEnhancedLoop} />
                  </Form.Item>

                  {llmEnhancedLoop ? (
                    <Alert showIcon type="info" style={{ marginBottom: 16, borderRadius: 14 }} title="已启用火山引擎多模态闭环" description="会自动生成构思稿、设计复核稿和复盘再构思稿，并在右侧阶段看板持续刷新。" />
                  ) : null}

                  <Form.Item name="project_dir" label="目标项目目录（可选）">
                    <Input placeholder="D:/Work/target-project" />
                  </Form.Item>

                  <Card size="small" title="上下文强化" style={{ marginBottom: 16, borderRadius: 16 }}>
                    <Row gutter={12}>
                      <Col span={12}>
                        <Form.Item name="context_mode" label="上下文模式">
                          <Input placeholder="off / auto / manual" />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="context_token_budget" label="Token 预算">
                          <InputNumber min={200} max={6000} step={100} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Row gutter={12}>
                      <Col span={12}>
                        <Form.Item name="context_max_items" label="最大条目数">
                          <InputNumber min={1} max={20} style={{ width: '100%' }} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="context_dynamic" label="动态阶段上下文" valuePropName="checked">
                          <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item name="memory_writeback" label="运行后写回记忆" valuePropName="checked" style={{ marginBottom: contextMode === 'manual' ? 12 : 0 }}>
                      <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                    </Form.Item>
                    {contextMode === 'manual' ? (
                      <Form.Item name="context_query" label="手动上下文查询" rules={[{ required: true, message: 'manual 模式下请输入 context_query' }]} style={{ marginBottom: 0 }}>
                        <Input.TextArea rows={3} placeholder="例如：检索最近 2 次质量门禁失败的根因与修复方案" />
                      </Form.Item>
                    ) : null}
                  </Card>

                  <Button type="primary" htmlType="submit" block size="large" loading={startMutation.isPending}>
                    启动流水线
                  </Button>
                </Form>
              )}
            </Card>
          </Col>

          <Col xs={24} xl={14}>
            <Space orientation="vertical" size="large" style={{ width: '100%' }}>
              <Card title="运行列表" extra={selectedRun ? <Tag color={statusColor(selectedRun.status)}>{selectedRun.status}</Tag> : null} style={{ borderRadius: 20 }}>
                {!activeProjectId ? (
                  <Empty description="请先选择项目" />
                ) : (
                  <Table<PipelineRun>
                    rowKey="id"
                    columns={runColumns}
                    dataSource={runs}
                    pagination={false}
                    scroll={{ x: 960, y: 360 }}
                    locale={{ emptyText: '当前项目暂无运行记录' }}
                    onRow={(record) => ({
                      onClick: () => {
                        setManualSelectedRunId(record.id);
                        setSelectedArtifactPath('');
                      },
                      style: {
                        cursor: 'pointer',
                        background: record.id === selectedRunId ? 'rgba(59, 130, 246, 0.08)' : undefined,
                      },
                    })}
                  />
                )}
              </Card>

              <Card
                title={"选中运行"}
                style={{ borderRadius: 20 }}
                extra={selectedRun ? (
                  <Space wrap>
                    <Button type="primary" data-testid="pipeline-run-details-open" onClick={() => openRunDetails('overview')}>
                      {"查看运行详情"}
                    </Button>
                    {(completionData?.artifacts?.length || completionData?.preview_url) ? (
                      <Button onClick={() => openRunDetails('preview')}>{"查看产物预览"}</Button>
                    ) : null}
                  </Space>
                ) : null}
              >
                {!selectedRun ? (
                  <Empty description={"请选择运行记录"} />
                ) : (
                  <Space orientation="vertical" size="middle" style={{ width: '100%' }} data-testid="pipeline-run-summary">
                    <Space wrap>
                      <Tag color={statusColor(selectedRun.status)}>{selectedRun.status}</Tag>
                      <Tag>{selectedRun.stage}</Tag>
                      {selectedRun.llm_enhanced_loop ? <Tag color="purple">LLM {"闭环"}</Tag> : null}
                      {selectedRun.full_cycle ? <Tag color="cyan">full-cycle</Tag> : null}
                      {selectedRun.step_by_step ? <Tag color="blue">step-by-step</Tag> : null}
                      {selectedRun.simulate ? <Tag color="orange">simulate</Tag> : null}
                    </Space>
                    <Typography.Paragraph type="secondary" ellipsis={{ rows: 2, tooltip: selectedRun.prompt }} style={{ marginBottom: 0 }}>
                      {selectedRun.prompt}
                    </Typography.Paragraph>
                    {(selectedRun.status === 'awaiting_human' || latestAgentEvaluation?.verdict === 'need_context') ? (
                      <Alert
                        showIcon
                        type={selectedRun.status === 'awaiting_human' ? 'warning' : 'info'}
                        style={{ marginBottom: 0, borderRadius: 14 }}
                        title={selectedRun.status === 'awaiting_human'
                          ? pendingApprovalToolCall
                            ? '高风险动作待确认'
                            : '需要人工接管'
                          : 'Agent 曾请求补强上下文'}
                        description={selectedRun.status === 'awaiting_human'
                          ? pendingApprovalToolCall
                            ? `${pendingApprovalToolCall.reason || '高风险 deploy 动作需要人工确认后继续。'}${pendingApprovalToolCall.risk_level ? `；风险级别：${pendingApprovalToolCall.risk_level}` : ''}`
                            : latestAgentEvaluation
                              ? `${latestAgentEvaluation.reason}${latestAgentEvaluation.next_action ? `；建议动作：${latestAgentEvaluation.next_action}` : ''}`
                              : 'Agent 已暂停，等待人工确认后继续。'
                          : `${latestAgentEvaluation?.reason ?? '本次运行曾请求补强上下文。'}${latestAgentEvaluation?.next_action ? `；下一步：${latestAgentEvaluation.next_action}` : ''}`}
                      />
                    ) : null}
                    {(selectedRun.status === 'failed' || selectedRun.status === 'awaiting_human' || selectedRun.status === 'completed') ? (
                      <Space wrap>
                        {selectedRun.status === 'failed' ? (
                          <Button danger onClick={() => retryMutation.mutate(selectedRun.id)} loading={retryMutation.isPending}>
                            {"重试失败运行"}
                          </Button>
                        ) : null}
                        {(selectedRun.status === 'failed' || selectedRun.status === 'completed') ? (
                          <Button onClick={() => autoAdvanceMutation.mutate(selectedRun.id)} loading={autoAdvanceMutation.isPending}>
                            {"Auto advance"}
                          </Button>
                        ) : null}
                        {selectedRun.status === 'awaiting_human' ? (
                          pendingApprovalToolCall ? (
                            <Button
                              type="primary"
                              onClick={() => approveToolMutation.mutate({ runId: selectedRun.id, toolName: pendingApprovalToolCall.tool_name })}
                              loading={approveToolMutation.isPending}
                            >
                              {"确认高风险动作并继续"}
                            </Button>
                          ) : (
                            <Button type="primary" onClick={() => resumeMutation.mutate(selectedRun.id)} loading={resumeMutation.isPending}>
                              {"人工确认后恢复"}
                            </Button>
                          )
                        ) : null}
                      </Space>
                    ) : null}
                    <Progress percent={selectedRun.progress} strokeColor={{ from: '#0ea5e9', to: '#7c3aed' }} />
                    <Descriptions size="small" column={{ xs: 1, sm: 2 }}>
                      <Descriptions.Item label={"运行 ID"}>{selectedRun.id}</Descriptions.Item>
                      <Descriptions.Item label={"运行模式"}>{formatRunMode(selectedRun)}</Descriptions.Item>
                      <Descriptions.Item label={"更新时间"}>{dayjs(selectedRun.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
                      <Descriptions.Item label={"产物数"}>{completionData?.artifacts?.length ?? 0}</Descriptions.Item>
                    </Descriptions>
                  </Space>
                )}
              </Card>

              <DeliveryHandoffCard
                run={selectedRun}
                completion={completionData}
                events={eventsQuery.data ?? []}
                previewSessions={previewSessions}
                approvalGates={approvalGates}
                residualItems={residualItems}
                apiBase={apiBase}
                loading={
                  completionQuery.isLoading ||
                  eventsQuery.isLoading ||
                  previewSessionsQuery.isLoading ||
                  approvalGatesQuery.isLoading ||
                  residualItemsQuery.isLoading
                }
              />

              <RunFollowupsCard
                residualItems={residualItems}
                approvalGates={approvalGates}
                loading={residualItemsQuery.isLoading || approvalGatesQuery.isLoading}
                onResolveResidual={(itemId) => updateResidualItemMutation.mutate({ itemId, status: 'resolved' })}
                resolvingItemId={updateResidualItemMutation.variables?.itemId}
              />

              <PreviewAcceptanceCard
                previewSessions={previewSessions}
                loading={previewSessionsQuery.isLoading}
                apiBase={apiBase}
                onAccept={(sessionId) => updatePreviewSessionMutation.mutate({ sessionId, status: 'accepted' })}
                onReject={(sessionId) => updatePreviewSessionMutation.mutate({ sessionId, status: 'rejected' })}
                updatingSessionId={updatePreviewSessionMutation.variables?.sessionId}
              />
            </Space>
          </Col>
        </Row>
      </Space>

      <PipelineRunDetailsModal
        open={detailsModalOpen && Boolean(selectedRun)}
        activeTab={detailsTabKey}
        onTabChange={setDetailsTabKey}
        onClose={() => setDetailsModalOpen(false)}
        selectedRun={selectedRun}
        completionData={completionData}
        stageBoardContent={
          <PipelineStageBoardCard
            loading={completionQuery.isLoading}
            completionData={completionData}
            stageBoard={stageBoard}
            selectedArtifact={selectedArtifact}
            previewVisible={false}
            onTogglePreview={() => setDetailsTabKey('preview')}
            onSelectArtifact={setSelectedArtifactPath}
          />
        }
        previewContent={
          <PipelineArtifactPreviewPanel
            apiBase={apiBase}
            selectedArtifact={selectedArtifact}
            artifactContent={artifactContentQuery.data}
            artifactLoading={artifactContentQuery.isLoading}
            artifactLoadFailed={artifactContentQuery.isError}
            previewVisible={Boolean(completionData?.preview_url)}
            previewUrl={completionData?.preview_url}
          />
        }
        executionContent={
          <Row gutter={[16, 16]} align="top">
            <Col xs={24} xl={10}>
              <PipelineChecklistCard checklist={completionData?.checklist ?? []} />
            </Col>
            <Col xs={24} xl={14}>
              <PipelineTimelineCard events={eventsQuery.data ?? []} />
            </Col>
          </Row>
        }
        agentContent={
          <AgentObservabilityCard
            agentRun={agentRunQuery.data}
            steps={agentStepsQuery.data ?? []}
            toolCalls={agentToolCallsQuery.data ?? []}
            evidence={agentEvidenceQuery.data ?? []}
            evaluations={agentEvaluationsQuery.data ?? []}
            visible={Boolean(runQuery.data?.step_by_step || runQuery.data?.full_cycle)}
          />
        }
        onRetry={selectedRun ? () => retryMutation.mutate(selectedRun.id) : undefined}
        retryLoading={retryMutation.isPending}
      />

      <FloatButton.BackTop visibilityHeight={240} />
    </>
  );
}

function MetricCard({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <div style={{ padding: '14px 16px', borderRadius: 18, background: 'rgba(255,255,255,0.08)', minHeight: 110 }}>
      <Typography.Text style={{ color: '#94a3b8', fontSize: 12 }}>{label}</Typography.Text>
      <Typography.Title level={5} style={{ margin: '8px 0 6px', color: '#f8fafc' }}>{value}</Typography.Title>
      <Typography.Text style={{ color: '#cbd5e1', fontSize: 12 }}>{note}</Typography.Text>
    </div>
  );
}

function buildStageBoard(completion?: PipelineCompletion) {
  if (!completion?.stages?.length) {
    return STAGE_FALLBACK;
  }
  return STAGE_FALLBACK.map((fallback) => completion.stages.find((stage) => stage.key === fallback.key) ?? fallback);
}

function pickDefaultArtifact(artifacts: PipelineArtifact[]) {
  const priority = ['html', 'markdown', 'text', 'image', 'binary'];
  for (const type of priority) {
    const item = artifacts.find((artifact) => artifact.preview_type === type);
    if (item) {
      return item;
    }
  }
  return artifacts[0];
}

function buildSelectOptions(values: Array<string | undefined>) {
  return Array.from(new Set(values.map((item) => (item ?? "").trim()).filter(Boolean))).map((value) => ({ value, label: value }));
}


function hasNamedOption(options: string[], value?: string) {
  const normalizedValue = (value ?? '').trim().toLowerCase();
  return normalizedValue !== '' && options.some((item) => item.trim().toLowerCase() === normalizedValue);
}

function firstNonEmptyValue(...values: Array<string | undefined>) {
  for (const value of values) {
    const normalizedValue = (value ?? '').trim();
    if (normalizedValue) {
      return normalizedValue;
    }
  }
  return '';
}

function resolveBundleSelection(rawValue: string | undefined, availableOptions: string[], fallbackValue: string) {
  const normalizedValue = (rawValue ?? '').trim();
  const normalizedFallback = firstNonEmptyValue(fallbackValue, availableOptions[0]);
  if (availableOptions.length === 0) {
    return firstNonEmptyValue(normalizedValue, normalizedFallback);
  }
  if (normalizedValue && hasNamedOption(availableOptions, normalizedValue)) {
    return normalizedValue;
  }
  if (normalizedFallback && hasNamedOption(availableOptions, normalizedFallback)) {
    return normalizedFallback;
  }
  return availableOptions[0];
}

function parseMultimodalAssets(raw?: string) {
  if (!raw) {
    return [] as string[];
  }
  return raw
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter((item, index, array) => item && array.indexOf(item) === index);
}

function normalizeContextMode(raw?: string): 'off' | 'auto' | 'manual' {
  switch (raw) {
    case 'manual':
      return 'manual';
    case 'off':
      return 'off';
    default:
      return 'auto';
  }
}

type ToolApprovalState = {
  tool_name: string;
  status?: string;
  risk_level?: string;
  requires_confirmation?: boolean;
  approved?: boolean;
  reason?: string;
};

function parseToolApprovalState(call: AgentToolCall): ToolApprovalState | undefined {
  try {
    const payload = JSON.parse(call.response_json || '{}') as Omit<ToolApprovalState, 'tool_name'>;
    if (!payload || typeof payload !== 'object') {
      return undefined;
    }
    return {
      tool_name: call.tool_name,
      status: typeof payload.status === 'string' ? payload.status : undefined,
      risk_level: typeof payload.risk_level === 'string' ? payload.risk_level : undefined,
      requires_confirmation: typeof payload.requires_confirmation === 'boolean' ? payload.requires_confirmation : undefined,
      approved: typeof payload.approved === 'boolean' ? payload.approved : undefined,
      reason: typeof payload.reason === 'string' ? payload.reason : undefined,
    };
  } catch {
    return undefined;
  }
}

function findPendingApprovalToolCall(
  toolCalls: AgentToolCall[],
  run?: PipelineRun,
  latestEvaluation?: AgentEvaluation,
  agentRun?: PipelineRunAgent,
) {
  for (const call of [...toolCalls].reverse()) {
    const state = parseToolApprovalState(call);
    if (state?.requires_confirmation && !state.approved && state.status === 'awaiting_approval') {
      return state;
    }
  }

  const approvalStage = run?.stage || agentRun?.run?.current_node;
  const awaitingHuman = run?.status === 'awaiting_human' || agentRun?.run?.status === 'awaiting_human';
  if (awaitingHuman && approvalStage === FULL_CYCLE_RELEASE_APPROVAL_STAGE) {
    return {
      tool_name: FULL_CYCLE_DEPLOY_TOOL,
      status: 'awaiting_approval',
      risk_level: 'high',
      requires_confirmation: true,
      approved: false,
      reason: latestEvaluation?.reason,
    } satisfies ToolApprovalState;
  }

  return undefined;
}

function statusColor(status?: string) {
  switch (status) {
    case 'completed':
      return 'green';
    case 'failed':
      return 'red';
    case 'awaiting_human':
    case 'blocked':
      return 'gold';
    case 'queued':
      return 'orange';
    default:
      return 'blue';
  }
}

function previewSessionStatusColor(status?: string) {
  switch ((status || '').toLowerCase()) {
    case 'accepted':
      return 'green';
    case 'rejected':
      return 'red';
    default:
      return 'blue';
  }
}

function buildAbsolutePreviewHref(apiBase: string, previewUrl?: string) {
  if (!previewUrl) {
    return '';
  }
  if (/^https?:\/\//.test(previewUrl)) {
    return previewUrl;
  }
  return `${apiBase}${previewUrl}`;
}

function residualSeverityColor(severity?: string) {
  switch ((severity || '').toLowerCase()) {
    case 'critical':
      return 'red';
    case 'high':
      return 'volcano';
    case 'medium':
      return 'gold';
    default:
      return 'blue';
  }
}

function agentVerdictColor(verdict?: string) {
  switch (verdict) {
    case 'pass':
      return 'green';
    case 'retry':
      return 'orange';
    case 'need_context':
      return 'blue';
    case 'need_human':
      return 'gold';
    default:
      return 'red';
  }
}

function formatRunMode(run: PipelineRun) {
  if (run.step_by_step) {
    return 'step-by-step';
  }
  if (run.full_cycle) {
    return 'full-cycle';
  }
  if (run.simulate) {
    return 'simulate';
  }
  return 'super-dev';
}

function RunFollowupsCard({
  residualItems,
  approvalGates,
  loading,
  onResolveResidual,
  resolvingItemId,
}: {
  residualItems: ResidualItem[];
  approvalGates: ApprovalGate[];
  loading: boolean;
  onResolveResidual: (itemId: string) => void;
  resolvingItemId?: string;
}) {
  const openResiduals = residualItems.filter((item) => item.status === 'open');
  const openGates = approvalGates.filter((item) => item.status === 'open');

  return (
    <Card title="Open follow-ups" style={{ borderRadius: 20 }} loading={loading}>
      {!openResiduals.length && !openGates.length ? (
        <Empty description="No approval gates or residual follow-ups remain." />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Space wrap>
            <Tag color="volcano">Open Residuals {openResiduals.length}</Tag>
            <Tag color="gold">Open Gates {openGates.length}</Tag>
          </Space>

          {openGates.length ? (
            <div>
              <Typography.Text strong>Approval gates</Typography.Text>
              <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
                {openGates.map((gate) => (
                  <div key={gate.id} style={{ border: '1px solid #fcd34d', borderRadius: 12, padding: 12 }}>
                    <Space wrap>
                      <Tag color="gold">{gate.status}</Tag>
                      <Typography.Text strong>{gate.title}</Typography.Text>
                      {gate.tool_name ? <Typography.Text code>{gate.tool_name}</Typography.Text> : null}
                      {gate.risk_level ? <Tag color="magenta">risk:{gate.risk_level}</Tag> : null}
                    </Space>
                    <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                      {gate.detail}
                    </Typography.Paragraph>
                  </div>
                ))}
              </Space>
            </div>
          ) : null}

          {openResiduals.length ? (
            <div>
              <Typography.Text strong>Residual items</Typography.Text>
              <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
                {openResiduals.map((item) => (
                  <div key={item.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                    <Space wrap>
                      <Tag color={residualSeverityColor(item.severity)}>{item.severity}</Tag>
                      <Tag>{item.category}</Tag>
                      <Typography.Text strong>{item.summary}</Typography.Text>
                    </Space>
                    {item.evidence ? (
                      <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                        {item.evidence}
                      </Typography.Paragraph>
                    ) : null}
                    {item.suggested_command ? (
                      <Typography.Paragraph style={{ marginBottom: 0, marginTop: 8 }}>
                        Suggested command: <Typography.Text code>{item.suggested_command}</Typography.Text>
                      </Typography.Paragraph>
                    ) : null}
                    <Space style={{ marginTop: 8 }}>
                      <Button
                        size="small"
                        onClick={() => onResolveResidual(item.id)}
                        loading={resolvingItemId === item.id}
                      >
                        Mark resolved
                      </Button>
                    </Space>
                  </div>
                ))}
              </Space>
            </div>
          ) : null}
        </Space>
      )}
    </Card>
  );
}

function PreviewAcceptanceCard({
  previewSessions,
  loading,
  apiBase,
  onAccept,
  onReject,
  updatingSessionId,
}: {
  previewSessions: PreviewSession[];
  loading: boolean;
  apiBase: string;
  onAccept: (sessionId: string) => void;
  onReject: (sessionId: string) => void;
  updatingSessionId?: string;
}) {
  const latestSessions = [...previewSessions].sort((left, right) => dayjs(right.updated_at).valueOf() - dayjs(left.updated_at).valueOf());

  return (
    <Card title="预览与验收" style={{ borderRadius: 20 }} loading={loading}>
      {!latestSessions.length ? (
        <Empty description="当前运行尚未生成可追踪的预览会话" />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Tag color="blue">Preview Sessions {latestSessions.length}</Tag>
          {latestSessions.map((session) => {
            const previewHref = buildAbsolutePreviewHref(apiBase, session.preview_url);
            const pending = updatingSessionId === session.id;
            return (
              <div key={session.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                <Space wrap>
                  <Tag color={previewSessionStatusColor(session.status)}>{session.status}</Tag>
                  <Typography.Text strong>{session.title}</Typography.Text>
                  <Typography.Text code>{session.preview_type || 'html'}</Typography.Text>
                  <Typography.Text type="secondary">{dayjs(session.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Typography.Text>
                </Space>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                  {session.reviewer_note || '等待人工预览确认；确认通过后可视为最终预览版。'}
                </Typography.Paragraph>
                <Space wrap style={{ marginTop: 8 }}>
                  {previewHref ? (
                    <Button onClick={() => window.open(previewHref, '_blank', 'noopener,noreferrer')}>打开预览</Button>
                  ) : null}
                  <Button type="primary" onClick={() => onAccept(session.id)} loading={pending} disabled={session.status === 'accepted'}>
                    验收通过
                  </Button>
                  <Button danger onClick={() => onReject(session.id)} loading={pending} disabled={session.status === 'rejected'}>
                    退回修改
                  </Button>
                </Space>
              </div>
            );
          })}
        </Space>
      )}
    </Card>
  );
}

function AgentObservabilityCard({

  agentRun,
  steps,
  toolCalls,
  evidence,
  evaluations,
  visible,
}: {
  agentRun?: PipelineRunAgent;
  steps: AgentStep[];
  toolCalls: AgentToolCall[];
  evidence: AgentEvidence[];
  evaluations: AgentEvaluation[];
  visible: boolean;
}) {
  if (!visible) {
    return null;
  }

  if (!agentRun) {
    return (
      <Card title="Agent 轨迹" style={{ borderRadius: 20 }}>
        <Empty description="当前运行尚未生成 Agent 轨迹" />
      </Card>
    );
  }

  const latestEvaluation = agentRun.latest_evaluation ?? evaluations[evaluations.length - 1];

  return (
    <Card title="Agent 轨迹" style={{ borderRadius: 20 }}>
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        {latestEvaluation && (latestEvaluation.verdict === 'need_human' || latestEvaluation.verdict === 'need_context') ? (
          <Alert
            showIcon
            type={latestEvaluation.verdict === 'need_human' ? 'warning' : 'info'}
            message={latestEvaluation.verdict === 'need_human' ? '最新评估：需要人工接管' : '最新评估：需要补强上下文'}
            description={`${latestEvaluation.reason}${latestEvaluation.next_action ? `；Next: ${latestEvaluation.next_action}` : ''}`}
            style={{ borderRadius: 14 }}
          />
        ) : null}

        <Descriptions size="small" column={1}>
          <Descriptions.Item label="Agent">{agentRun.run.agent_name}</Descriptions.Item>
          <Descriptions.Item label="Mode">{agentRun.run.mode_name}</Descriptions.Item>
          <Descriptions.Item label="当前节点">{agentRun.run.current_node || '-'}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColor(agentRun.run.status)}>{agentRun.run.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="摘要">{agentRun.run.summary || '等待 Agent 输出摘要'}</Descriptions.Item>
        </Descriptions>

        <Space wrap>
          <Tag color="blue">Steps {agentRun.step_count}</Tag>
          <Tag color="purple">Tool Calls {agentRun.tool_call_count}</Tag>
          <Tag color="gold">Evidence {agentRun.evidence_count}</Tag>
          <Tag color="green">Evaluations {agentRun.evaluation_count}</Tag>
        </Space>

        <div>
          <Typography.Text strong>最近步骤</Typography.Text>
          <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
            {steps.slice(-4).map((step) => (
              <div key={step.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                <Space wrap>
                  <Tag color={statusColor(step.status)}>{step.status}</Tag>
                  <Typography.Text strong>{step.title || step.node_name}</Typography.Text>
                </Space>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                  {step.decision_summary || step.node_name}
                </Typography.Paragraph>
              </div>
            ))}
          </Space>
        </div>

        <div>
          <Typography.Text strong>最近工具调用</Typography.Text>
          <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
            {toolCalls.slice(-3).map((call) => {
              const approvalState = parseToolApprovalState(call);
              return (
                <div key={call.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                  <Space wrap>
                    <Tag color={call.success ? 'green' : approvalState?.status === 'awaiting_approval' ? 'gold' : 'red'}>
                      {call.success ? 'success' : approvalState?.status === 'awaiting_approval' ? 'awaiting_approval' : 'failed'}
                    </Tag>
                    <Typography.Text code>{call.tool_name}</Typography.Text>
                    {approvalState?.risk_level ? <Tag color="magenta">risk:{approvalState.risk_level}</Tag> : null}
                    <Typography.Text type="secondary">{call.latency_ms} ms</Typography.Text>
                  </Space>
                </div>
              );
            })}
          </Space>
        </div>

        <div>
          <Typography.Text strong>关键证据</Typography.Text>
          <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
            {evidence.slice(-3).map((item) => (
              <div key={item.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                <Space wrap>
                  <Tag>{item.source_type}</Tag>
                  <Typography.Text strong>{item.title || item.source_id}</Typography.Text>
                  <Typography.Text type="secondary">score {item.score.toFixed(2)}</Typography.Text>
                </Space>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                  {item.snippet}
                </Typography.Paragraph>
              </div>
            ))}
          </Space>
        </div>

        <div>
          <Typography.Text strong>评估结果</Typography.Text>
          <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 8 }}>
            {evaluations.slice(-3).map((item) => (
              <div key={item.id} style={{ border: '1px solid #e5e7eb', borderRadius: 12, padding: 12 }}>
                <Space wrap>
                  <Tag color={agentVerdictColor(item.verdict)}>{item.verdict}</Tag>
                  <Typography.Text>{item.reason}</Typography.Text>
                </Space>
                {item.acceptance_delta ? (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                    Acceptance delta: {item.acceptance_delta}
                  </Typography.Paragraph>
                ) : null}
                {item.missing_items?.length ? (
                  <Space wrap style={{ marginTop: 8 }}>
                    {item.missing_items.map((missingItem) => (
                      <Tag key={`${item.id}-${missingItem}`} color="orange">
                        {missingItem}
                      </Tag>
                    ))}
                  </Space>
                ) : null}
                {item.next_action ? (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                    Next: {item.next_action}
                  </Typography.Paragraph>
                ) : null}
                {item.next_command ? (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                    Dispatch: {item.next_command}
                  </Typography.Paragraph>
                ) : null}
              </div>
            ))}
          </Space>
        </div>
      </Space>
    </Card>
  );
}
