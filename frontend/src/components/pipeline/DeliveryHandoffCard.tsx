import { Alert, Button, Card, Divider, Empty, Space, Tag, Typography } from 'antd';
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

type LocalPreviewGuide = {
  outputDir: string;
  serveCommand: string;
  localUrl: string;
  previewFilePath?: string;
  note: string;
};

type HandoffSummary = {
  overall: 'ready' | 'blocked' | 'in_progress';
  title: string;
  description: string;
  checks: HandoffCheck[];
  packageArtifacts: PipelineArtifact[];
  previewHref: string;
  acceptanceTitle: string;
  acceptanceNote: string;
  acceptanceSteps: string[];
  localPreview?: LocalPreviewGuide;
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
    <Card title="Delivery Handoff / Readiness" style={{ borderRadius: 20 }} loading={loading} data-testid="delivery-handoff-card">
      {!run ? (
        <Empty description="Run data is not available yet" />
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
          <Typography.Text strong>Artifacts</Typography.Text>
          <Tag color="blue">{summary.packageArtifacts.length} files</Tag>
        </Space>
        {!summary.packageArtifacts.length ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            No preview or report artifacts are ready yet.
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
                Open preview
              </Button>
            ) : null}
          </Space>
        )}
      </div>

      <Divider style={{ margin: '4px 0' }} />

      <div data-testid="delivery-handoff-acceptance">
        <Space wrap style={{ marginBottom: 8 }}>
          <Typography.Text strong>{summary.acceptanceTitle}</Typography.Text>
          <Tag color={overallTagColor(summary.overall)}>{overallTagLabel(summary.overall)}</Tag>
        </Space>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
          {summary.acceptanceNote}
        </Typography.Paragraph>
        <Space orientation="vertical" size={4} style={{ width: '100%' }}>
          {summary.acceptanceSteps.map((step, index) => (
            <Typography.Paragraph key={`${index + 1}-${step}`} style={{ marginBottom: 0 }}>
              {index + 1}. {step}
            </Typography.Paragraph>
          ))}
        </Space>
      </div>

      <div data-testid="delivery-handoff-local-preview">
        <Space wrap style={{ marginBottom: 8 }}>
          <Typography.Text strong>Local preview / handoff</Typography.Text>
          {summary.localPreview ? <Tag color="blue">Output ready</Tag> : <Tag>Pending</Tag>}
        </Space>
        {summary.localPreview ? (
          <Space orientation="vertical" size={8} style={{ width: '100%' }}>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              {summary.localPreview.note}
            </Typography.Paragraph>
            <InfoRow label="Output dir" value={summary.localPreview.outputDir} />
            <InfoRow label="Serve command" value={summary.localPreview.serveCommand} />
            <InfoRow label="Local URL" value={summary.localPreview.localUrl} />
            {summary.localPreview.previewFilePath ? <InfoRow label="Preview file" value={summary.localPreview.previewFilePath} /> : null}
          </Space>
        ) : (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            Local preview guidance will appear once the run exposes an output directory.
          </Typography.Paragraph>
        )}
      </div>
    </Space>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div
      style={{
        border: '1px solid #e5e7eb',
        borderRadius: 12,
        padding: 10,
        background: '#fff',
      }}
    >
      <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>
        {label}
      </Typography.Text>
      <Typography.Text code>{value}</Typography.Text>
    </div>
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
  const previewArtifact = pickPreviewArtifact(completion?.artifacts ?? []);
  const localPreview = buildLocalPreviewGuide(completion, previewArtifact);

  let overall: HandoffSummary['overall'] = 'in_progress';
  let title = 'Release handoff is still in progress';
  let description = 'Some checks are still running or waiting for a human decision before the handoff package is fully ready.';

  if (checks.some((check) => check.status === 'failed')) {
    const blocking = checks.find((check) => check.status === 'failed');
    overall = 'blocked';
    title = 'Release handoff is blocked';
    description = blocking?.note || 'Resolve the blocking items before continuing to handoff.';
  } else if (
    run.status === 'completed' &&
    previewCheck.status === 'completed' &&
    qualityCheck.status === 'completed' &&
    packageCheck.status === 'completed'
  ) {
    overall = 'ready';
    title = 'Release handoff is ready';
    description = 'Preview, quality, and package checks are complete. The run is ready for final review and pre-release handoff.';
  }

  const acceptance = buildAcceptanceGuidance(overall, previewHref, localPreview?.localUrl);

  return {
    overall,
    title,
    description,
    checks,
    packageArtifacts,
    previewHref,
    acceptanceTitle: acceptance.title,
    acceptanceNote: acceptance.note,
    acceptanceSteps: acceptance.steps,
    localPreview,
  };
}

