import { Alert, Card, Col, Empty, Row, Space, Tag, Timeline, Typography } from 'antd';
import dayjs from 'dayjs';
import type { PipelineRun } from '../../types';
import { stageLabel } from './presentation';

export type DeliveryLedgerRunSignal = {
  preview: 'accepted' | 'rejected' | 'pending' | 'missing';
  quality: 'passed' | 'failed' | 'pending';
  openApprovals: number;
  openResiduals: number;
};

type Props = {
  batchId?: string;
  batchTitle?: string;
  mode?: string;
  runs: PipelineRun[];
  currentRunId?: string;
  loading?: boolean;
  totalAttempts?: number;
  runSignals?: Record<string, DeliveryLedgerRunSignal>;
};

type LedgerRun = PipelineRun & {
  attemptNumber: number;
};

export default function DeliveryLedgerCard({
  batchId,
  batchTitle,
  mode,
  runs,
  currentRunId,
  loading,
  totalAttempts,
  runSignals,
}: Props) {
  const orderedRuns = [...runs]
    .sort((left, right) => dayjs(left.created_at).valueOf() - dayjs(right.created_at).valueOf())
    .map((run, index) => ({ ...run, attemptNumber: index + 1 }));

  const latestRun = orderedRuns[orderedRuns.length - 1];
  const displayRuns = [...orderedRuns].reverse();
  const attemptTotal = totalAttempts ?? orderedRuns.length;
  const completedCount = orderedRuns.filter((run) => run.status === 'completed').length;
  const failedCount = orderedRuns.filter((run) => run.status === 'failed').length;
  const activeCount = orderedRuns.filter((run) => run.status !== 'completed' && run.status !== 'failed').length;

  return (
    <Card title="Delivery ledger" style={{ borderRadius: 16 }} loading={loading} data-testid="simple-delivery-ledger-card">
      {!displayRuns.length ? (
        <Empty description="No delivery attempts have been recorded for this change batch yet" />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Alert
            data-testid="simple-delivery-ledger-summary"
            showIcon
            type={latestRun?.status === 'completed' ? 'success' : latestRun?.status === 'failed' ? 'warning' : 'info'}
            title={summaryTitle(batchTitle, attemptTotal)}
            description={summaryDescription({ batchId, mode, latestRun, completedCount, failedCount, activeCount })}
          />

          <Row gutter={[12, 12]}>
            <Col xs={24} md={8}>
              <MetricTile label="Attempts" value={String(attemptTotal)} note="Total autonomous delivery runs in this change batch" />
            </Col>
            <Col xs={24} md={8}>
              <MetricTile label="Completed" value={String(completedCount)} note="Displayed runs that reached a completed delivery state" />
            </Col>
            <Col xs={24} md={8}>
              <MetricTile
                label="Latest status"
                value={statusLabel(latestRun?.status)}
                note={latestRun ? `Latest run ${shortId(latestRun.id)} is the current batch head` : 'No runs recorded yet'}
              />
            </Col>
          </Row>

          <Space align="center" wrap>
            <Typography.Text strong>Recent attempt history</Typography.Text>
            {attemptTotal > displayRuns.length ? (
              <Typography.Text type="secondary">Showing latest {displayRuns.length} of {attemptTotal} attempts</Typography.Text>
            ) : null}
          </Space>
          <div style={{ maxHeight: 360, overflowY: 'auto', paddingRight: 8 }}>
            <Timeline
              items={displayRuns.map((run) => {
                const signal = runSignals?.[run.id];
                return {
                  color: runStatusColor(run.status),
                  content: (
                    <Space orientation="vertical" size={4} style={{ width: '100%' }} data-testid={`delivery-ledger-run-${run.id}`}>
                      <Space wrap>
                        <Tag color="blue">Attempt {run.attemptNumber}</Tag>
                        <Tag color={runStatusColor(run.status)}>{run.status}</Tag>
                        <Tag>{stageLabel(run.stage)}</Tag>
                        {run.id === currentRunId ? <Tag color="processing">current</Tag> : null}
                        {run.id === latestRun?.id ? <Tag color="cyan">latest</Tag> : null}
                        {run.retry_of ? <Tag color="purple">retry</Tag> : null}
                      </Space>
                      <Typography.Text>{run.prompt}</Typography.Text>
                      {signal ? (
                        <Space wrap>
                          <Tag color={previewSignalColor(signal.preview)}>{previewSignalLabel(signal.preview)}</Tag>
                          <Tag color={qualitySignalColor(signal.quality)}>{qualitySignalLabel(signal.quality)}</Tag>
                          {signal.openApprovals > 0 ? <Tag color="gold">Approvals {signal.openApprovals}</Tag> : null}
                          {signal.openResiduals > 0 ? <Tag color="volcano">Residuals {signal.openResiduals}</Tag> : null}
                        </Space>
                      ) : null}
                      {run.retry_of ? (
                        <Typography.Text type="secondary">Retried from {shortId(run.retry_of)}</Typography.Text>
                      ) : null}
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        Created {dayjs(run.created_at).format('YYYY-MM-DD HH:mm:ss')} | Updated {dayjs(run.updated_at).format('YYYY-MM-DD HH:mm:ss')}
                      </Typography.Text>
                    </Space>
                  ),
                };
              })}
            />
          </div>
        </Space>
      )}
    </Card>
  );
}

