import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
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
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import AutonomyActivityCard from '../components/pipeline/AutonomyActivityCard';
import DeliveryHandoffCard from '../components/pipeline/DeliveryHandoffCard';
import DeliveryLedgerCard, { type DeliveryLedgerRunSignal } from '../components/pipeline/DeliveryLedgerCard';
import DeliveryProcessPreviewCard from '../components/pipeline/DeliveryProcessPreviewCard';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';
import type {
  ApprovalGate,
  PipelineAutoAdvanceResult,
  PreviewSession,
  RequirementDocVersion,
  RequirementSessionBundle,
  ResidualItem,
  RunEvent,
} from '../types';

const zh = {
  missingWorkspace: '缺少工作区',
  missingSession: '缺少会话',
  draftCreated: '已生成需求草案，请先确认理解是否正确',
  draftRevised: '已重新生成需求草案，请再次确认',
  confirmDeliveryFailed:
    '需求已确认，但自动交付启动失败，请前往交付看板继续处理',
  confirmDeliveryStarted: '需求已确认，系统已启动自动交付流程',
  autoAdvanced: '系统已自动推进到下一轮交付',
  noNeedAutoAdvance: '当前运行无需继续自动推进',
  autoAdvanceFailed: '自动推进失败',
  previewAccepted: '预览已验收通过，系统继续评估下一步',
  previewRejected: '预览已驳回，系统将准备重新交付',
  previewUpdateFailed: '更新预览验收状态失败',
  finalAcceptanceRecorded: '?????????',
  finalAcceptanceReopened: '?????????',
  finalAcceptanceUpdateFailed: '??????????',
  previewAcceptedLabel: '已通过',
  previewRejectedLabel: '已驳回',
  previewPendingLabel: '待验收',
  previewMissingLabel: '未生成',
  fullCycleMode: '全自动模式',
  stepByStepMode: '分步自治模式',
  sessionConfirmed: '已确认',
  sessionAwaitingConfirm: '待确认',
  sessionDraft: '草稿',
  blockApproval: '需要人工处理高风险审批',
  blockAwaitingHuman: '等待人工确认后继续',
  blockPreviewReview: '请先完成预览验收',
  deliveryCompleted: '交付已完成',
  needHumanAction: '存在需要人工确认的事项',
  approvalFallback:
    '系统检测到当前步骤涉及高风险操作，请前往交付看板处理。',
  previewFallback: '系统已生成预览，请先确认是否满足需求。',
  failedDelivery: '交付运行失败，系统准备继续修复',
  failedDeliveryFallback:
    '系统已经收集失败信息，并准备继续推进下一轮修复。',
  previewReady: '已生成可预览产物',
  previewReadyDesc:
    '你可以直接打开预览，也可以在本页完成验收并继续自动推进。',
  deliveryRunning: '正在持续推进交付',
  deliveryRunningDesc:
    '系统会持续轮询当前运行状态，并在可自动推进时自动继续执行。',
  nextStepPrefix: '下一步：',
  pageTitle: '智能交付',
  pickProject: '请先选择一个工作区项目',
  requestInputCard: '需求输入',
  requestInputTitle: '输入一句需求即可开始',
  requestInputDesc:
    '系统会先生成需求摘要、PRD、开发计划和风险清单，再以 {mode} 启动标准 super-dev 交付流程。',
  titleLabel: '需求标题（可选）',
  titlePlaceholder: '例如：时间线记事本 + 知识图谱',
  inputLabel: '需求描述',
  inputRequired: '请输入需求内容',
  inputPlaceholder: '一句话或一段话描述需求',
  generateDraft: '生成需求草案',
  refreshSession: '刷新会话',
  createdAt: '创建时间',
  updatedAt: '更新',
  summaryDoc: '需求摘要',
  riskDoc: '风险与待确认项',
  prdDoc: 'PRD 草案',
  planDoc: '开发计划草案',
  confirmAndStart: '确认并启动交付',
  regenerateDraft: '重新生成草案',
  confirmedAlert: '需求已确认',
  deliveryStartFailed: '自动交付未成功启动',
  deliveryResult: '交付结果',
  openPipeline: '打开交付看板',
  openPreview: '打开预览',
  runId: '运行 ID',
  previewStatus: '预览状态',
  pendingItems: '待处理事项',
  continueAutoAdvance: '继续自动推进',
  approvePreview: '验收通过',
  rejectPreview: '验收驳回',
  handleApproval: '前往处理审批',
  previewTagPrefix: '预览:',
  approvalTagPrefix: '审批',
  residualTagPrefix: '残留',
  resultCockpitTitle: '结果驾驶舱',
  resultCockpitOverview: '总览',
  resultCockpitAutonomy: '自治过程',
  resultCockpitHistory: '交付历史',
  resultCockpitOverviewDesc: '集中查看过程预览、交付交接和当前验收动作。',
  resultCockpitAutonomyDesc: '查看 Agent 如何评估结果、跟踪残留项并驱动下一步 super-dev 执行。',
  resultCockpitHistoryDesc: '回看当前变更批次的交付尝试、关键信号和质量收敛轨迹。',
  howToUse: '如何使用',
  howTo1:
    '1) 输入一句需求，系统会先生成摘要、PRD、计划和风险说明。',
  howTo2:
    '2) 你只需要确认需求草案是否正确，普通问题会自动继续推进。',
  howTo3:
    '3) 交付过程中，系统会基于 Agent 评估结果自动触发 super-dev 的下一步标准命令。',
  howTo4:
    '4) 只有高风险审批和最终预览验收需要你介入处理，其余步骤会持续自动推进。',
};

