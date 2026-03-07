import { Button, Descriptions, Modal, Progress, Space, Tag, Tabs, Typography } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import type { PipelineCompletion, PipelineRun } from '../../types';

type Props = {
  open: boolean;
  activeTab: string;
  onTabChange: (key: string) => void;
  onClose: () => void;
  selectedRun?: PipelineRun;
  completionData?: PipelineCompletion;
  stageBoardContent: ReactNode;
  previewContent: ReactNode;
  executionContent: ReactNode;
  agentContent: ReactNode;
  onRetry?: () => void;
  retryLoading?: boolean;
};

export default function PipelineRunDetailsModal({
  open,
  activeTab,
  onTabChange,
  onClose,
  selectedRun,
  completionData,
  stageBoardContent,
  previewContent,
  executionContent,
  agentContent,
  onRetry,
  retryLoading,
}: Props) {
  if (!selectedRun) {
    return null;
  }

  const tabs = [
    {
      key: 'overview',
      label: '概览',
      children: (
        <Space orientation="vertical" size="large" style={{ width: '100%' }}>
          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="运行 ID">{selectedRun.id}</Descriptions.Item>
            <Descriptions.Item label="变更批次">{selectedRun.change_batch_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="项目目录">{selectedRun.project_dir || '-'}</Descriptions.Item>
            <Descriptions.Item label="运行模式">{formatRunMode(selectedRun)}</Descriptions.Item>
            <Descriptions.Item label="多模态素材数">{selectedRun.multimodal_assets?.length ?? 0}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{dayjs(selectedRun.updated_at).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
            <Descriptions.Item label="产物总数">{completionData?.artifacts?.length ?? 0}</Descriptions.Item>
            <Descriptions.Item label="完成清单">{completionData?.checklist?.length ?? 0}</Descriptions.Item>
            <Descriptions.Item label="输出目录" span={2}>{completionData?.output_dir || '-'}</Descriptions.Item>
          </Descriptions>

          {selectedRun.status === 'failed' && onRetry ? (
            <Button danger onClick={onRetry} loading={retryLoading}>
              {'重试失败运行'}
            </Button>
          ) : null}
        </Space>
      ),
    },
    { key: 'stages', label: '阶段产物', children: stageBoardContent },
    { key: 'preview', label: '产物预览', children: previewContent },
    { key: 'execution', label: '执行轨迹', children: executionContent },
    { key: 'agent', label: 'Agent 轨迹', children: agentContent },
  ];

  return (
    <Modal
      open={open}
      onCancel={onClose}
      width={1180}
      destroyOnHidden={false}
      title={'运行详情'}
      footer={[
        <Button key="close" onClick={onClose}>
          {'关闭'}
        </Button>,
      ]}
    >
      <Space orientation="vertical" size="large" style={{ width: '100%' }} data-testid="pipeline-run-details-modal">
        <Space orientation="vertical" size={8} style={{ width: '100%' }}>
          <Space wrap>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {'运行详情'}
            </Typography.Title>
            <Tag color={statusColor(selectedRun.status)}>{selectedRun.status}</Tag>
            <Tag>{selectedRun.stage}</Tag>
            {selectedRun.llm_enhanced_loop ? <Tag color="purple">LLM {'闭环'}</Tag> : null}
            {selectedRun.full_cycle ? <Tag color="cyan">full-cycle</Tag> : null}
            {selectedRun.step_by_step ? <Tag color="blue">step-by-step</Tag> : null}
            {selectedRun.simulate ? <Tag color="orange">simulate</Tag> : null}
          </Space>
          <Typography.Text type="secondary">{selectedRun.prompt}</Typography.Text>
          <Progress percent={selectedRun.progress} strokeColor={{ from: '#0ea5e9', to: '#7c3aed' }} />
        </Space>

        <Tabs activeKey={activeTab} onChange={onTabChange} items={tabs} />
      </Space>
    </Modal>
  );
}

function statusColor(status?: string) {
  switch (status) {
    case 'completed':
      return 'green';
    case 'failed':
      return 'red';
    case 'queued':
      return 'orange';
    default:
      return 'blue';
  }
}

function formatRunMode(run: PipelineRun) {
  if (run.step_by_step) {
    return 'step-by-step';
  }
  if (run.full_cycle) {
    return 'full-cycle';
  }
  if (run.simulate) {
    return 'simulate';
  }
  return 'super-dev';
}