function MetricTile({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <div
      style={{
        border: '1px solid #e5e7eb',
        borderRadius: 14,
        padding: 12,
        minHeight: 108,
        background: '#fff',
      }}
    >
      <Typography.Text type="secondary">{label}</Typography.Text>
      <Typography.Title level={4} style={{ margin: '8px 0 6px' }}>
        {value}
      </Typography.Title>
      <Typography.Text type="secondary">{note}</Typography.Text>
    </div>
  );
}

function summaryTitle(batchTitle: string | undefined, runCount: number) {
  if (batchTitle) {
    return `${batchTitle}: ${runCount} delivery attempt(s)`;
  }
  return `${runCount} delivery attempt(s) recorded`;
}

function summaryDescription({
  batchId,
  mode,
  latestRun,
  completedCount,
  failedCount,
  activeCount,
}: {
  batchId?: string;
  mode?: string;
  latestRun?: LedgerRun;
  completedCount: number;
  failedCount: number;
  activeCount: number;
}) {
  const parts = [
    batchId ? `Batch ${shortId(batchId)}` : '',
    mode ? `Mode ${modeLabel(mode)}` : '',
    latestRun ? `Latest run ${shortId(latestRun.id)} is ${statusLabel(latestRun.status).toLowerCase()}` : '',
    `${completedCount} completed`,
    `${failedCount} failed`,
    `${activeCount} active`,
  ].filter(Boolean);
  return parts.join(' | ');
}

function modeLabel(mode?: string) {
  switch (mode) {
    case 'full_cycle':
      return 'full cycle';
    case 'step_by_step':
      return 'step by step';
    default:
      return mode || 'unknown';
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case 'completed':
      return 'Completed';
    case 'failed':
      return 'Failed';
    case 'awaiting_human':
      return 'Awaiting human';
    case 'queued':
      return 'Queued';
    case 'blocked':
      return 'Blocked';
    default:
      return status ? status.replace(/_/g, ' ') : 'Pending';
  }
}

function runStatusColor(status?: string) {
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

function previewSignalLabel(status: DeliveryLedgerRunSignal['preview']) {
  switch (status) {
    case 'accepted':
      return 'Preview accepted';
    case 'rejected':
      return 'Preview rejected';
    case 'pending':
      return 'Preview waiting';
    default:
      return 'Preview missing';
  }
}

function previewSignalColor(status: DeliveryLedgerRunSignal['preview']) {
  switch (status) {
    case 'accepted':
      return 'green';
    case 'rejected':
      return 'red';
    case 'pending':
      return 'gold';
    default:
      return 'default';
  }
}

function qualitySignalLabel(status: DeliveryLedgerRunSignal['quality']) {
  switch (status) {
    case 'passed':
      return 'Quality passed';
    case 'failed':
      return 'Quality failed';
    default:
      return 'Quality pending';
  }
}

function qualitySignalColor(status: DeliveryLedgerRunSignal['quality']) {
  switch (status) {
    case 'passed':
      return 'green';
    case 'failed':
      return 'red';
    default:
      return 'blue';
  }
}

function shortId(value?: string) {
  return value ? value.slice(0, 8) : '-';
}