function withNextStep(text: string, nextCommand?: string) {
  if (!nextCommand) {
    return text;
  }
  return `${text} ${zh.nextStepPrefix}${nextCommand}`;
}

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

function previewStatusLabel(status?: string) {
  switch (status) {
    case 'accepted':
      return zh.previewAcceptedLabel;
    case 'rejected':
      return zh.previewRejectedLabel;
    case 'generated':
      return zh.previewPendingLabel;
    default:
      return zh.previewMissingLabel;
  }
}

function previewStatusColor(status?: string) {
  switch (status) {
    case 'accepted':
      return 'green';
    case 'rejected':
      return 'red';
    case 'generated':
      return 'gold';
    default:
      return 'blue';
  }
}

function findLatestPreviewSession(items: PreviewSession[]) {
  return [...items].sort((left, right) => dayjs(right.updated_at).valueOf() - dayjs(left.updated_at).valueOf())[0];
}

function countOpenResiduals(items: ResidualItem[]) {
  return items.filter((item) => item.status === 'open').length;
}

function countOpenApprovalGates(items: ApprovalGate[]) {
  return items.filter((item) => item.status === 'open').length;
}

function buildDeliveryLedgerRunSignal({
  events,
  previewSessions,
  approvalGates,
  residualItems,
}: {
  events: RunEvent[];
  previewSessions: PreviewSession[];
  approvalGates: ApprovalGate[];
  residualItems: ResidualItem[];
}): DeliveryLedgerRunSignal {
  return {
    preview: deriveLedgerPreviewSignal(previewSessions),
    quality: deriveLedgerQualitySignal(events),
    openApprovals: countOpenApprovalGates(approvalGates),
    openResiduals: countOpenResiduals(residualItems),
  };
}

function deriveLedgerPreviewSignal(previewSessions: PreviewSession[]): DeliveryLedgerRunSignal['preview'] {
  const latest = findLatestPreviewSession(previewSessions);
  if (!latest) {
    return 'missing';
  }
  switch (latest.status) {
    case 'accepted':
      return 'accepted';
    case 'rejected':
      return 'rejected';
    case 'generated':
      return 'pending';
    default:
      return 'missing';
  }
}

function deriveLedgerQualitySignal(events: RunEvent[]): DeliveryLedgerRunSignal['quality'] {
  const qualityEvent = [...events]
    .filter((item) => item.stage.toLowerCase().includes('quality'))
    .sort((left, right) => dayjs(right.created_at).valueOf() - dayjs(left.created_at).valueOf())[0];

  if (!qualityEvent) {
    return 'pending';
  }

  const message = qualityEvent.message.toLowerCase();
  if (
    qualityEvent.status === 'failed' ||
    message.includes('quality gate failed') ||
    message.includes('still failing') ||
    message.includes('not passed')
  ) {
    return 'failed';
  }
  if (qualityEvent.status === 'completed' || message.includes('quality gate passed')) {
    return 'passed';
  }
  return 'pending';
}

function updateBundleWithRun(bundle: RequirementSessionBundle | null, runId: string, run: RequirementSessionBundle['run']) {
  if (!bundle || !run) {
    return bundle;
  }
  return {
    ...bundle,
    run,
    session: {
      ...bundle.session,
      latest_run_id: runId,
    },
  };
}

