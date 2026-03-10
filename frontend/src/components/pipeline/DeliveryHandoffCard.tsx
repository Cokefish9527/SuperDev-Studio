import { Alert, Button, Card, Empty, Space, Tag, Typography } from 'antd';
import type {
  ApprovalGate,
  PipelineArtifact,
  PipelineCompletion,
  PipelineRun,
  PreviewSession,
  ResidualItem,
  RunEvent,
} from '../../types';

type CheckStatus = 'completed' | 'failed' | 'in_progress' | 'missing';

type HandoffCheck = {
  key: string;
  title: string;
  status: CheckStatus;
  note: string;
};

type HandoffSummary = {
  overall: 'ready' | 'blocked' | 'in_progress';
  title: string;
  description: string;
  checks: HandoffCheck[];
  packageArtifacts: PipelineArtifact[];
  previewHref: string;
};

type Props = {
  run?: PipelineRun;
  completion?: PipelineCompletion;
  events: RunEvent[];
  previewSessions: PreviewSession[];
  approvalGates: ApprovalGate[];
  residualItems: ResidualItem[];
  apiBase: string;
  loading?: boolean;
};

export default function DeliveryHandoffCard({
  run,
  completion,
  events,
  previewSessions,
  approvalGates,
  residualItems,
  apiBase,
  loading,
}: Props) {
  return (
    <Card title="???? / ?????" style={{ borderRadius: 20 }} loading={loading} data-testid="delivery-handoff-card">
      {!run ? (
        <Empty description="????????????" />
      ) : (
        <HandoffBody
          summary={buildHandoffSummary({ run, completion, events, previewSessions, approvalGates, residualItems, apiBase })}
          apiBase={apiBase}
        />
      )}
    </Card>
  );
}

function HandoffBody({ summary, apiBase }: { summary: HandoffSummary; apiBase: string }) {
  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Alert
        data-testid="delivery-handoff-alert"
        showIcon
        type={overallAlertType(summary.overall)}
        title={summary.title}
        description={summary.description}
      />

      <Space wrap size={[12, 12]} style={{ width: '100%' }}>
        {summary.checks.map((check) => (
          <div
            key={check.key}
            style={{
              flex: '1 1 220px',
              minWidth: 220,
              border: '1px solid #e5e7eb',
              borderRadius: 14,
              padding: 12,
              background: '#fff',
            }}
          >
            <Space orientation="vertical" size={6} style={{ width: '100%' }}>
              <Space wrap>
                <Tag color={checkStatusColor(check.status)}>{checkStatusLabel(check.status)}</Tag>
                <Typography.Text strong>{check.title}</Typography.Text>
              </Space>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                {check.note}
              </Typography.Paragraph>
            </Space>
          </div>
        ))}
      </Space>

      <div>
        <Space wrap style={{ marginBottom: 8 }}>
          <Typography.Text strong>???</Typography.Text>
          <Tag color="blue">{summary.packageArtifacts.length} ?</Tag>
        </Space>
        {!summary.packageArtifacts.length ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            ????????????????????????????
          </Typography.Paragraph>
        ) : (
          <Space wrap>
            {summary.packageArtifacts.map((artifact) => {
              const href = buildArtifactHref(apiBase, artifact.preview_url);
              return (
                <Button
                  key={artifact.path}
                  size="small"
                  onClick={() => href && window.open(href, '_blank', 'noopener,noreferrer')}
                  disabled={!href}
                >
                  {artifact.name}
                </Button>
              );
            })}
            {summary.previewHref ? (
              <Button size="small" type="primary" onClick={() => window.open(summary.previewHref, '_blank', 'noopener,noreferrer')}>
                ??????
              </Button>
            ) : null}
          </Space>
        )}
      </div>
    </Space>
  );
}

