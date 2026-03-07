import { Alert, Card, Empty, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import type { CSSProperties } from 'react';
import type { PipelineArtifact } from '../../types';
import PipelineArtifactTemplatePreview from '../PipelineArtifactTemplatePreview';
import { formatFileSize, stageLabel } from './presentation';

const PREVIEW_SANDBOX = 'allow-same-origin';
const FRAME_STYLE: CSSProperties = {
  width: '100%',
  minHeight: 460,
  border: '1px solid #e2e8f0',
  borderRadius: 14,
  background: '#fff',
};
const IMAGE_STYLE: CSSProperties = {
  width: '100%',
  borderRadius: 14,
  border: '1px solid #e2e8f0',
  background: '#fff',
};

type Props = {
  apiBase: string;
  selectedArtifact?: PipelineArtifact;
  previewVisible: boolean;
  previewUrl?: string;
  artifactContent?: string;
  artifactLoading: boolean;
  artifactLoadFailed: boolean;
};

function resolvePreviewHref(apiBase: string, previewUrl?: string) {
  if (!previewUrl) {
    return '';
  }
  if (/^https?:\/\//.test(previewUrl)) {
    return previewUrl;
  }
  return `${apiBase}${previewUrl}`;
}

function PreviewGuardNotice({ href, linkLabel }: { href: string; linkLabel: string }) {
  return (
    <Alert
      showIcon
      type="info"
      title="安全预览模式"
      description="HTML 预览已启用沙箱、延迟加载与无来源引用策略；如需完整查看，请在新窗口中打开。"
      action={
        <Typography.Link href={href} target="_blank" rel="noreferrer">
          {linkLabel}
        </Typography.Link>
      }
    />
  );
}

export default function PipelineArtifactPreviewPanel({
  apiBase,
  selectedArtifact,
  previewVisible,
  previewUrl,
  artifactContent,
  artifactLoading,
  artifactLoadFailed,
}: Props) {
  const artifactPreviewHref = resolvePreviewHref(apiBase, selectedArtifact?.preview_url);
  const pipelinePreviewHref = resolvePreviewHref(apiBase, previewUrl);

  return (
    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
      <Card title="阶段产物预览" size="small" style={{ borderRadius: 16 }}>
        {selectedArtifact ? (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <Space wrap>
              <Typography.Text strong>{selectedArtifact.name}</Typography.Text>
              {selectedArtifact.stage ? <Tag>{stageLabel(selectedArtifact.stage)}</Tag> : null}
              {selectedArtifact.preview_type ? <Tag color="purple">{selectedArtifact.preview_type}</Tag> : null}
            </Space>
            <Typography.Text type="secondary">
              {selectedArtifact.path} {'·'} {formatFileSize(selectedArtifact.size_bytes)} {'·'} {dayjs(selectedArtifact.updated_at).format('YYYY-MM-DD HH:mm:ss')}
            </Typography.Text>

            {selectedArtifact.preview_type === 'html' && artifactPreviewHref ? (
              <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                <PreviewGuardNotice href={artifactPreviewHref} linkLabel="新窗口打开产物" />
                <iframe
                  title="artifact-preview"
                  src={artifactPreviewHref}
                  sandbox={PREVIEW_SANDBOX}
                  referrerPolicy="no-referrer"
                  loading="lazy"
                  style={FRAME_STYLE}
                />
              </Space>
            ) : null}

            {selectedArtifact.preview_type === 'image' && artifactPreviewHref ? (
              <img
                src={artifactPreviewHref}
                alt={selectedArtifact.name}
                loading="lazy"
                referrerPolicy="no-referrer"
                style={IMAGE_STYLE}
              />
            ) : null}

            {selectedArtifact.preview_type === 'markdown' || selectedArtifact.preview_type === 'text' ? (
              artifactLoading ? (
                <Typography.Text type="secondary">预览加载中...</Typography.Text>
              ) : artifactLoadFailed ? (
                <Alert
                  showIcon
                  type="error"
                  title="预览内容加载失败"
                  description="请稍后重试，或直接打开输出目录查看原始文件。"
                />
              ) : (
                <PipelineArtifactTemplatePreview artifact={selectedArtifact} content={artifactContent || ''} />
              )
            ) : null}

            {selectedArtifact.preview_type === 'binary' ? (
              <Alert
                showIcon
                type="warning"
                title="当前产物为二进制文件"
                description="请通过输出目录或本地文件系统打开。"
              />
            ) : null}
          </Space>
        ) : (
          <Empty description="请选择某个阶段产物进行预览" />
        )}
      </Card>

      {previewVisible && pipelinePreviewHref ? (
        <Card title="统一页面预览" size="small" style={{ borderRadius: 16 }}>
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <PreviewGuardNotice href={pipelinePreviewHref} linkLabel="新窗口打开页面" />
            <iframe
              title="pipeline-preview"
              src={pipelinePreviewHref}
              sandbox={PREVIEW_SANDBOX}
              referrerPolicy="no-referrer"
              loading="lazy"
              style={FRAME_STYLE}
            />
          </Space>
        </Card>
      ) : null}
    </Space>
  );
}
