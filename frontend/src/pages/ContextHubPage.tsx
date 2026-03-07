import { Card, Tabs, Typography } from 'antd';
import { useSearchParams } from 'react-router-dom';
import ContextPage from './ContextPage';
import KnowledgePage from './KnowledgePage';
import MemoryPage from './MemoryPage';

export default function ContextHubPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get('tab') || 'context-pack';
  const activeTab = ['context-pack', 'memory', 'knowledge'].includes(requestedTab) ? requestedTab : 'context-pack';

  return (
    <Card bordered={false} bodyStyle={{ padding: 0 }}>
      <Typography.Title level={2} style={{ margin: '0 0 20px', fontFamily: 'var(--heading-font)' }}>
        上下文中心
      </Typography.Title>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => setSearchParams(key === 'context-pack' ? {} : { tab: key }, { replace: true })}
        items={[
          { key: 'context-pack', label: 'Context Pack', children: <ContextPage /> },
          { key: 'memory', label: '记忆库', children: <MemoryPage /> },
          { key: 'knowledge', label: '知识库', children: <KnowledgePage /> },
        ]}
      />
    </Card>
  );
}