function buildAcceptanceGuidance(
  overall: HandoffSummary['overall'],
  previewHref: string,
  localPreviewUrl?: string,
): { title: string; note: string; steps: string[] } {
  if (overall === 'ready') {
    return {
      title: 'Final acceptance',
      note: 'The run looks ready for final sign-off. Review the generated preview, verify the handoff package, and treat this build as the current release candidate.',
      steps: [
        previewHref
          ? 'Review the latest final preview and confirm the page matches the approved requirement.'
          : 'Review the latest packaged output before sign-off.',
        'Check the quality, execution, and other handoff artifacts for anything that still needs clarification.',
        localPreviewUrl
          ? `If you want a local browser pass, serve the output directory and open ${localPreviewUrl}.`
          : 'If you want a local browser pass, serve the output directory before sign-off.',
      ],
    };
  }

  if (overall === 'blocked') {
    return {
      title: 'Final acceptance',
      note: 'Resolve the blocked checks before requesting final sign-off.',
      steps: [
        'Clear open approvals and residual follow-up items first.',
        previewHref
          ? 'Re-review or regenerate the preview if the current output is not yet acceptable.'
          : 'Wait until a reviewable preview is generated.',
        'After the blocking items are cleared, re-check the handoff package and request sign-off again.',
      ],
    };
  }

  return {
    title: 'Final acceptance',
    note: 'The run is still preparing the final handoff package.',
    steps: [
      'Wait for preview, quality, and package checks to finish converging.',
      'Use the generated preview and process documents as they appear to monitor progress.',
      localPreviewUrl
        ? `Once you want a local browser check, serve the output directory and open ${localPreviewUrl}.`
        : 'Once a preview artifact is generated, a local browser check path will appear here.',
    ],
  };
}

function buildLocalPreviewGuide(completion: PipelineCompletion | undefined, previewArtifact?: PipelineArtifact): LocalPreviewGuide | undefined {
  if (!completion?.output_dir) {
    return undefined;
  }

  const previewFilePath = previewArtifact?.path;
  const relativePath = previewFilePath ? normalizeLocalPreviewPath(completion.output_dir, previewFilePath) : '';

  return {
    outputDir: completion.output_dir,
    serveCommand: `python -m http.server 4173 --directory "${completion.output_dir}"`,
    localUrl: relativePath ? `http://127.0.0.1:4173/${relativePath}` : 'http://127.0.0.1:4173/',
    previewFilePath,
    note: relativePath
      ? 'Run the command below, then open the local URL to inspect the packaged preview from the output directory.'
      : 'Run the command below to browse the packaged output directory locally. The preview URL will become more specific once the final HTML artifact is available.',
  };
}

function normalizeLocalPreviewPath(outputDir: string, previewPath: string) {
  const normalizedOutput = trimTrailingSlashes(outputDir.replaceAll('\\', '/'));
  let normalizedPreview = previewPath.replaceAll('\\', '/');

  if (normalizedPreview.startsWith('./')) {
    normalizedPreview = normalizedPreview.slice(2);
  }
  if (normalizedPreview.toLowerCase().startsWith(`${normalizedOutput.toLowerCase()}/`)) {
    normalizedPreview = normalizedPreview.slice(normalizedOutput.length + 1);
  }
  if (normalizedPreview.toLowerCase().startsWith('output/')) {
    normalizedPreview = normalizedPreview.slice('output/'.length);
  }
  while (normalizedPreview.startsWith('/')) {
    normalizedPreview = normalizedPreview.slice(1);
  }
  return normalizedPreview;
}

function trimTrailingSlashes(value: string) {
  let trimmed = value;
  while (trimmed.endsWith('/')) {
    trimmed = trimmed.slice(0, -1);
  }
  return trimmed;
}

