import { Button, Card, Empty, Space, Timeline, Typography } from 'antd';
import dayjs from 'dayjs';
import { useMemo, useState } from 'react';
import type { RunEvent } from '../../types';
import { stageStatusColor } from './presentation';

const TIMELINE_PAGE_SIZE = 8;

type Props = {
  events: RunEvent[];
};

export default function PipelineTimelineCard({ events }: Props) {
  const [page, setPage] = useState(1);
  const maxPage = Math.max(1, Math.ceil(events.length / TIMELINE_PAGE_SIZE));
  const currentPage = Math.min(page, maxPage);

  const range = useMemo(() => {
    if (events.length === 0) {
      return { start: 0, end: 0 };
    }
    const start = (currentPage - 1) * TIMELINE_PAGE_SIZE + 1;
    const end = Math.min(currentPage * TIMELINE_PAGE_SIZE, events.length);
    return { start, end };
  }, [currentPage, events.length]);

  const visibleEvents = useMemo(
    () => events.slice((currentPage - 1) * TIMELINE_PAGE_SIZE, currentPage * TIMELINE_PAGE_SIZE),
    [currentPage, events],
  );

  return (
    <Card
      title={'推进时间线'}
      style={{ borderRadius: 18 }}
      extra={events.length > 0 ? (
        <Space wrap>
          <Typography.Text type="secondary" data-testid="pipeline-timeline-summary">
            {'当前'} {range.start}-{range.end} / {events.length}
          </Typography.Text>
          <Button data-testid="pipeline-timeline-back-top" size="small" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
            {'回到顶部'}
          </Button>
        </Space>
      ) : null}
    >
      {events.length === 0 ? (
        <Empty description={'暂无时间线数据'} />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <div style={{ maxHeight: 480, overflowY: 'auto', paddingRight: 8 }}>
            <Timeline
              items={visibleEvents.map((event) => ({
                color: stageStatusColor(event.status),
                content: (
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
          </div>

          {maxPage > 1 ? (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
              <Button data-testid="pipeline-timeline-prev" onClick={() => setPage((value) => Math.max(1, Math.min(value, maxPage) - 1))} disabled={currentPage === 1}>
                {'上一页'}
              </Button>
              <Typography.Text type="secondary">{'第'} {currentPage} / {maxPage} {'页'}</Typography.Text>
              <Button data-testid="pipeline-timeline-next" onClick={() => setPage((value) => Math.min(maxPage, Math.min(value, maxPage) + 1))} disabled={currentPage === maxPage}>
                {'下一页'}
              </Button>
            </div>
          ) : null}
        </Space>
      )}
    </Card>
  );
}
