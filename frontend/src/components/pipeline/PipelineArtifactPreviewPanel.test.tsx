import { render, screen } from '@testing-library/react';
import PipelineArtifactPreviewPanel from './PipelineArtifactPreviewPanel';

describe('PipelineArtifactPreviewPanel', () => {
  it('renders sandboxed html previews with external-open links', () => {
    render(
      <PipelineArtifactPreviewPanel
        apiBase="http://localhost:8080"
        selectedArtifact={{
          name: 'design-preview',
          path: 'output/design-preview.html',
          kind: 'html',
          size_bytes: 256,
          updated_at: '2026-03-08T00:00:00Z',
          preview_url: '/api/pipeline/runs/run-1/preview/design-preview.html',
          preview_type: 'html',
          stage: 'design',
        }}
        previewVisible
        previewUrl="/api/pipeline/runs/run-1/preview/index.html"
        artifactContent=""
        artifactLoading={false}
        artifactLoadFailed={false}
      />,
    );

    const artifactFrame = screen.getByTitle('artifact-preview');
    expect(artifactFrame).toHaveAttribute('sandbox', 'allow-same-origin');
    expect(artifactFrame).toHaveAttribute('referrerpolicy', 'no-referrer');
    expect(artifactFrame).toHaveAttribute('loading', 'lazy');
    expect(artifactFrame).toHaveAttribute('src', 'http://localhost:8080/api/pipeline/runs/run-1/preview/design-preview.html');

    const pipelineFrame = screen.getByTitle('pipeline-preview');
    expect(pipelineFrame).toHaveAttribute('sandbox', 'allow-same-origin');
    expect(pipelineFrame).toHaveAttribute('referrerpolicy', 'no-referrer');
    expect(pipelineFrame).toHaveAttribute('loading', 'lazy');
    expect(pipelineFrame).toHaveAttribute('src', 'http://localhost:8080/api/pipeline/runs/run-1/preview/index.html');

    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(2);
    expect(links[0]).toHaveAttribute('href', 'http://localhost:8080/api/pipeline/runs/run-1/preview/design-preview.html');
    expect(links[1]).toHaveAttribute('href', 'http://localhost:8080/api/pipeline/runs/run-1/preview/index.html');

    expect(screen.getAllByRole('alert')).toHaveLength(2);
  });
});