function buildHandoffSummary({
  run,
  completion,
  events,
  previewSessions,
  approvalGates,
  residualItems,
  apiBase,
}: {
  run: PipelineRun;
  completion?: PipelineCompletion;
  events: RunEvent[];
  previewSessions: PreviewSession[];
  approvalGates: ApprovalGate[];
  residualItems: ResidualItem[];
  apiBase: string;
}): HandoffSummary {
  const previewCheck = buildPreviewCheck(run, completion, previewSessions);
  const qualityCheck = buildQualityCheck(run, completion, events);
  const approvalCheck = buildApprovalCheck(approvalGates);
  const residualCheck = buildResidualCheck(residualItems);
  const packageArtifacts = pickPackageArtifacts(completion);
  const packageCheck = buildPackageCheck(run, packageArtifacts);
  const checks = [previewCheck, qualityCheck, approvalCheck, residualCheck, packageCheck];
  const previewHref = buildArtifactHref(apiBase, completion?.preview_url);

  if (checks.some((check) => check.status === 'failed')) {
    const blocking = checks.find((check) => check.status === 'failed');
    return {
      overall: 'blocked',
      title: '???????????',
      description: blocking?.note || '?????????????????',
      checks,
      packageArtifacts,
      previewHref,
    };
  }

  if (run.status === 'completed' && previewCheck.status === 'completed' && qualityCheck.status === 'completed' && packageCheck.status === 'completed') {
    return {
      overall: 'ready',
      title: '??????????',
      description: '??????????????????????????????????',
      checks,
      packageArtifacts,
      previewHref,
    };
  }

  return {
    overall: 'in_progress',
    title: '????????????',
    description: '???????????????????????????????????????',
    checks,
    packageArtifacts,
    previewHref,
  };
}

function buildPreviewCheck(run: PipelineRun, completion: PipelineCompletion | undefined, previewSessions: PreviewSession[]): HandoffCheck {
  const latest = [...previewSessions].sort((left, right) => new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime())[0];
  if (latest?.status === 'accepted') {
    return {
      key: 'preview',
      title: '????',
      status: 'completed',
      note: latest.reviewer_note || '?????????????',
    };
  }
  if (latest?.status === 'rejected') {
    return {
      key: 'preview',
      title: '????',
      status: 'failed',
      note: latest.reviewer_note || '?????????????????????',
    };
  }
  if (latest?.status === 'generated' || completion?.preview_url) {
    return {
      key: 'preview',
      title: '????',
      status: 'in_progress',
      note: latest?.reviewer_note || '???????????????????',
    };
  }
  if (run.status === 'completed') {
    return {
      key: 'preview',
      title: '????',
      status: 'missing',
      note: '??????????????????????????',
    };
  }
  return {
    key: 'preview',
    title: '????',
    status: 'in_progress',
    note: '?????????????',
  };
}

function buildQualityCheck(run: PipelineRun, completion: PipelineCompletion | undefined, events: RunEvent[]): HandoffCheck {
  const qualityEvents = [...events].filter((item) => item.stage.toLowerCase().includes('quality'));
  for (let index = qualityEvents.length - 1; index >= 0; index -= 1) {
    const item = qualityEvents[index];
    const lowerMessage = item.message.toLowerCase();
    if (item.status === 'completed' || lowerMessage.includes('quality gate passed')) {
      return {
        key: 'quality',
        title: '????',
        status: 'completed',
        note: item.message,
      };
    }
    if (item.status === 'failed' || lowerMessage.includes('still failing') || lowerMessage.includes('not passed')) {
      return {
        key: 'quality',
        title: '????',
        status: 'failed',
        note: item.message,
      };
    }
  }

  if (hasQualityArtifact(completion?.artifacts || [])) {
    return {
      key: 'quality',
      title: '????',
      status: run.status === 'completed' ? 'completed' : 'in_progress',
      note: run.status === 'completed' ? '????????????????????' : '?????????????????????',
    };
  }

  if (run.status === 'failed') {
    return {
      key: 'quality',
      title: '????',
      status: 'failed',
      note: '?????????????????????',
    };
  }

  return {
    key: 'quality',
    title: '????',
    status: 'in_progress',
    note: '?????????????',
  };
}