export default function SimpleDeliveryPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { activeProjectId, setActiveChangeBatchId } = useProjectState();
  const [form] = Form.useForm<{ title?: string; raw_input: string }>();
  const [sessionBundle, setSessionBundle] = useState<RequirementSessionBundle | null>(null);
  const [activeRunIdOverride, setActiveRunIdOverride] = useState('');
  const [resultView, setResultView] = useState<'overview' | 'autonomy' | 'history'>('overview');
  const [autoAdvanceState, setAutoAdvanceState] = useState<{ runId: string; result: PipelineAutoAdvanceResult } | null>(null);
  const attemptedAutoAdvanceRuns = useRef<Set<string>>(new Set());
  const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

  const invalidateRunQueries = (runId?: string) => {
    if (!runId) {
      return;
    }
    void queryClient.invalidateQueries({ queryKey: ['simple-run', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-completion', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-agent', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-events', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-preview-sessions', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-approval-gates', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-residual-items', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-run-delivery-acceptance', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-ledger-run-signal', runId] });
    void queryClient.invalidateQueries({ queryKey: ['simple-change-batch-runs'] });
  };

  const projectQuery = useQuery({
    queryKey: ['project', activeProjectId],
    queryFn: () => apiClient.getProject(activeProjectId),
    enabled: !!activeProjectId,
  });

  const latest = sessionBundle?.session;
  const latestRunId = activeRunIdOverride || latest?.latest_run_id || sessionBundle?.run?.id || '';
  const latestChangeBatchId = latest?.latest_change_batch_id || sessionBundle?.change_batch?.id || sessionBundle?.run?.change_batch_id || '';

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

  const eventsQuery = useQuery({
    queryKey: ['simple-run-events', latestRunId],
    queryFn: () => apiClient.listRunEvents(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 5000 : false,
  });

  const agentQuery = useQuery({
    queryKey: ['simple-run-agent', latestRunId],
    queryFn: async () => (await apiClient.getRunAgent(latestRunId)) ?? null,
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 2500 : false,
  });

  const previewSessionsQuery = useQuery({
    queryKey: ['simple-run-preview-sessions', latestRunId],
    queryFn: () => apiClient.listRunPreviewSessions(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 5000 : false,
  });

  const deliveryAcceptanceQuery = useQuery({
    queryKey: ['simple-run-delivery-acceptance', latestRunId],
    queryFn: () => apiClient.getRunDeliveryAcceptance(latestRunId),
    enabled: !!latestRunId,
  });

  const approvalGatesQuery = useQuery({
    queryKey: ['simple-run-approval-gates', latestRunId],
    queryFn: () => apiClient.listRunApprovalGates(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 5000 : false,
  });

  const residualItemsQuery = useQuery({
    queryKey: ['simple-run-residual-items', latestRunId],
    queryFn: () => apiClient.listRunResidualItems(latestRunId),
    enabled: !!latestRunId,
    refetchInterval: latestRunId ? 5000 : false,
  });

  const changeBatchRunsQuery = useQuery({
    queryKey: ['simple-change-batch-runs', activeProjectId, latestChangeBatchId],
    queryFn: async () => {
      if (!activeProjectId) throw new Error(zh.missingWorkspace);
      return apiClient.listRuns(activeProjectId, 50);
    },
    enabled: !!activeProjectId && !!latestChangeBatchId,
    refetchInterval: latestChangeBatchId ? 5000 : false,
  });

  const fetchSession = useMutation({
    mutationFn: async (sessionId: string) => {
      if (!activeProjectId) throw new Error(zh.missingWorkspace);
      return apiClient.getRequirementSession(activeProjectId, sessionId);
    },
    onSuccess: (data) => {
      setSessionBundle(data);
      setActiveRunIdOverride(data.run?.id || data.session.latest_run_id || '');
      setAutoAdvanceState(null);
      if (data.change_batch?.id) {
        setActiveChangeBatchId(data.change_batch.id);
      }
    },
    onError: (err: Error) => message.error(err.message),
  });

  const createSession = useMutation({
    mutationFn: async (values: { title?: string; raw_input: string }) => {
      if (!activeProjectId) throw new Error(zh.missingWorkspace);
      return apiClient.createRequirementSession(activeProjectId, values);
    },
    onSuccess: (data) => {
      message.success(zh.draftCreated);
      setSessionBundle(data);
      setActiveRunIdOverride('');
      setAutoAdvanceState(null);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const reviseSession = useMutation({
    mutationFn: async (payload: { title?: string; raw_input?: string }) => {
      const sid = sessionBundle?.session.id;
      if (!activeProjectId || !sid) throw new Error(zh.missingSession);
      return apiClient.reviseRequirementSession(activeProjectId, sid, payload);
    },
    onSuccess: (data) => {
      message.success(zh.draftRevised);
      setSessionBundle(data);
      setActiveRunIdOverride(data.run?.id || data.session.latest_run_id || '');
      setAutoAdvanceState(null);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const confirmSession = useMutation({
    mutationFn: async (payload: { note?: string }) => {
      const sid = sessionBundle?.session.id;
      if (!activeProjectId || !sid) throw new Error(zh.missingSession);
      return apiClient.confirmRequirementSession(activeProjectId, sid, payload);
    },
    onSuccess: (data) => {
      setSessionBundle(data);
      setActiveRunIdOverride(data.run?.id || data.session.latest_run_id || '');
      setAutoAdvanceState(null);
      if (data.change_batch?.id) {
        setActiveChangeBatchId(data.change_batch.id);
      }
      if (data.delivery_error) {
        message.warning(zh.confirmDeliveryFailed);
        return;
      }
      message.success(zh.confirmDeliveryStarted);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const autoAdvanceMutation = useMutation({
    mutationFn: async ({ runId, manual }: { runId: string; manual?: boolean }) => {
      const result = await apiClient.autoAdvancePipeline(runId);
      return { runId, manual: !!manual, result };
    },
    onSuccess: ({ runId, manual, result }) => {
      setAutoAdvanceState({ runId, result });
      if (result.executed && result.run) {
        setSessionBundle((previous) => updateBundleWithRun(previous, result.run!.id, result.run));
        setActiveRunIdOverride(result.run.id);
        invalidateRunQueries(runId);
        invalidateRunQueries(result.run.id);
        if (manual) {
          message.success(zh.autoAdvanced);
        }
        return;
      }
      if (manual) {
        const toast = result.blocking === 'approval_gate' || result.blocking === 'awaiting_human' ? message.warning : message.info;
        toast(result.reason || zh.noNeedAutoAdvance);
      }
    },
    onError: (err: Error, variables) => {
      attemptedAutoAdvanceRuns.current.delete(variables.runId);
      if (variables.manual) {
        message.error(err.message || zh.autoAdvanceFailed);
      }
    },
  });

  const updatePreviewSession = useMutation({
    mutationFn: async ({ sessionId, status }: { sessionId: string; status: 'accepted' | 'rejected' }) =>
      apiClient.updatePreviewSession(sessionId, { status }),
    onSuccess: (_updated, variables) => {
      const accepted = variables.status === 'accepted';
      message.success(accepted ? zh.previewAccepted : zh.previewRejected);
      invalidateRunQueries(latestRunId);
      setAutoAdvanceState(null);
      if (latestRunId) {
        attemptedAutoAdvanceRuns.current.add(latestRunId);
        autoAdvanceMutation.mutate({ runId: latestRunId, manual: false });
      }
    },
    onError: (err: Error) => message.error(err.message || zh.previewUpdateFailed),
  });

  const updateDeliveryAcceptance = useMutation({
    mutationFn: async ({ status }: { status: 'accepted' | 'revoked' }) => {
      if (!latestRunId) {
        throw new Error(zh.missingSession);
      }
      return apiClient.updateRunDeliveryAcceptance(latestRunId, { status });
    },
    onSuccess: (_updated, variables) => {
      message.success(variables.status === 'accepted' ? zh.finalAcceptanceRecorded : zh.finalAcceptanceReopened);
      invalidateRunQueries(latestRunId);
    },
    onError: (err: Error) => message.error(err.message || zh.finalAcceptanceUpdateFailed),
  });

  const docVersions = sessionBundle?.doc_versions ?? [];
  const summaryDoc = useMemo(() => pickLatestDoc(docVersions, 'summary'), [docVersions]);
  const prdDoc = useMemo(() => pickLatestDoc(docVersions, 'prd'), [docVersions]);
  const planDoc = useMemo(() => pickLatestDoc(docVersions, 'plan'), [docVersions]);
  const riskDoc = useMemo(() => pickLatestDoc(docVersions, 'risks'), [docVersions]);
  const run = runQuery.data ?? (sessionBundle?.run?.id === latestRunId ? sessionBundle.run : undefined);
  const previewSessions = previewSessionsQuery.data ?? [];
  const latestPreviewSession = findLatestPreviewSession(previewSessions);
  const pendingPreviewSession = previewSessions.find((item) => item.status === 'generated') ?? latestPreviewSession;
  const previewHref = buildPreviewHref(apiBase, completionQuery.data?.preview_url || latestPreviewSession?.preview_url);
  const approvalGates = approvalGatesQuery.data ?? [];
  const residualItems = residualItemsQuery.data ?? [];
  const deliveryAcceptance = deliveryAcceptanceQuery.data ?? null;
  const deliveryLedgerRuns = useMemo(
    () =>
      (changeBatchRunsQuery.data ?? [])
        .filter((item) => item.change_batch_id === latestChangeBatchId)
        .sort((left, right) => dayjs(left.created_at).valueOf() - dayjs(right.created_at).valueOf()),
    [changeBatchRunsQuery.data, latestChangeBatchId],
  );
  const deliveryLedgerDisplayRuns = useMemo(() => deliveryLedgerRuns.slice(-6), [deliveryLedgerRuns]);
  const deliveryLedgerSignalQueries = useQueries({
    queries: deliveryLedgerDisplayRuns.map((item) => ({
      queryKey: ['simple-ledger-run-signal', item.id],
      queryFn: async (): Promise<DeliveryLedgerRunSignal> => {
        const [events, previewSessions, approvalGates, residualItems] = await Promise.all([
          apiClient.listRunEvents(item.id),
          apiClient.listRunPreviewSessions(item.id),
          apiClient.listRunApprovalGates(item.id),
          apiClient.listRunResidualItems(item.id),
        ]);
        return buildDeliveryLedgerRunSignal({ events, previewSessions, approvalGates, residualItems });
      },
      enabled: !!latestChangeBatchId,
      staleTime: 30000,
    })),
  });
  const deliveryLedgerSignals = Object.fromEntries(
    deliveryLedgerDisplayRuns
      .map((item, index) => {
        const signal = deliveryLedgerSignalQueries[index]?.data;
        return signal ? [item.id, signal] : null;
      })
      .filter((item): item is [string, DeliveryLedgerRunSignal] => item !== null),
  );
  const deliveryLedgerLoading =
    changeBatchRunsQuery.isLoading || deliveryLedgerSignalQueries.some((query) => query.isLoading);
  const resultViewOptions = [
    { key: 'overview' as const, label: zh.resultCockpitOverview, description: zh.resultCockpitOverviewDesc },
    { key: 'autonomy' as const, label: zh.resultCockpitAutonomy, description: zh.resultCockpitAutonomyDesc },
    { key: 'history' as const, label: zh.resultCockpitHistory, description: zh.resultCockpitHistoryDesc },
  ];
  const activeResultView = resultViewOptions.find((item) => item.key === resultView) ?? resultViewOptions[0];
  const openApprovalGateCount = countOpenApprovalGates(approvalGates);
  const openResidualCount = countOpenResiduals(residualItems);
  const latestEvaluation = agentQuery.data?.latest_evaluation;
  const currentAutoAdvanceResult = autoAdvanceState?.runId === latestRunId ? autoAdvanceState.result : null;
  const autoModeLabel = projectQuery.data?.default_agent_mode === 'full_cycle' ? zh.fullCycleMode : zh.stepByStepMode;
  const canManualAutoAdvance = !!run && (run.status === 'failed' || run.status === 'completed');
  const needsApprovalHandling =
    openApprovalGateCount > 0 ||
    run?.status === 'awaiting_human' ||
    currentAutoAdvanceResult?.blocking === 'approval_gate' ||
    currentAutoAdvanceResult?.blocking === 'awaiting_human';

  useEffect(() => {
    if (!latestRunId) {
      return;
    }
    setResultView('overview');
  }, [latestRunId]);

  useEffect(() => {
    if (!run?.id) {
      return;
    }
    if (run.status !== 'failed' && run.status !== 'completed') {
      return;
    }
    if (attemptedAutoAdvanceRuns.current.has(run.id) || autoAdvanceMutation.isPending) {
      return;
    }
    attemptedAutoAdvanceRuns.current.add(run.id);
    autoAdvanceMutation.mutate({ runId: run.id, manual: false });
  }, [autoAdvanceMutation.isPending, run?.id, run?.status]);

  const statusTag = (status?: string) => {
    switch (status) {
      case 'confirmed':
        return <Tag color="green">{zh.sessionConfirmed}</Tag>;
      case 'awaiting_confirm':
        return <Tag color="orange">{zh.sessionAwaitingConfirm}</Tag>;
      default:
        return <Tag>{zh.sessionDraft}</Tag>;
    }
  };

  const deliveryAlert = useMemo(() => {
    if (!run) {
      return null;
    }
    if (currentAutoAdvanceResult && !currentAutoAdvanceResult.executed) {
      const type =
        currentAutoAdvanceResult.blocking === 'approval_gate' || currentAutoAdvanceResult.blocking === 'awaiting_human'
          ? 'warning'
          : currentAutoAdvanceResult.blocking === 'preview_review'
            ? 'info'
            : 'success';
      return {
        type,
        message:
          currentAutoAdvanceResult.blocking === 'approval_gate'
            ? zh.blockApproval
            : currentAutoAdvanceResult.blocking === 'awaiting_human'
              ? zh.blockAwaitingHuman
              : currentAutoAdvanceResult.blocking === 'preview_review'
                ? zh.blockPreviewReview
                : currentAutoAdvanceResult.next_command === 'complete_delivery'
                  ? zh.deliveryCompleted
                  : zh.noNeedAutoAdvance,
        description: withNextStep(currentAutoAdvanceResult.reason, currentAutoAdvanceResult.next_command),
      };
    }
    if (needsApprovalHandling) {
      return {
        type: 'warning',
        message: zh.needHumanAction,
        description: withNextStep(approvalGates[0]?.detail || latestEvaluation?.reason || zh.approvalFallback, latestEvaluation?.next_command),
      };
    }
    if (pendingPreviewSession?.status === 'generated') {
      return {
        type: 'info',
        message: zh.blockPreviewReview,
        description: withNextStep(latestEvaluation?.reason || zh.previewFallback, latestEvaluation?.next_command),
      };
    }
    if (run.status === 'failed') {
      return {
        type: 'warning',
        message: zh.failedDelivery,
        description: withNextStep(latestEvaluation?.reason || zh.failedDeliveryFallback, latestEvaluation?.next_command),
      };
    }
    if (completionQuery.data?.preview_url || latestPreviewSession?.preview_url) {
      return {
        type: 'success',
        message: zh.previewReady,
        description: zh.previewReadyDesc,
      };
    }
    return {
      type: 'info',
      message: zh.deliveryRunning,
      description: zh.deliveryRunningDesc,
    };
  }, [
    approvalGates,
    completionQuery.data?.preview_url,
    currentAutoAdvanceResult,
    latestEvaluation,
    latestPreviewSession?.preview_url,
    needsApprovalHandling,
    pendingPreviewSession?.status,
    run,
  ]);

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        {zh.pageTitle}
      </Typography.Title>

      {!activeProjectId ? (
        <Alert type="warning" showIcon title={zh.pickProject} />
      ) : (
        <Card title={zh.requestInputCard} style={{ borderRadius: 16 }}>
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            <Alert
              type="info"
              showIcon
              title={zh.requestInputTitle}
              description={zh.requestInputDesc.replace('{mode}', autoModeLabel)}
            />
            <Form
              layout="vertical"
              form={form}
              initialValues={{ raw_input: '' }}
              onFinish={(values) => createSession.mutate(values)}
            >
              <Form.Item label={zh.titleLabel} name="title">
                <Input placeholder={zh.titlePlaceholder} allowClear />
              </Form.Item>
              <Form.Item label={zh.inputLabel} name="raw_input" rules={[{ required: true, message: zh.inputRequired }]}>
                <Input.TextArea rows={4} placeholder={zh.inputPlaceholder} allowClear />
              </Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" loading={createSession.isPending} disabled={!activeProjectId}>
                  {zh.generateDraft}
                </Button>
                {latest ? (
                  <Button onClick={() => fetchSession.mutate(latest.id)} loading={fetchSession.isPending}>
                    {zh.refreshSession}
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
              <Statistic title={zh.createdAt} value={dayjs(latest.created_at).format('MM-DD HH:mm')} styles={{ content: { fontSize: 14 } }} />
              <Statistic title={zh.updatedAt} value={dayjs(latest.updated_at).format('MM-DD HH:mm')} styles={{ content: { fontSize: 14 } }} />
            </Space>
          }
          style={{ borderRadius: 16 }}
        >
          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <DocPreview title={zh.summaryDoc} content={summaryDoc?.content || latest.latest_summary} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title={zh.riskDoc} content={riskDoc?.content || latest.latest_risks} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title={zh.prdDoc} content={prdDoc?.content || latest.latest_prd} />
            </Col>
            <Col xs={24} md={12}>
              <DocPreview title={zh.planDoc} content={planDoc?.content || latest.latest_plan} />
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
              {zh.confirmAndStart}
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
              {zh.regenerateDraft}
            </Button>
          </Space>

          {confirmSession.isPending || reviseSession.isPending ? <Skeleton active style={{ marginTop: 12 }} /> : null}
          {sessionBundle?.confirmation ? (
            <Alert
              type="success"
              showIcon
              style={{ marginTop: 12 }}
              title={zh.confirmedAlert}
              description={dayjs(sessionBundle.confirmation.created_at).format('YYYY-MM-DD HH:mm:ss')}
            />
          ) : null}
          {sessionBundle?.delivery_error ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 12 }}
              title={zh.deliveryStartFailed}
              description={sessionBundle.delivery_error}
            />
          ) : null}
        </Card>
      ) : null}

      {latestRunId ? (
        <Card
          title={zh.deliveryResult}
          style={{ borderRadius: 16 }}
          extra={
            <Space>
              <Button data-testid="simple-delivery-open-pipeline" onClick={() => navigate('/pipeline')}>
                {zh.openPipeline}
              </Button>
              {previewHref ? (
                <Button type="primary" onClick={() => window.open(previewHref, '_blank', 'noopener,noreferrer')}>
                  {zh.openPreview}
                </Button>
              ) : null}
            </Space>
          }
        >
          {!run ? (
            <Skeleton active />
          ) : (
            <Space orientation="vertical" style={{ width: '100%' }} size="middle">
              <Space wrap>
                <Tag color={runStatusColor(run.status)}>{run.status}</Tag>
                <Tag>{run.stage}</Tag>
                {run.full_cycle ? <Tag color="cyan">full-cycle</Tag> : null}
                {run.step_by_step ? <Tag color="blue">step-by-step</Tag> : null}
                {latest?.latest_change_batch_id ? <Tag>{latest.latest_change_batch_id.slice(0, 8)}</Tag> : null}
                {latestEvaluation?.next_command ? <Tag color="purple">{latestEvaluation.next_command}</Tag> : null}
              </Space>
              <Typography.Text type="secondary">{run.prompt}</Typography.Text>
              <Progress percent={run.progress} strokeColor={{ from: '#0ea5e9', to: '#7c3aed' }} />
              {deliveryAlert ? (
                <Alert
                  data-testid="simple-delivery-status-alert"
                  type={deliveryAlert.type as 'success' | 'info' | 'warning' | 'error'}
                  showIcon
                  title={deliveryAlert.message}
                  description={deliveryAlert.description}
                />
              ) : null}
              <Row gutter={[16, 16]}>
                <Col xs={24} md={6}>
                  <Statistic title={zh.runId} value={run.id.slice(0, 8)} styles={{ content: { fontSize: 16 } }} />
                </Col>
                <Col xs={24} md={6}>
                  <Statistic title={zh.updatedAt} value={dayjs(run.updated_at).format('MM-DD HH:mm:ss')} styles={{ content: { fontSize: 16 } }} />
                </Col>
                <Col xs={24} md={6}>
                  <Statistic
                    title={zh.previewStatus}
                    value={previewStatusLabel(latestPreviewSession?.status || (completionQuery.data?.preview_url ? 'generated' : ''))}
                    styles={{ content: { fontSize: 16 } }}
                  />
                </Col>
                <Col xs={24} md={6}>
                  <Statistic title={zh.pendingItems} value={openResidualCount + openApprovalGateCount} styles={{ content: { fontSize: 16 } }} />
                </Col>
              </Row>

              <Space wrap>
                {canManualAutoAdvance ? (
                  <Button
                    data-testid="simple-delivery-auto-advance"
                    onClick={() => autoAdvanceMutation.mutate({ runId: run.id, manual: true })}
                    loading={autoAdvanceMutation.isPending}
                  >
                    {zh.continueAutoAdvance}
                  </Button>
                ) : null}
                {pendingPreviewSession?.status === 'generated' ? (
                  <>
                    <Button
                      data-testid="simple-delivery-preview-accept"
                      type="primary"
                      onClick={() => updatePreviewSession.mutate({ sessionId: pendingPreviewSession.id, status: 'accepted' })}
                      loading={updatePreviewSession.isPending}
                    >
                      {zh.approvePreview}
                    </Button>
                    <Button
                      data-testid="simple-delivery-preview-reject"
                      danger
                      onClick={() => updatePreviewSession.mutate({ sessionId: pendingPreviewSession.id, status: 'rejected' })}
                      loading={updatePreviewSession.isPending}
                    >
                      {zh.rejectPreview}
                    </Button>
                  </>
                ) : null}
                {needsApprovalHandling ? <Button onClick={() => navigate('/pipeline')}>{zh.handleApproval}</Button> : null}
              </Space>

              <Space wrap>
                {latestPreviewSession ? (
                  <Tag color={previewStatusColor(latestPreviewSession.status)}>
                    {zh.previewTagPrefix} {previewStatusLabel(latestPreviewSession.status)}
                  </Tag>
                ) : null}
                {openApprovalGateCount > 0 ? <Tag color="orange">{`${zh.approvalTagPrefix} ${openApprovalGateCount}`}</Tag> : null}
                {openResidualCount > 0 ? <Tag color="gold">{`${zh.residualTagPrefix} ${openResidualCount}`}</Tag> : null}
              </Space>

              <div
                style={{
                  border: '1px solid #e5e7eb',
                  borderRadius: 18,
                  padding: 16,
                  background: 'linear-gradient(180deg, #f8fafc 0%, #ffffff 100%)',
                }}
              >
                <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                  <Space wrap align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                    <Space wrap>
                      <Typography.Text strong>{zh.resultCockpitTitle}</Typography.Text>
                      {resultViewOptions.map((item) => (
                        <Button
                          key={item.key}
                          size="small"
                          type={resultView === item.key ? 'primary' : 'default'}
                          data-testid={`simple-delivery-view-${item.key}`}
                          onClick={() => setResultView(item.key)}
                        >
                          {item.label}
                        </Button>
                      ))}
                    </Space>
                    <Tag color="blue">{activeResultView.label}</Tag>
                  </Space>
                  <Typography.Text type="secondary">{activeResultView.description}</Typography.Text>

                  {resultView === 'overview' ? (
                    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                      <DeliveryProcessPreviewCard
                        completion={completionQuery.data}
                        apiBase={apiBase}
                        loading={completionQuery.isLoading}
                      />

                      <DeliveryHandoffCard
                        run={run}
                        completion={completionQuery.data}
                        events={eventsQuery.data ?? []}
                        previewSessions={previewSessions}
                        approvalGates={approvalGates}
                        residualItems={residualItems}
                        deliveryAcceptance={deliveryAcceptance}
                        onAcceptFinalAcceptance={() => updateDeliveryAcceptance.mutate({ status: 'accepted' })}
                        onRevokeFinalAcceptance={() => updateDeliveryAcceptance.mutate({ status: 'revoked' })}
                        submittingFinalAcceptance={updateDeliveryAcceptance.isPending}
                        apiBase={apiBase}
                        loading={
                          completionQuery.isLoading ||
                          eventsQuery.isLoading ||
                          previewSessionsQuery.isLoading ||
                          deliveryAcceptanceQuery.isLoading ||
                          approvalGatesQuery.isLoading ||
                          residualItemsQuery.isLoading
                        }
                      />
                    </Space>
                  ) : null}

                  {resultView === 'autonomy' ? (
                    <AutonomyActivityCard
                      events={eventsQuery.data ?? []}
                      loading={eventsQuery.isLoading}
                    />
                  ) : null}

                  {resultView === 'history' ? (
                    <DeliveryLedgerCard
                      batchId={latestChangeBatchId}
                      batchTitle={sessionBundle?.change_batch?.title || latest?.title}
                      mode={sessionBundle?.change_batch?.mode || (run?.full_cycle ? 'full_cycle' : run?.step_by_step ? 'step_by_step' : '')}
                      runs={deliveryLedgerDisplayRuns}
                      totalAttempts={deliveryLedgerRuns.length}
                      currentRunId={latestRunId}
                      runSignals={deliveryLedgerSignals}
                      loading={deliveryLedgerLoading}
                    />
                  ) : null}
                </Space>
              </div>
            </Space>
          )}
        </Card>
      ) : null}

      <Card title={zh.howToUse} size="small" style={{ borderRadius: 12 }}>
        <Space orientation="vertical">
          <Typography.Text>{zh.howTo1}</Typography.Text>
          <Typography.Text>{zh.howTo2}</Typography.Text>
          <Typography.Text>{zh.howTo3}</Typography.Text>
          <Typography.Text>{zh.howTo4}</Typography.Text>
        </Space>
      </Card>
    </Space>
  );
}
