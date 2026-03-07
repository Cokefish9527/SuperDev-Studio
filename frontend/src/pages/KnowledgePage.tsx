import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  InputNumber,
  Space,
  Table,
  Typography,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import { apiClient } from '../api/client';
import type { KnowledgeChunk, KnowledgeDocument } from '../types';
import { useProjectState } from '../state/project-context';

const searchPageSize = 4;
const knowledgeSearchLimit = 8;

export default function KnowledgePage() {
  const { activeProjectId } = useProjectState();
  const [form] = Form.useForm();
  const [searchQuery, setSearchQuery] = useState('');
  const [resultPage, setResultPage] = useState(1);
  const queryClient = useQueryClient();

  const docsQuery = useQuery({
    queryKey: ['knowledge-docs', activeProjectId],
    queryFn: () => apiClient.listKnowledgeDocuments(activeProjectId),
    enabled: !!activeProjectId,
  });

  const searchResults = useQuery({
    queryKey: ['knowledge-search', activeProjectId, searchQuery],
    queryFn: () => apiClient.searchKnowledge(activeProjectId, searchQuery, knowledgeSearchLimit),
    enabled: !!activeProjectId && !!searchQuery,
  });

  const createDoc = useMutation({
    mutationFn: (payload: { title: string; source: string; content: string; chunk_size?: number }) =>
      apiClient.createKnowledgeDocument(activeProjectId, payload),
    onSuccess: (res) => {
      form.resetFields();
      message.success(`文档已入库，生成 ${res.chunks.length} 个切片`);
      void queryClient.invalidateQueries({ queryKey: ['knowledge-docs', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message),
  });

  const resultItems = searchResults.data ?? [];
  const totalPages = Math.max(1, Math.ceil(resultItems.length / searchPageSize));
  const currentPage = Math.min(resultPage, totalPages);
  const visibleResults = useMemo(
    () => resultItems.slice((currentPage - 1) * searchPageSize, currentPage * searchPageSize),
    [currentPage, resultItems],
  );

  useEffect(() => {
    setResultPage(1);
  }, [searchQuery]);

  const columns = [
    { title: '标题', dataIndex: 'title', key: 'title' },
    { title: '来源', dataIndex: 'source', key: 'source' },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        {'知识库管理'}
      </Typography.Title>

      <Card title={'导入文档'}>
        {!activeProjectId ? (
          <Empty description={'请先选择项目'} />
        ) : (
          <Form
            form={form}
            layout="vertical"
            initialValues={{ chunk_size: 500 }}
            onFinish={(values) => createDoc.mutate(values)}
          >
            <Form.Item name="title" label={'文档标题'} rules={[{ required: true }]}>
              <Input placeholder={'产品设计规范 v1'} />
            </Form.Item>
            <Form.Item name="source" label={'来源'}>
              <Input placeholder="Confluence / URL / Meeting Notes" />
            </Form.Item>
            <Form.Item name="content" label={'文档正文'} rules={[{ required: true }]}>
              <Input.TextArea rows={8} />
            </Form.Item>
            <Form.Item name="chunk_size" label={'切片大小（字符）'}>
              <InputNumber min={100} max={2000} step={50} style={{ width: 200 }} />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={createDoc.isPending}>
              {'入库并切片'}
            </Button>
          </Form>
        )}
      </Card>

      <Card
        title={'知识检索'}
        extra={
          resultItems.length > 0 ? (
            <Space wrap>
              <Typography.Text type="secondary" data-testid="knowledge-hit-count">
                {'共'} {resultItems.length} {'条'}
              </Typography.Text>
              <Button
                data-testid="knowledge-back-top"
                size="small"
                onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
              >
                {'回到顶部'}
              </Button>
            </Space>
          ) : null
        }
      >
        {!activeProjectId ? (
          <Empty description={'请先选择项目'} />
        ) : (
          <Space orientation="vertical" style={{ width: '100%' }}>
            <div data-testid="knowledge-search-box">
              <Input.Search
                allowClear
                placeholder={'输入关键词检索知识库'}
                onSearch={(value) => setSearchQuery(value.trim())}
                enterButton={'检索'}
              />
            </div>
            <Card loading={searchResults.isLoading}>
              {resultItems.length > 0 ? (
                <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                  <Typography.Text type="secondary" data-testid="knowledge-summary">
                    {'当前显示'} {(currentPage - 1) * searchPageSize + 1}-
                    {Math.min(currentPage * searchPageSize, resultItems.length)} / {resultItems.length}
                  </Typography.Text>
                  {visibleResults.map((item: KnowledgeChunk) => (
                    <Card key={item.id} size="small">
                      <Typography.Text strong>
                        {'文档'} {item.document_id} / chunk #{item.chunk_index}
                      </Typography.Text>
                      <Typography.Paragraph style={{ marginBottom: 0 }}>{item.content}</Typography.Paragraph>
                    </Card>
                  ))}
                  {totalPages > 1 ? (
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                      <Button
                        data-testid="knowledge-prev-page"
                        size="small"
                        onClick={() => setResultPage((value) => Math.max(1, value - 1))}
                        disabled={currentPage === 1}
                      >
                        {'上一页'}
                      </Button>
                      <Typography.Text type="secondary">
                        {'第'} {currentPage} / {totalPages} {'页'}
                      </Typography.Text>
                      <Button
                        data-testid="knowledge-next-page"
                        size="small"
                        onClick={() => setResultPage((value) => Math.min(totalPages, value + 1))}
                        disabled={currentPage === totalPages}
                      >
                        {'下一页'}
                      </Button>
                    </div>
                  ) : null}
                </Space>
              ) : (
                <Empty description={searchQuery ? '暂无匹配结果' : '请输入关键词开始检索'} />
              )}
            </Card>
          </Space>
        )}
      </Card>

      <Card title={'文档列表'}>
        <Table<KnowledgeDocument>
          rowKey="id"
          columns={columns}
          dataSource={docsQuery.data ?? []}
          loading={docsQuery.isLoading}
          pagination={{ pageSize: 6 }}
          scroll={{ x: 880 }}
        />
      </Card>
    </Space>
  );
}
