import { useMutation } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Tag,
  Typography,
  message,
} from 'antd';
import { useState } from 'react';
import { apiClient } from '../api/client';
import type { ContextPack } from '../types';
import { useProjectState } from '../state/project-context';

export default function ContextPage() {
  const { activeProjectId } = useProjectState();
  const [pack, setPack] = useState<ContextPack | null>(null);

  const mutation = useMutation({
    mutationFn: (payload: { query: string; token_budget?: number; max_items?: number }) =>
      apiClient.buildContextPack(activeProjectId, payload),
    onSuccess: (data) => {
      setPack(data);
      message.success('上下文包已生成');
    },
    onError: (error: Error) => message.error(error.message),
  });

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        上下文优化器
      </Typography.Title>
      <Card title="构建 Context Pack">
        {!activeProjectId ? (
          <Empty description="请先选择项目" />
        ) : (
          <Form
            layout="vertical"
            onFinish={(values) => {
              mutation.mutate(values);
            }}
            initialValues={{ token_budget: 1200, max_items: 8 }}
          >
            <Form.Item name="query" label="当前目标问题" rules={[{ required: true }]}> 
              <Input.TextArea rows={3} placeholder="例如：我现在要实现流水线错误回滚，给我最相关上下文" />
            </Form.Item>
            <Row gutter={16}>
              <Col xs={24} md={8}>
                <Form.Item name="token_budget" label="Token 预算">
                  <InputNumber min={200} max={8000} step={100} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} md={8}>
                <Form.Item name="max_items" label="最大条目数">
                  <InputNumber min={2} max={20} step={1} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
            <Button type="primary" htmlType="submit" loading={mutation.isPending}>
              生成上下文包
            </Button>
          </Form>
        )}
      </Card>

      <Card title="优化结果">
        {!pack ? (
          <Empty description="提交查询后查看结果" />
        ) : (
          <Space orientation="vertical" style={{ width: '100%' }} size="large">
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="查询">{pack.query}</Descriptions.Item>
              <Descriptions.Item label="预算">{pack.token_budget}</Descriptions.Item>
              <Descriptions.Item label="估算 Token">{pack.estimated_tokens}</Descriptions.Item>
              <Descriptions.Item label="摘要">
                <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>
                  {pack.summary}
                </Typography.Paragraph>
              </Descriptions.Item>
            </Descriptions>

            <Row gutter={16}>
              <Col xs={24} lg={12}>
                <Card size="small" title={`记忆片段 (${pack.memories.length})`}>
                  {pack.memories.length > 0 ? (
                    <Space orientation="vertical" style={{ width: '100%' }}>
                      {pack.memories.map((item) => (
                        <Card key={item.id} size="small">
                          <Space>
                            <Tag color="cyan">{item.role}</Tag>
                            <Tag color="gold">importance {item.importance.toFixed(1)}</Tag>
                          </Space>
                          <Typography.Paragraph style={{ marginBottom: 0 }}>{item.content}</Typography.Paragraph>
                        </Card>
                      ))}
                    </Space>
                  ) : (
                    <Empty description="暂无记忆片段" />
                  )}
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card size="small" title={`知识片段 (${pack.knowledge.length})`}>
                  {pack.knowledge.length > 0 ? (
                    <Space orientation="vertical" style={{ width: '100%' }}>
                      {pack.knowledge.map((item) => (
                        <Card key={item.id} size="small">
                          <Tag color="blue">doc: {item.document_id}</Tag>
                          <Typography.Paragraph style={{ marginBottom: 0 }}>{item.content}</Typography.Paragraph>
                        </Card>
                      ))}
                    </Space>
                  ) : (
                    <Empty description="暂无知识片段" />
                  )}
                </Card>
              </Col>
            </Row>
          </Space>
        )}
      </Card>
    </Space>
  );
}
