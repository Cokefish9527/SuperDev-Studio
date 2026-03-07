import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import { apiClient } from '../api/client';
import type { Memory } from '../types';
import { useProjectState } from '../state/project-context';

const memoryFetchLimit = 50;
const memoryPageSize = 6;

export default function MemoryPage() {
  const { activeProjectId } = useProjectState();
  const [form] = Form.useForm();
  const [keyword, setKeyword] = useState('');
  const [roleFilter, setRoleFilter] = useState('all');
  const [memoryPage, setMemoryPage] = useState(1);
  const queryClient = useQueryClient();

  const memoriesQuery = useQuery({
    queryKey: ['memories', activeProjectId],
    queryFn: () => apiClient.listMemories(activeProjectId, memoryFetchLimit),
    enabled: !!activeProjectId,
  });

  const createMutation = useMutation({
    mutationFn: (payload: Partial<Memory>) => apiClient.createMemory(activeProjectId, payload),
    onSuccess: () => {
      form.resetFields();
      message.success('记忆已写入');
      void queryClient.invalidateQueries({ queryKey: ['memories', activeProjectId] });
    },
    onError: (error: Error) => message.error(error.message),
  });

  const memories = memoriesQuery.data ?? [];
  const roleOptions = useMemo(
    () => ['all', ...Array.from(new Set(memories.map((item) => item.role).filter(Boolean)))],
    [memories],
  );

  const filteredMemories = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    return memories.filter((item) => {
      const roleMatched = roleFilter === 'all' || item.role === roleFilter;
      const keywordMatched =
        normalizedKeyword === '' ||
        item.content.toLowerCase().includes(normalizedKeyword) ||
        item.role.toLowerCase().includes(normalizedKeyword) ||
        (item.tags ?? []).some((tag) => tag.toLowerCase().includes(normalizedKeyword));
      return roleMatched && keywordMatched;
    });
  }, [keyword, memories, roleFilter]);

  const totalMemoryPages = Math.max(1, Math.ceil(filteredMemories.length / memoryPageSize));
  const currentMemoryPage = Math.min(memoryPage, totalMemoryPages);

  useEffect(() => {
    setMemoryPage(1);
  }, [keyword, roleFilter]);

  const columns = [
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (role: string) => <Tag color="cyan">{role}</Tag>,
    },
    {
      title: '内容',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: '标签',
      dataIndex: 'tags',
      key: 'tags',
      width: 220,
      render: (tags: string[]) => (
        <Space wrap>
          {tags?.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '重要度',
      dataIndex: 'importance',
      key: 'importance',
      width: 100,
      render: (value: number) => value.toFixed(1),
    },
    {
      title: '写入时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (value: string) => dayjs(value).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
        {'记忆模块'}
      </Typography.Title>

      <Card title={'新增记忆'}>
        {!activeProjectId ? (
          <Empty description={'请先选择项目'} />
        ) : (
          <Form
            form={form}
            layout="vertical"
            initialValues={{ role: 'note', importance: 0.7 }}
            onFinish={(values: { role: string; content: string; tags?: string; importance: number }) => {
              createMutation.mutate({
                role: values.role,
                content: values.content,
                importance: values.importance,
                tags: (values.tags || '')
                  .split(',')
                  .map((item) => item.trim())
                  .filter(Boolean),
              });
            }}
          >
            <Form.Item name="role" label={'角色'} rules={[{ required: true }]}>
              <Input placeholder="user / assistant / note" />
            </Form.Item>
            <Form.Item name="content" label={'内容'} rules={[{ required: true }]}>
              <Input.TextArea rows={4} />
            </Form.Item>
            <Form.Item name="tags" label={'标签（逗号分隔）'}>
              <Input placeholder={'架构, 风险, 接口'} />
            </Form.Item>
            <Form.Item name="importance" label={'重要度（0-1）'}>
              <InputNumber min={0} max={1} step={0.1} style={{ width: 180 }} />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={createMutation.isPending}>
              {'写入记忆'}
            </Button>
          </Form>
        )}
      </Card>

      <Card
        title={'记忆列表'}
        extra={
          memories.length > 0 ? (
            <Space wrap>
              <Typography.Text type="secondary" data-testid="memory-summary">
                {'已筛选'} {filteredMemories.length} / {memories.length}
              </Typography.Text>
              <Button
                data-testid="memory-back-top"
                size="small"
                onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
              >
                {'回到顶部'}
              </Button>
            </Space>
          ) : null
        }
      >
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
            <div data-testid="memory-search-box" style={{ width: 320, maxWidth: '100%' }}>
              <Input.Search
                allowClear
                placeholder={'搜索内容 / 角色 / 标签'}
                style={{ width: '100%' }}
                onSearch={(value) => setKeyword(value.trim())}
                onChange={(event) => setKeyword(event.target.value)}
                value={keyword}
              />
            </div>
            <Select
              value={roleFilter}
              style={{ width: 220 }}
              onChange={setRoleFilter}
              options={roleOptions.map((item) => ({
                value: item,
                label: item === 'all' ? '全部角色' : item,
              }))}
            />
          </Space>

          <Table<Memory>
            rowKey="id"
            columns={columns}
            dataSource={filteredMemories}
            loading={memoriesQuery.isLoading}
            locale={{ emptyText: activeProjectId ? '暂无记忆数据' : '请先选择项目' }}
            pagination={{
              current: currentMemoryPage,
              pageSize: memoryPageSize,
              showSizeChanger: false,
              onChange: (page) => setMemoryPage(page),
            }}
            scroll={{ x: 980 }}
          />
        </Space>
      </Card>
    </Space>
  );
}
