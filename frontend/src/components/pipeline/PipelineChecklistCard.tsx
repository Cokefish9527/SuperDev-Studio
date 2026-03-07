import { Card, Empty, Space, Tag, Typography } from 'antd';
import type { PipelineCompletionItem } from '../../types';
import { stageStatusColor } from './presentation';

type Props = {
  checklist: PipelineCompletionItem[];
};

export default function PipelineChecklistCard({ checklist }: Props) {
  return (
    <Card title="完成清单" size="small" style={{ borderRadius: 16 }}>
      {checklist.length === 0 ? (
        <Empty description="暂无完成清单" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Space orientation="vertical" size={10} style={{ width: '100%' }}>
          {checklist.map((item) => (
            <div
              key={item.key}
              style={{
                padding: '12px 14px',
                borderRadius: 14,
                border: '1px solid #e2e8f0',
                background: '#fff',
              }}
            >
              <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                <Space wrap>
                  <Tag color={stageStatusColor(item.status)}>{item.status}</Tag>
                  <Typography.Text>{item.title}</Typography.Text>
                </Space>
                {item.note ? <Typography.Text type="secondary">{item.note}</Typography.Text> : null}
              </Space>
            </div>
          ))}
        </Space>
      )}
    </Card>
  );
}
