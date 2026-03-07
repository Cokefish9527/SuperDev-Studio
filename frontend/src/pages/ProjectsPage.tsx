import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import dayjs from 'dayjs';
import { useMemo, useState } from 'react';
import { apiClient } from '../api/client';
import ProjectScheduleGanttCard from '../components/projects/ProjectScheduleGanttCard';
import { useProjectState } from '../state/project-context';
import type { Project, Task } from '../types';

type TaskFormValues = {
  title: string;
  description?: string;
  status: string;
  priority: string;
  assignee?: string;
  start_date?: dayjs.Dayjs;
  due_date?: dayjs.Dayjs;
  estimated_days?: number;
};

const formatDateValue = (value?: string) => {
  if (!value) {
    return '-';
  }
  const parsed = dayjs(value);
  if (!parsed.isValid()) {
    return '-';
  }
  return parsed.format('YYYY-MM-DD');
};


export default function ProjectsPage() {
  const [open, setOpen] = useState(false);
  const [taskOpen, setTaskOpen] = useState(false);
  const [form] = Form.useForm();
  const [taskForm] = Form.useForm<TaskFormValues>();
  const queryClient = useQueryClient();
  const { activeProjectId, setActiveProjectId } = useProjectState();

  const projectsQuery = useQuery({ queryKey: ['projects'], queryFn: apiClient.listProjects });

  const tasksQuery = useQuery({
    queryKey: ['tasks', activeProjectId],
    queryFn: () => apiClient.listTasks(activeProjectId),
    enabled: !!activeProjectId,
  });

  const createProject = useMutation({
    mutationFn: apiClient.createProject,
    onSuccess: (project) => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] });
      message.success('项目已创建');
      setOpen(false);
      form.resetFields();
      setActiveProjectId(project.id);
    },
  });

  const updateTask = useMutation({
    mutationFn: ({ taskId, payload }: { taskId: string; payload: Partial<Task> }) =>
      apiClient.updateTask(taskId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tasks', activeProjectId] });
    },
  });

  const createTask = useMutation({
    mutationFn: (payload: Partial<Task>) => apiClient.createTask(activeProjectId, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tasks', activeProjectId] });
      message.success('任务已新增');
      setTaskOpen(false);
      taskForm.resetFields();
    },
  });

  const autoScheduleTasks = useMutation({
    mutationFn: () => apiClient.autoScheduleTasks(activeProjectId),
    onSuccess: (result) => {
      queryClient.setQueryData(['tasks', activeProjectId], result.items);
      message.success(`自动排期完成，已更新 ${result.scheduled_count} 个任务`);
    },
    onError: (error: Error) => {
      message.error(error.message || '自动排期失败');
    },
  });

  const advanceProject = useMutation({
    mutationFn: () =>
      apiClient.advanceProject(activeProjectId, {
        mode: 'step_by_step',
        iteration_limit: 3,
        platform: 'web',
        frontend: 'react',
        backend: 'go',
      }),
    onSuccess: (result) => {
      const memoryHint = result.memory_written ? '已写入 super-dev 使用记忆。' : '已复用现有 super-dev 使用记忆。';
      message.success(`项目推进运行已启动（${result.run.id}）。${memoryHint}`);
      void queryClient.invalidateQueries({ queryKey: ['tasks', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['runs', activeProjectId] });
      void queryClient.invalidateQueries({ queryKey: ['memories', activeProjectId] });
    },
    onError: (error: Error) => {
      message.error(error.message || '项目推进启动失败');
    },
  });

  const projectColumns = useMemo(
    () => [
      {
        title: '项目名',
        dataIndex: 'name',
        key: 'name',
        render: (_: unknown, record: Project) => (
          <Button
            type={activeProjectId === record.id ? 'primary' : 'link'}
            onClick={() => setActiveProjectId(record.id)}
          >
            {record.name}
          </Button>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => <Tag color={status === 'active' ? 'green' : 'default'}>{status}</Tag>,
      },
      {
        title: '仓库路径',
        dataIndex: 'repo_path',
        key: 'repo_path',
      },
      {
        title: '描述',
        dataIndex: 'description',
        key: 'description',
      },
    ],
    [activeProjectId, setActiveProjectId],
  );

  const taskColumns = [
    {
      title: '任务',
      dataIndex: 'title',
      key: 'title',
      width: 220,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (status: string, record: Task) => (
        <Select
          value={status}
          style={{ width: 130 }}
          options={[
            { value: 'todo', label: 'todo' },
            { value: 'in_progress', label: 'in_progress' },
            { value: 'done', label: 'done' },
          ]}
          onChange={(value) => updateTask.mutate({ taskId: record.id, payload: { status: value } })}
        />
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 120,
    },
    {
      title: '负责人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 140,
    },
    {
      title: '开始日期',
      dataIndex: 'start_date',
      key: 'start_date',
      width: 140,
      render: (value?: string) => formatDateValue(value),
    },
    {
      title: '截止日期',
      dataIndex: 'due_date',
      key: 'due_date',
      width: 140,
      render: (value?: string) => formatDateValue(value),
    },
    {
      title: '工期(天)',
      dataIndex: 'estimated_days',
      key: 'estimated_days',
      width: 110,
      render: (value: number) => (value > 0 ? value : '-'),
    },
  ];


  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Row justify="space-between" align="middle">
        <Typography.Title level={2} style={{ margin: 0, fontFamily: 'var(--heading-font)' }}>
          工作区与计划任务
        </Typography.Title>
        <Space>
          <Button
            type="primary"
            ghost
            disabled={!activeProjectId}
            loading={advanceProject.isPending}
            onClick={() => advanceProject.mutate()}
          >
            一键推进
          </Button>
          <Button onClick={() => setTaskOpen(true)} disabled={!activeProjectId}>
            新建任务
          </Button>
          <Button type="primary" onClick={() => setOpen(true)}>
            新建工作区
          </Button>
        </Space>
      </Row>

      <Card title="工作区列表">
        <Table<Project>
          rowKey="id"
          columns={projectColumns}
          dataSource={projectsQuery.data ?? []}
          loading={projectsQuery.isLoading}
          pagination={false}
        />
      </Card>

      <Card
        title="计划任务看板"
        extra={
          <Button
            type="primary"
            disabled={!activeProjectId || !(tasksQuery.data ?? []).length}
            loading={autoScheduleTasks.isPending}
            onClick={() => autoScheduleTasks.mutate()}
          >
            自动生成排期
          </Button>
        }
      >
        {!activeProjectId ? (
          <Typography.Text type="secondary">请选择一个工作区以查看任务。</Typography.Text>
        ) : (
          <Table<Task>
            rowKey="id"
            columns={taskColumns}
            dataSource={tasksQuery.data ?? []}
            loading={tasksQuery.isLoading}
            pagination={{ pageSize: 8 }}
            scroll={{ x: 1080 }}
          />
        )}
      </Card>

      <ProjectScheduleGanttCard tasks={tasksQuery.data ?? []} projectSelected={!!activeProjectId} />

      <Modal
        open={open}
        title="创建工作区"
        onCancel={() => setOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createProject.isPending}
      >
        <Form
          layout="vertical"
          form={form}
          onFinish={(values) => createProject.mutate(values)}
          initialValues={{ status: 'active' }}
        >
          <Form.Item name="name" label="项目名" rules={[{ required: true }]}>
            <Input placeholder="SuperDev Studio 工作区" />
          </Form.Item>
          <Form.Item name="repo_path" label="仓库路径">
            <Input placeholder="D:/Work/your-project" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select options={[{ value: 'active' }, { value: 'paused' }, { value: 'archived' }]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={taskOpen}
        title="创建任务"
        onCancel={() => setTaskOpen(false)}
        onOk={() => taskForm.submit()}
        confirmLoading={createTask.isPending}
      >
        <Form
          layout="vertical"
          form={taskForm}
          onFinish={(values: TaskFormValues) =>
            createTask.mutate({
              title: values.title,
              description: values.description,
              status: values.status,
              priority: values.priority,
              assignee: values.assignee,
              start_date: values.start_date ? values.start_date.format('YYYY-MM-DD') : undefined,
              due_date: values.due_date ? values.due_date.format('YYYY-MM-DD') : undefined,
              estimated_days: values.estimated_days,
            })
          }
          initialValues={{ status: 'todo', priority: 'medium' }}
        >
          <Form.Item name="title" label="任务标题" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="status" label="状态">
                <Select options={[{ value: 'todo' }, { value: 'in_progress' }, { value: 'done' }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="priority" label="优先级">
                <Select options={[{ value: 'low' }, { value: 'medium' }, { value: 'high' }]} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name="start_date" label="开始日期">
                <DatePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="due_date" label="截止日期">
                <DatePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="estimated_days" label="工期(天)">
                <InputNumber min={0} precision={0} style={{ width: '100%' }} placeholder="自动估算" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="assignee" label="负责人">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