function buildPreviewCheck(run: PipelineRun, completion: PipelineCompletion | undefined, previewSessions: PreviewSession[]): HandoffCheck {
  const latest = [...previewSessions].sort((left, right) => new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime())[0];
  if (latest?.status === 'accepted') {
    return {
      key: 'preview',
      title: 'Preview',
      status: 'completed',
      note: latest.reviewer_note || 'Preview was accepted and is ready for handoff.',
    };
  }
  if (latest?.status === 'rejected') {
    return {
      key: 'preview',
      title: 'Preview',
      status: 'failed',
      note: latest.reviewer_note || 'Preview was rejected and must be regenerated before handoff.',
    };
  }
  if (latest?.status === 'generated' || completion?.preview_url) {
    return {
      key: 'preview',
      title: 'Preview',
      status: 'in_progress',
      note: latest?.reviewer_note || 'Preview is generated and waiting for reviewer sign-off.',
    };
  }
  if (run.status === 'completed') {
    return {
      key: 'preview',
      title: 'Preview',
      status: 'missing',
      note: 'The run finished, but no preview artifact was found for final review.',
    };
  }
  return {
    key: 'preview',
    title: 'Preview',
    status: 'in_progress',
    note: 'Preview has not been generated yet.',
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
        title: 'Quality',
        status: 'completed',
        note: item.message,
      };
    }
    if (item.status === 'failed' || lowerMessage.includes('still failing') || lowerMessage.includes('not passed')) {
      return {
        key: 'quality',
        title: 'Quality',
        status: 'failed',
        note: item.message,
      };
    }
  }

  if (hasQualityArtifact(completion?.artifacts || [])) {
    return {
      key: 'quality',
      title: 'Quality',
      status: run.status === 'completed' ? 'completed' : 'in_progress',
      note:
        run.status === 'completed'
          ? 'Quality artifacts are available and the run completed successfully.'
          : 'Quality artifacts are being generated for this run.',
    };
  }

  if (run.status === 'failed') {
    return {
      key: 'quality',
      title: 'Quality',
      status: 'failed',
      note: 'The run failed before the quality gate could pass.',
    };
  }

  return {
    key: 'quality',
    title: 'Quality',
    status: 'in_progress',
    note: 'Quality validation is still running.',
  };
}

function buildApprovalCheck(approvalGates: ApprovalGate[]): HandoffCheck {
  const openCount = approvalGates.filter((item) => item.status === 'open').length;
  if (openCount > 0) {
    return {
      key: 'approval',
      title: 'Approvals',
      status: 'failed',
      note: `${openCount} approval gate(s) still need human review.`,
    };
  }
  return {
    key: 'approval',
    title: 'Approvals',
    status: 'completed',
    note: 'No open approval gates remain.',
  };
}

function buildResidualCheck(residualItems: ResidualItem[]): HandoffCheck {
  const openCount = residualItems.filter((item) => item.status === 'open').length;
  if (openCount > 0) {
    return {
      key: 'residual',
      title: 'Residuals',
      status: 'failed',
      note: `${openCount} residual item(s) still need follow-up.`,
    };
  }
  return {
    key: 'residual',
    title: 'Residuals',
    status: 'completed',
    note: 'No open residual items remain.',
  };
}

function buildPackageCheck(run: PipelineRun, packageArtifacts: PipelineArtifact[]): HandoffCheck {
  if (packageArtifacts.length > 0) {
    return {
      key: 'package',
      title: 'Package',
      status: 'completed',
      note: `${packageArtifacts.length} handoff artifact(s) are ready to review.`,
    };
  }
  if (run.status === 'completed') {
    return {
      key: 'package',
      title: 'Package',
      status: 'missing',
      note: 'The run completed, but no handoff artifacts were collected.',
    };
  }
  return {
    key: 'package',
    title: 'Package',
    status: 'in_progress',
    note: 'The handoff package is still being assembled.',
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

function pickPreviewArtifact(artifacts: PipelineArtifact[]) {
  return artifacts.find((artifact) => isPreviewArtifact(artifact));
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

function overallTagColor(status: HandoffSummary['overall']) {
  switch (status) {
    case 'ready':
      return 'green';
    case 'blocked':
      return 'red';
    default:
      return 'blue';
  }
}

function overallTagLabel(status: HandoffSummary['overall']) {
  switch (status) {
    case 'ready':
      return 'Ready for sign-off';
    case 'blocked':
      return 'Blocked';
    default:
      return 'Preparing';
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
      return 'Passed';
    case 'failed':
      return 'Blocked';
    case 'missing':
      return 'Missing';
    default:
      return 'Running';
  }
}
