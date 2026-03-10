import { Alert, Card, Col, Empty, Row, Space, Tag, Timeline, Typography } from 'antd';
import dayjs from 'dayjs';
import type { RunEvent } from '../../types';
import { stageStatusColor } from './presentation';

type Props = {
  events: RunEvent[];
  loading?: boolean;
  maxItems?: number;
};

type ActivityKind = 'auto_advance' | 'backlog' | 'quality' | 'preview' | 'delivery';

type ActivityEvent = RunEvent & {
  kind: ActivityKind;
  title: string;
};

export default function AutonomyActivityCard({ events, loading, maxItems = 6 }: Props) {
  const activityEvents = events
    .map(toActivityEvent)
    .filter((item): item is ActivityEvent => item !== null)
    .sort((left, right) => dayjs(right.created_at).valueOf() - dayjs(left.created_at).valueOf());

  const latestEvent = activityEvents[0];
  const autoAdvanceCount = activityEvents.filter((item) => item.kind === 'auto_advance').length;
  const backlogCount = activityEvents.filter((item) => item.kind === 'backlog').length;
  const latestQuality = activityEvents.find((item) => item.kind === 'quality');

  return (
    <Card title="Autonomy Activity" style={{ borderRadius: 16 }} loading={loading} data-testid="simple-delivery-autonomy-card">
      {!activityEvents.length ? (
        <Empty description="No autonomous progress events yet" />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Alert
            data-testid="simple-delivery-autonomy-summary"
            showIcon
            type={summaryAlertType(latestEvent)}
            title={summaryTitle(latestEvent)}
            description={summaryDescription(latestEvent)}
          />

          <Row gutter={[12, 12]}>
            <Col xs={24} md={8}>
              <MetricTile label="Auto-advance" value={String(autoAdvanceCount)} note="Safe delivery steps executed automatically" />
            </Col>
            <Col xs={24} md={8}>
              <MetricTile label="Backlog shrink" value={String(backlogCount)} note="Historical residual work re-evaluated by the latest run" />
            </Col>
            <Col xs={24} md={8}>
              <MetricTile
                label="Latest quality"
                value={latestQuality ? eventStatusLabel(latestQuality.status) : 'Pending'}
                note={latestQuality ? latestQuality.message : 'No quality conclusion has been recorded yet.'}
              />
            </Col>
          </Row>

          <Typography.Text strong>Recent key actions</Typography.Text>
          <div style={{ maxHeight: 360, overflowY: 'auto', paddingRight: 8 }}>
            <Timeline
              items={activityEvents.slice(0, maxItems).map((event) => ({
                color: timelineColor(event),
                content: (
                  <Space orientation="vertical" size={4} style={{ width: '100%' }} data-testid={`autonomy-activity-event-${event.id}`}>
                    <Space wrap>
                      <Tag color={activityTagColor(event.kind)}>{event.title}</Tag>
                      <Tag color={stageStatusColor(event.status)}>{event.status}</Tag>
                    </Space>
                    <Typography.Text>{event.message}</Typography.Text>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {dayjs(event.created_at).format('YYYY-MM-DD HH:mm:ss')}
                    </Typography.Text>
                  </Space>
                ),
              }))}
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

function toActivityEvent(event: RunEvent): ActivityEvent | null {
  const stage = event.stage.toLowerCase();
  const message = event.message.toLowerCase();

  if (stage.includes('auto-advance') || message.includes('auto advance')) {
    return { ...event, kind: 'auto_advance', title: 'Auto advance' };
  }
  if (stage.includes('backlog-reconcile') || message.includes('carried forward') || message.includes('residual backlog')) {
    return { ...event, kind: 'backlog', title: 'Backlog shrink' };
  }
  if (stage.includes('quality') || message.includes('quality gate')) {
    return { ...event, kind: 'quality', title: 'Quality check' };
  }
  if (stage.includes('preview') || message.includes('preview')) {
    return { ...event, kind: 'preview', title: 'Preview review' };
  }
  if (stage === 'done' || message.includes('finished')) {
    return { ...event, kind: 'delivery', title: 'Delivery done' };
  }
  return null;
}

function summaryAlertType(event?: ActivityEvent) {
  if (!event) {
    return 'info';
  }
  if (event.kind === 'delivery' && event.status === 'completed') {
    return 'success';
  }
  if (event.status === 'failed') {
    return 'warning';
  }
  return 'info';
}

function summaryTitle(event?: ActivityEvent) {
  if (!event) {
    return 'Autonomy has not started yet';
  }
  switch (event.kind) {
    case 'delivery':
      return 'Delivery reached the final completed state';
    case 'backlog':
      return 'Historical backlog was reconciled';
    case 'auto_advance':
      return 'System is auto-advancing delivery';
    case 'quality':
      return 'Quality gate result updated';
    case 'preview':
      return 'Preview review is waiting for sign-off';
    default:
      return 'Key delivery activity recorded';
  }
}

function summaryDescription(event?: ActivityEvent) {
  if (!event) {
    return 'This card updates when the system auto-advances, reconciles backlog, refreshes quality results, or requests preview review.';
  }
  switch (event.kind) {
    case 'delivery':
      return `${event.message}. The run is now in its final delivery state and ready for acceptance handoff.`;
    case 'backlog':
      return `${event.message}. Historical residual work has been re-evaluated so the latest run can stay focused on remaining gaps.`;
    case 'auto_advance':
      return `${event.message}. The system continues dispatching the next safe step without manual intervention.`;
    case 'quality':
      return `${event.message}. The latest quality outcome has been refreshed for the current run.`;
    case 'preview':
      return `${event.message}. Human preview sign-off is still required before automatic progress can continue.`;
    default:
      return event.message;
  }
}

function activityTagColor(kind: ActivityKind) {
  switch (kind) {
    case 'auto_advance':
      return 'blue';
    case 'backlog':
      return 'cyan';
    case 'quality':
      return 'green';
    case 'preview':
      return 'purple';
    case 'delivery':
      return 'gold';
    default:
      return 'default';
  }
}

function timelineColor(event: ActivityEvent) {
  if (event.kind === 'backlog') {
    return '#06b6d4';
  }
  if (event.kind === 'delivery') {
    return '#22c55e';
  }
  return stageStatusColor(event.status);
}

function eventStatusLabel(status: string) {
  switch (status) {
    case 'completed':
      return 'Passed';
    case 'failed':
      return 'Failed';
    default:
      return 'Running';
  }
}