function buildApprovalCheck(approvalGates: ApprovalGate[]): HandoffCheck {
  const openCount = approvalGates.filter((item) => item.status === 'open').length;
  if (openCount > 0) {
    return {
      key: 'approval',
      title: '?????',
      status: 'failed',
      note: `?? ${openCount} ?????????????`,
    };
  }
  return {
    key: 'approval',
    title: '?????',
    status: 'completed',
    note: '???????????????',
  };
}

function buildResidualCheck(residualItems: ResidualItem[]): HandoffCheck {
  const openCount = residualItems.filter((item) => item.status === 'open').length;
  if (openCount > 0) {
    return {
      key: 'residual',
      title: '????',
      status: 'failed',
      note: `?? ${openCount} ??????????????????`,
    };
  }
  return {
    key: 'residual',
    title: '????',
    status: 'completed',
    note: '?????????? open residual items?',
  };
}

function buildPackageCheck(run: PipelineRun, packageArtifacts: PipelineArtifact[]): HandoffCheck {
  if (packageArtifacts.length > 0) {
    return {
      key: 'package',
      title: '???',
      status: 'completed',
      note: `??? ${packageArtifacts.length} ?????????`,
    };
  }
  if (run.status === 'completed') {
    return {
      key: 'package',
      title: '???',
      status: 'missing',
      note: '??????????????????????',
    };
  }
  return {
    key: 'package',
    title: '???',
    status: 'in_progress',
    note: '??????????????',
  };
}

function pickPackageArtifacts(completion?: PipelineCompletion): PipelineArtifact[] {
  if (!completion) {
    return [];
  }
  const matchers = [isPreviewArtifact, isQualityArtifact, isRedteamArtifact, isExecutionArtifact];
  const picked: PipelineArtifact[] = [];
  const seen = new Set<string>();
  for (const matcher of matchers) {
    const artifact = completion.artifacts.find((item) => matcher(item) && !seen.has(item.path));
    if (artifact) {
      picked.push(artifact);
      seen.add(artifact.path);
    }
  }
  if (!picked.length) {
    return completion.artifacts.slice(0, 4);
  }
  return picked;
}

function isPreviewArtifact(artifact: PipelineArtifact) {
  const lowerPath = artifact.path.toLowerCase();
  return artifact.preview_type === 'html' || lowerPath.endsWith('preview.html') || lowerPath.endsWith('frontend/index.html');
}

function isQualityArtifact(artifact: PipelineArtifact) {
  const lower = `${artifact.name} ${artifact.path}`.toLowerCase();
  return lower.includes('quality-gate');
}

function isRedteamArtifact(artifact: PipelineArtifact) {
  const lower = `${artifact.name} ${artifact.path}`.toLowerCase();
  return lower.includes('redteam');
}

function isExecutionArtifact(artifact: PipelineArtifact) {
  const lower = `${artifact.name} ${artifact.path}`.toLowerCase();
  return lower.includes('task-execution') || lower.includes('execution-report') || lower.includes('execution-plan');
}

function hasQualityArtifact(artifacts: PipelineArtifact[]) {
  return artifacts.some((artifact) => isQualityArtifact(artifact));
}

function buildArtifactHref(apiBase: string, previewUrl?: string) {
  if (!previewUrl) {
    return '';
  }
  if (/^https?:\/\//.test(previewUrl)) {
    return previewUrl;
  }
  return `${apiBase}${previewUrl}`;
}

function overallAlertType(status: HandoffSummary['overall']) {
  switch (status) {
    case 'ready':
      return 'success';
    case 'blocked':
      return 'warning';
    default:
      return 'info';
  }
}

function checkStatusColor(status: CheckStatus) {
  switch (status) {
    case 'completed':
      return 'green';
    case 'failed':
      return 'red';
    case 'missing':
      return 'orange';
    default:
      return 'blue';
  }
}

function checkStatusLabel(status: CheckStatus) {
  switch (status) {
    case 'completed':
      return '??';
    case 'failed':
      return '??';
    case 'missing':
      return '??';
    default:
      return '???';
  }
}
