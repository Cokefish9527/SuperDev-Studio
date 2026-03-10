import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import DeliveryHandoffCard from './DeliveryHandoffCard';

const baseRun = {
  id: 'run-1',
  project_id: 'project-1',
  prompt: 'build notebook',
  status: 'completed',
  progress: 100,
  stage: 'done',
  created_at: '2026-03-11T00:00:00Z',
  updated_at: '2026-03-11T00:00:00Z',
};

const baseCompletion = {
  run_id: 'run-1',
  status: 'completed',
  output_dir: 'D:/Work/output',
  checklist: [],
  stages: [],
  preview_url: '/api/pipeline/runs/run-1/preview/index.html',
  artifacts: [
    {
      name: 'Frontend preview',
      path: 'output/frontend/index.html',
      kind: 'frontend',
      size_bytes: 128,
      updated_at: '2026-03-11T00:00:00Z',
      preview_url: '/api/pipeline/runs/run-1/preview/frontend/index.html',
      preview_type: 'html',
      stage: 'output',
    },
    {
      name: 'Quality gate report',
      path: 'output/superdev-studio-quality-gate.md',
      kind: 'markdown',
      size_bytes: 256,
      updated_at: '2026-03-11T00:00:00Z',
      preview_url: '/api/pipeline/runs/run-1/preview/superdev-studio-quality-gate.md',
      preview_type: 'markdown',
      stage: 'superdev',
    },
    {
      name: 'Red-team report',
      path: 'output/superdev-studio-redteam.md',
      kind: 'markdown',
      size_bytes: 256,
      updated_at: '2026-03-11T00:00:00Z',
      preview_url: '/api/pipeline/runs/run-1/preview/superdev-studio-redteam.md',
      preview_type: 'markdown',
      stage: 'superdev',
    },
    {
      name: 'Execution plan',
      path: 'output/superdev-studio-execution-plan.md',
      kind: 'markdown',
      size_bytes: 256,
      updated_at: '2026-03-11T00:00:00Z',
      preview_url: '/api/pipeline/runs/run-1/preview/superdev-studio-execution-plan.md',
      preview_type: 'markdown',
      stage: 'design',
    },
  ],
};

describe('DeliveryHandoffCard', () => {
  it('shows a ready handoff state when preview, quality, and package are complete', () => {
    render(
      <DeliveryHandoffCard
        run={baseRun}
        completion={baseCompletion}
        events={[
          {
            id: 1,
            run_id: 'run-1',
            stage: 'lifecycle-quality',
            status: 'completed',
            message: 'Quality gate passed on iteration 1',
            created_at: '2026-03-11T00:00:00Z',
          },
        ]}
        previewSessions={[
          {
            id: 'preview-1',
            project_id: 'project-1',
            pipeline_run_id: 'run-1',
            preview_url: '/api/pipeline/runs/run-1/preview/index.html',
            preview_type: 'html',
            title: 'Final preview',
            source_key: 'preview:1',
            status: 'accepted',
            reviewer_note: 'Looks good',
            created_at: '2026-03-11T00:00:00Z',
            updated_at: '2026-03-11T00:00:00Z',
          },
        ]}
        approvalGates={[]}
        residualItems={[]}
        apiBase="http://localhost:8080"
      />,
    );

    expect(screen.getByTestId('delivery-handoff-alert')).toHaveTextContent('Release handoff is ready');
    expect(screen.getByText('Quality gate passed on iteration 1')).toBeInTheDocument();
    expect(screen.getByText('Artifacts')).toBeInTheDocument();
    expect(screen.getByText('4 files')).toBeInTheDocument();
    expect(screen.getByTestId('delivery-handoff-acceptance')).toHaveTextContent('Ready for sign-off');
    expect(screen.getByTestId('delivery-handoff-acceptance')).toHaveTextContent('Review the latest final preview and confirm the page matches the approved requirement.');
    expect(screen.getByTestId('delivery-handoff-local-preview')).toHaveTextContent('python -m http.server 4173 --directory "D:/Work/output"');
    expect(screen.getByTestId('delivery-handoff-local-preview')).toHaveTextContent('http://127.0.0.1:4173/frontend/index.html');
    expect(screen.getByTestId('delivery-handoff-local-preview')).toHaveTextContent('output/frontend/index.html');
    expect(screen.getAllByRole('button').length).toBeGreaterThanOrEqual(4);
  });

  it('shows a blocked handoff state when open approvals or residuals still exist', () => {
    render(
      <DeliveryHandoffCard
        run={baseRun}
        completion={baseCompletion}
        events={[]}
        previewSessions={[
          {
            id: 'preview-1',
            project_id: 'project-1',
            pipeline_run_id: 'run-1',
            preview_url: '/api/pipeline/runs/run-1/preview/index.html',
            preview_type: 'html',
            title: 'Final preview',
            source_key: 'preview:1',
            status: 'generated',
            created_at: '2026-03-11T00:00:00Z',
            updated_at: '2026-03-11T00:00:00Z',
          },
        ]}
        approvalGates={[
          {
            id: 'gate-1',
            project_id: 'project-1',
            pipeline_run_id: 'run-1',
            gate_type: 'tool_governance',
            title: 'Manual approval',
            detail: 'need approval',
            source_key: 'gate:1',
            status: 'open',
            created_at: '2026-03-11T00:00:00Z',
            updated_at: '2026-03-11T00:00:00Z',
          },
        ]}
        residualItems={[
          {
            id: 'residual-1',
            project_id: 'project-1',
            pipeline_run_id: 'run-1',
            stage: 'quality',
            category: 'quality',
            severity: 'high',
            summary: 'Need more tests',
            evidence: 'tests missing',
            suggested_command: 'rerun',
            source_key: 'residual:1',
            status: 'open',
            created_at: '2026-03-11T00:00:00Z',
            updated_at: '2026-03-11T00:00:00Z',
          },
        ]}
        apiBase="http://localhost:8080"
      />,
    );

    expect(screen.getByTestId('delivery-handoff-alert')).toHaveTextContent('Release handoff is blocked');
    expect(screen.getAllByText('1 approval gate(s) still need human review.').length).toBeGreaterThan(0);
    expect(screen.getByText('1 residual item(s) still need follow-up.')).toBeInTheDocument();
    expect(screen.getByTestId('delivery-handoff-acceptance')).toHaveTextContent('Resolve the blocked checks before requesting final sign-off.');
    expect(screen.getByTestId('delivery-handoff-local-preview')).toHaveTextContent('python -m http.server 4173 --directory "D:/Work/output"');
  });
});
