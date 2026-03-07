import { Button, Card, Empty, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import type { Task } from '../../types';

const ganttLabelWidth = 240;
const ganttCellWidth = 36;
const rowPageSize = 6;
const dayWindowSize = 14;

type Props = {
  tasks: Task[];
  projectSelected: boolean;
};

const taskBarColor = (status: string) => {
  switch (status) {
    case 'done':
      return '#52c41a';
    case 'in_progress':
      return '#1677ff';
    default:
      return '#faad14';
  }
};

export default function ProjectScheduleGanttCard({ tasks, projectSelected }: Props) {
  const [rowPage, setRowPage] = useState(1);
  const [dayPage, setDayPage] = useState(1);

  const ganttData = useMemo(() => {
    const scheduledTasks = tasks
      .map((task) => {
        if (!task.start_date || !task.due_date) {
          return null;
        }
        const start = dayjs(task.start_date).startOf('day');
        const end = dayjs(task.due_date).startOf('day');
        if (!start.isValid() || !end.isValid() || end.isBefore(start)) {
          return null;
        }
        return { ...task, start, end };
      })
      .filter((task): task is Task & { start: dayjs.Dayjs; end: dayjs.Dayjs } => task !== null)
      .sort((left, right) => left.start.valueOf() - right.start.valueOf());

    if (!scheduledTasks.length) {
      return null;
    }

    const timelineStart = scheduledTasks.reduce(
      (current, task) => (task.start.isBefore(current) ? task.start : current),
      scheduledTasks[0].start,
    );
    const timelineEnd = scheduledTasks.reduce(
      (current, task) => (task.end.isAfter(current) ? task.end : current),
      scheduledTasks[0].end,
    );
    const totalDays = Math.max(1, timelineEnd.diff(timelineStart, 'day') + 1);
    const days = Array.from({ length: totalDays }, (_, index) => timelineStart.add(index, 'day'));
    const rows = scheduledTasks.map((task) => {
      const offset = Math.max(0, task.start.diff(timelineStart, 'day'));
      const span = Math.max(1, task.end.diff(task.start, 'day') + 1);
      return { ...task, offset, span };
    });

    return {
      days,
      rows,
      totalDays,
      totalRows: rows.length,
    };
  }, [tasks]);

  useEffect(() => {
    setRowPage(1);
    setDayPage(1);
  }, [tasks]);

  if (!projectSelected) {
    return (
      <Card title="计划排期视图">
        <Typography.Text type="secondary">请选择一个工作区以查看甘特图。</Typography.Text>
      </Card>
    );
  }

  if (!ganttData) {
    return (
      <Card title="计划排期视图">
        <Typography.Text type="secondary">当前暂无可绘制的排期数据，请先点击“自动生成排期”。</Typography.Text>
      </Card>
    );
  }

  const totalRowPages = Math.max(1, Math.ceil(ganttData.totalRows / rowPageSize));
  const totalDayPages = Math.max(1, Math.ceil(ganttData.totalDays / dayWindowSize));
  const currentRowPage = Math.min(rowPage, totalRowPages);
  const currentDayPage = Math.min(dayPage, totalDayPages);
  const rowStartIndex = (currentRowPage - 1) * rowPageSize;
  const rowEndIndex = Math.min(ganttData.totalRows, rowStartIndex + rowPageSize);
  const dayStartIndex = (currentDayPage - 1) * dayWindowSize;
  const dayEndIndex = Math.min(ganttData.totalDays, dayStartIndex + dayWindowSize);
  const visibleRows = ganttData.rows.slice(rowStartIndex, rowEndIndex);
  const visibleDays = ganttData.days.slice(dayStartIndex, dayEndIndex);
  const visibleGridWidth = visibleDays.length * ganttCellWidth;
  const dayRangeLabel = `${visibleDays[0].format('YYYY-MM-DD')} - ${visibleDays[visibleDays.length - 1].format('YYYY-MM-DD')}`;

  return (
    <Card
      title="计划排期视图"
      style={{ borderRadius: 18 }}
      extra={(
        <Space wrap>
          <Tag color="blue">任务 {rowStartIndex + 1}-{rowEndIndex} / {ganttData.totalRows}</Tag>
          <Tag color="purple">日期 {dayStartIndex + 1}-{dayEndIndex} / {ganttData.totalDays}</Tag>
          <Button size="small" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
            回到顶部
          </Button>
        </Space>
      )}
    >
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 12,
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Space wrap>
            <Typography.Text strong>当前日期窗口</Typography.Text>
            <Typography.Text type="secondary">{dayRangeLabel}</Typography.Text>
          </Space>
          <Space wrap>
            <Button size="small" onClick={() => setDayPage((value) => Math.max(1, value - 1))} disabled={currentDayPage === 1}>
              上一段日期
            </Button>
            <Typography.Text type="secondary">日期 {currentDayPage} / {totalDayPages}</Typography.Text>
            <Button
              size="small"
              onClick={() => setDayPage((value) => Math.min(totalDayPages, value + 1))}
              disabled={currentDayPage === totalDayPages}
            >
              下一段日期
            </Button>
          </Space>
        </div>

        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: 12,
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Typography.Text type="secondary">
            当前显示第 {rowStartIndex + 1}-{rowEndIndex} 个任务条目，共 {ganttData.totalRows} 个。
          </Typography.Text>
          <Space wrap>
            <Button size="small" onClick={() => setRowPage((value) => Math.max(1, value - 1))} disabled={currentRowPage === 1}>
              上一组任务
            </Button>
            <Typography.Text type="secondary">任务 {currentRowPage} / {totalRowPages}</Typography.Text>
            <Button
              size="small"
              onClick={() => setRowPage((value) => Math.min(totalRowPages, value + 1))}
              disabled={currentRowPage === totalRowPages}
            >
              下一组任务
            </Button>
          </Space>
        </div>

        <div style={{ overflowX: 'auto', border: '1px solid #f0f0f0', borderRadius: 10 }}>
          <div style={{ minWidth: ganttLabelWidth + visibleGridWidth }}>
            <div style={{ display: 'flex', borderBottom: '1px solid #f0f0f0', background: '#fafafa' }}>
              <div
                style={{
                  width: ganttLabelWidth,
                  padding: '10px 12px',
                  fontWeight: 600,
                  flex: '0 0 auto',
                }}
              >
                任务
              </div>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: `repeat(${visibleDays.length}, ${ganttCellWidth}px)`,
                  flex: '0 0 auto',
                }}
              >
                {visibleDays.map((day) => (
                  <div
                    key={day.format('YYYY-MM-DD')}
                    style={{
                      borderLeft: '1px solid #f0f0f0',
                      textAlign: 'center',
                      fontSize: 12,
                      color: '#666',
                      padding: '10px 0',
                    }}
                  >
                    {day.format('MM/DD')}
                  </div>
                ))}
              </div>
            </div>

            {visibleRows.length === 0 ? (
              <div style={{ padding: 24 }}>
                <Empty description="当前分页暂无任务" />
              </div>
            ) : (
              visibleRows.map((row) => {
                const visibleBarStart = Math.max(row.offset, dayStartIndex);
                const visibleBarEnd = Math.min(row.offset + row.span, dayEndIndex);
                const visibleSpan = Math.max(0, visibleBarEnd - visibleBarStart);
                const shouldRenderBar = visibleSpan > 0;
                const clipped = row.offset < dayStartIndex || row.offset + row.span > dayEndIndex;

                return (
                  <div key={row.id} style={{ display: 'flex', borderBottom: '1px solid #f5f5f5' }}>
                    <div
                      style={{
                        width: ganttLabelWidth,
                        padding: '8px 12px',
                        flex: '0 0 auto',
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'center',
                      }}
                    >
                      <Typography.Text strong ellipsis>
                        {row.title}
                      </Typography.Text>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {row.status} | {row.priority}
                      </Typography.Text>
                    </div>
                    <div
                      style={{
                        width: visibleGridWidth,
                        height: 44,
                        position: 'relative',
                        flex: '0 0 auto',
                        backgroundImage: 'linear-gradient(to right, #f5f5f5 1px, transparent 1px)',
                        backgroundSize: `${ganttCellWidth}px 100%`,
                      }}
                    >
                      {shouldRenderBar ? (
                        <div
                          style={{
                            position: 'absolute',
                            left: (visibleBarStart - dayStartIndex) * ganttCellWidth + 2,
                            top: 8,
                            width: Math.max(visibleSpan * ganttCellWidth - 4, 18),
                            height: 28,
                            background: taskBarColor(row.status),
                            borderRadius: 6,
                            color: '#fff',
                            fontSize: 12,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            padding: '0 8px',
                            whiteSpace: 'nowrap',
                            opacity: clipped ? 0.88 : 1,
                          }}
                        >
                          {clipped ? `续 ${row.span}d` : `${row.span}d`}
                        </div>
                      ) : null}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </Space>
    </Card>
  );
}

