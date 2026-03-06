import {
  AppstoreOutlined,
  BookOutlined,
  DashboardOutlined,
  FolderOpenOutlined,
  RobotOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Layout, Select, Space, Tag, Typography, theme } from 'antd';
import { useEffect } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { apiClient } from '../api/client';
import { useProjectState } from '../state/project-context';

const { Header, Sider, Content } = Layout;

const menuItems = [
  { key: '/', label: '概览', icon: <DashboardOutlined /> },
  { key: '/projects', label: '项目管理', icon: <FolderOpenOutlined /> },
  { key: '/pipeline', label: '流水线', icon: <ThunderboltOutlined /> },
  { key: '/memory', label: '记忆模块', icon: <RobotOutlined /> },
  { key: '/knowledge', label: '知识库', icon: <BookOutlined /> },
  { key: '/context', label: '上下文优化', icon: <AppstoreOutlined /> },
];

export default function AppShell() {
  const location = useLocation();
  const { token } = theme.useToken();
  const { activeProjectId, setActiveProjectId } = useProjectState();

  const projectsQuery = useQuery({
    queryKey: ['projects'],
    queryFn: apiClient.listProjects,
  });

  const projects = projectsQuery.data ?? [];
  const firstProjectId = projects[0]?.id ?? '';
  const selectOptions = projects.map((project) => ({ label: project.name, value: project.id }));

  const currentItemKey = menuItems.find((item) =>
    item.key === '/'
      ? location.pathname === '/'
      : location.pathname.startsWith(item.key),
  )?.key;

  useEffect(() => {
    if (!activeProjectId && firstProjectId) {
      setActiveProjectId(firstProjectId);
    }
  }, [activeProjectId, firstProjectId, setActiveProjectId]);

  return (
    <Layout style={{ minHeight: '100vh', background: 'var(--app-bg)' }}>
      <Sider
        width={260}
        style={{
          background: 'linear-gradient(195deg, #083344 0%, #164e63 40%, #1f2937 100%)',
          borderRight: '1px solid rgba(148, 163, 184, 0.2)',
        }}
      >
        <div style={{ padding: '22px 20px 10px', color: '#f8fafc' }}>
          <Typography.Title
            level={3}
            style={{ margin: 0, color: '#f8fafc', fontFamily: 'var(--heading-font)' }}
          >
            SuperDev Studio
          </Typography.Title>
          <Typography.Text style={{ color: '#cbd5e1' }}>基于 super-dev 的可视化工程中枢</Typography.Text>
        </div>
        <nav style={{ padding: '14px 10px', display: 'grid', gap: 8 }}>
          {menuItems.map((item) => {
            const active = item.key === currentItemKey;
            return (
              <Link
                key={item.key}
                to={item.key}
                style={{
                  color: active ? '#0f172a' : '#e2e8f0',
                  background: active ? '#f59e0b' : 'transparent',
                  borderRadius: 12,
                  textDecoration: 'none',
                  padding: '12px 14px',
                  display: 'flex',
                  gap: 10,
                  alignItems: 'center',
                  fontWeight: 600,
                }}
              >
                {item.icon}
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>
      </Sider>
      <Layout>
        <Header
          style={{
            background: token.colorBgElevated,
            borderBottom: '1px solid rgba(148, 163, 184, 0.25)',
            paddingInline: 24,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Space size="middle">
            <Tag color="geekblue">Super Dev 12 阶段模型</Tag>
            <Typography.Text type="secondary">CLI 能力 + GUI 协作面板</Typography.Text>
          </Space>
          <Space>
            <Typography.Text strong>当前项目</Typography.Text>
            <Select
              value={activeProjectId || undefined}
              placeholder={projects.length ? '请选择项目' : '请先创建项目'}
              options={selectOptions}
              style={{ minWidth: 260 }}
              onChange={setActiveProjectId}
              loading={projectsQuery.isLoading}
            />
          </Space>
        </Header>
        <Content style={{ padding: 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
