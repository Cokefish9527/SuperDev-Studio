import { useQuery } from '@tanstack/react-query';
import { Alert, Button, Card, Divider, Empty, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import type { CSSProperties } from 'react';
import { useEffect, useMemo, useState } from 'react';
import type { PipelineArtifact, PipelineCompletion } from '../../types';
import PipelineArtifactPreviewPanel from './PipelineArtifactPreviewPanel';

const PREVIEW_SANDBOX = 'allow-same-origin';
const FINAL_PREVIEW_FRAME_STYLE: CSSProperties = {
  width: '100%',
  minHeight: 460,
  border: '1px solid #e2e8f0',
  borderRadius: 14,
  background: '#fff',
};

type Props = {
  completion?: PipelineCompletion;
  apiBase: string;
  loading?: boolean;
};

export default function DeliveryProcessPreviewCard({ completion, apiBase, loading }: Props) {
  const processArtifacts = useMemo(() => pickProcessArtifacts(completion?.artifacts ?? []), [completion?.artifacts]);
  const [selectedArtifactPath, setSelectedArtifactPath] = useState('');
  const [finalPreviewExpanded, setFinalPreviewExpanded] = useState(false);

  useEffect(() => {
    if (!processArtifacts.length) {
      setSelectedArtifactPath('');
      return;
    }
    if (!processArtifacts.some((artifact) => artifact.path === selectedArtifactPath)) {
      setSelectedArtifactPath(processArtifacts[0].path);
    }
  }, [processArtifacts, selectedArtifactPath]);

  const selectedArtifact = useMemo(() => {
    if (!processArtifacts.length) {
      return undefined;
    }
    return processArtifacts.find((artifact) => artifact.path === selectedArtifactPath) ?? processArtifacts[0];
  }, [processArtifacts, selectedArtifactPath]);

  const artifactContentQuery = useQuery({
    queryKey: ['simple-delivery-process-artifact', selectedArtifact?.preview_url],
    queryFn: async () => {
      if (!selectedArtifact?.preview_url) {
        return '';
      }
      const response = await fetch(resolvePreviewHref(apiBase, selectedArtifact.preview_url));
      if (!response.ok) {
        throw new Error('Failed to load artifact preview');
      }
      return response.text();
    },
    enabled: Boolean(
      selectedArtifact?.preview_url &&
        (selectedArtifact.preview_type === 'markdown' || selectedArtifact.preview_type === 'text'),
    ),
    staleTime: 30000,
  });

  const finalPreviewHref = resolvePreviewHref(apiBase, completion?.preview_url);

  return (
    <Card title="Process docs & final preview" style={{ borderRadius: 20 }} loading={loading} data-testid="delivery-process-preview-card">
      {!completion ? (
        <Empty description="Delivery output is not available yet" />
      ) : (
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Alert
            showIcon
            type={finalPreviewHref ? 'success' : 'info'}
            title={buildSummaryTitle(processArtifacts.length, Boolean(finalPreviewHref))}
            description={buildSummaryDescription(completion, processArtifacts.length, Boolean(finalPreviewHref))}
          />

          <Space orientation="vertical" size={8} style={{ width: '100%' }}>
            <Space wrap>
              <Typography.Text strong>Process documents</Typography.Text>
              <Tag color="blue">{processArtifacts.length} docs</Tag>
              {completion.output_dir ? <Tag>{completion.output_dir}</Tag> : null}
            </Space>
            {!processArtifacts.length ? (
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                No process documents are ready yet. Execution reports and quality evidence will appear here once they are generated.
              </Typography.Paragraph>
            ) : (
              <Space wrap>
                {processArtifacts.map((artifact) => (
                  <Button
                    key={artifact.path}
                    size="small"
                    type={artifact.path === selectedArtifact?.path ? 'primary' : 'default'}
                    onClick={() => setSelectedArtifactPath(artifact.path)}
                  >
                    {artifact.name}
                  </Button>
                ))}
              </Space>
            )}
          </Space>

          {selectedArtifact ? (
            <PipelineArtifactPreviewPanel
              apiBase={apiBase}
              selectedArtifact={selectedArtifact}
              previewVisible={false}
              artifactContent={artifactContentQuery.data}
              artifactLoading={artifactContentQuery.isLoading}
              artifactLoadFailed={artifactContentQuery.isError}
            />
          ) : null}

          <Divider style={{ margin: '4px 0' }} />

          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <Space wrap>
              <Typography.Text strong>Final preview</Typography.Text>
              <Tag color={finalPreviewHref ? 'green' : 'default'}>{finalPreviewHref ? 'Ready' : 'Pending'}</Tag>
            </Space>

            {finalPreviewHref ? (
              <>
                <Alert
                  showIcon
                  type="success"
                  title="The latest generated page is ready to review."
                  description="Open it in a new window or inspect it inline below without leaving the simplified delivery page."
                />
                <Space wrap>
                  <Button
                    data-testid="delivery-process-open-final-preview"
                    type="primary"
                    onClick={() => window.open(finalPreviewHref, '_blank', 'noopener,noreferrer')}
                  >
                    Open final preview
                  </Button>
                  <Button
                    data-testid="delivery-process-toggle-final-preview"
                    onClick={() => setFinalPreviewExpanded((value) => !value)}
                  >
                    {finalPreviewExpanded ? 'Hide inline preview' : 'Show inline preview'}
                  </Button>
                </Space>
                {finalPreviewExpanded ? (
                  <iframe
                    title="delivery-process-final-preview"
                    src={finalPreviewHref}
                    sandbox={PREVIEW_SANDBOX}
                    referrerPolicy="no-referrer"
                    loading="lazy"
                    style={FINAL_PREVIEW_FRAME_STYLE}
                  />
                ) : null}
              </>
            ) : (
              <Alert
                showIcon
                type="info"
                title="The final preview is not ready yet."
                description="The delivery run is still generating the latest page output."
              />
            )}
          </Space>
        </Space>
      )}
    </Card>
  );
}

function pickProcessArtifacts(artifacts: PipelineArtifact[]) {
  const picked = artifacts.filter((artifact) => isProcessArtifact(artifact));
  const source = picked.length ? picked : artifacts.filter((artifact) => artifact.preview_type === 'markdown' || artifact.preview_type === 'text');
  return [...source]
    .sort((left, right) => artifactPriority(left) - artifactPriority(right) || dayjs(right.updated_at).valueOf() - dayjs(left.updated_at).valueOf())
    .slice(0, 6);
}

function isProcessArtifact(artifact: PipelineArtifact) {
  const lower = `${artifact.name} ${artifact.path}`.toLowerCase();
  if (artifact.preview_type === 'markdown' || artifact.preview_type === 'text') {
    return true;
  }
  return [
    'quality-gate',
    'redteam',
    'execution-report',
    'task-execution',
    'execution-plan',
    'release-note',
    'release-notes',
    'manifest',
    'prd',
    'architecture',
    'uiux',
  ].some((token) => lower.includes(token));
}

function artifactPriority(artifact: PipelineArtifact) {
  const lower = `${artifact.name} ${artifact.path}`.toLowerCase();
  if (lower.includes('quality-gate')) {
    return 0;
  }
  if (lower.includes('redteam')) {
    return 1;
  }
  if (lower.includes('execution-report')) {
    return 2;
  }
  if (lower.includes('task-execution')) {
    return 3;
  }
  if (lower.includes('release-note') || lower.includes('release-notes')) {
    return 4;
  }
  if (lower.includes('manifest')) {
    return 5;
  }
  return 6;
}

function buildSummaryTitle(documentCount: number, previewReady: boolean) {
  if (previewReady) {
    return `${documentCount} process document(s) ready with a final preview entry.`;
  }
  return `${documentCount} process document(s) ready; final preview still pending.`;
}

function buildSummaryDescription(completion: PipelineCompletion, documentCount: number, previewReady: boolean) {
  const parts = [
    completion.status ? `Completion ${completion.status}` : '',
    completion.output_dir ? `Output ${completion.output_dir}` : '',
    documentCount > 0 ? `${documentCount} document preview(s) available` : 'No process documents yet',
    previewReady ? 'Final preview available' : 'Final preview not available yet',
  ].filter(Boolean);
  return parts.join(' | ');
}

function resolvePreviewHref(apiBase: string, previewUrl?: string) {
  if (!previewUrl) {
    return '';
  }
  if (/^https?:\/\//.test(previewUrl)) {
    return previewUrl;
  }
  return `${apiBase}${previewUrl}`;
}
