import { useQuery } from '@tanstack/react-query';
import { Card, Col, Empty, Progress, Row, Space, Statistic, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';

export default function DashboardPage() {
  const { activeProjectId } = useProjectState();

  const dashboardQuery = useQuery({
    queryKey: ['dashboard', activeProjectId],
    queryFn: () => apiClient.getDashboard(activeProjectId || undefined),
  });

  const stats = dashboardQuery.data?.stats;
  const recentRuns = dashboardQuery.data?.recent_runs ?? [];

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ marginBottom: 0, fontFamily: 'var(--heading-font)' }}>
        工作台总览
      </Typography.Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} xl={4}>
          <Card>
            <Statistic title="项目总数" value={stats?.projects ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} xl={4}>
          <Card>
            <Statistic title="计划任务" value={stats?.tasks ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} xl={4}>
          <Card>
            <Statistic title="交付运行" value={stats?.runs ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} xl={4}>
          <Card>
            <Statistic title="记忆条目" value={stats?.memories ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} xl={4}>
          <Card>
            <Statistic title="知识文档" value={stats?.docs ?? 0} />
          </Card>
        </Col>
      </Row>

      <Card title="最近交付运行">
        {!activeProjectId ? (
          <Empty description="请在右上角选择工作区" />
        ) : recentRuns.length === 0 ? (
          <Empty description="当前工作区暂无运行记录" />
        ) : (
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            {recentRuns.map((run) => (
              <Card key={run.id} size="small">
                <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                  <Space orientation="vertical" size={4}>
                    <Space>
                      <Tag color={run.status === 'completed' ? 'green' : run.status === 'failed' ? 'red' : 'blue'}>
                        {run.status}
                      </Tag>
                      <Typography.Text>{run.prompt}</Typography.Text>
                    </Space>
                    <Typography.Text type="secondary">
                      阶段: {run.stage} | 创建时间: {dayjs(run.created_at).format('YYYY-MM-DD HH:mm:ss')}
                    </Typography.Text>
                  </Space>
                  <Progress percent={run.progress} size="small" style={{ minWidth: 200 }} />
                </Space>
              </Card>
            ))}
          </Space>
        )}
      </Card>
    </Space>
  );
}
