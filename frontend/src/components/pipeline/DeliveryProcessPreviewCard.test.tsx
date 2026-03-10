import { afterEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DeliveryProcessPreviewCard from './DeliveryProcessPreviewCard';
import { renderWithProviders } from '../../test/render';

describe('DeliveryProcessPreviewCard', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows process documents and opens the inline final preview', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(async (input: RequestInfo | URL) => {
        const url = String(input);
        return {
          ok: true,
          text: async () => (url.includes('quality-gate') ? 'Quality gate passed' : 'Execution report body'),
        } as Response;
      }),
    );

    renderWithProviders(
      <DeliveryProcessPreviewCard
        apiBase="http://localhost:8080"
        completion={{
          run_id: 'run-1',
          status: 'completed',
          output_dir: 'D:/Work/output',
          checklist: [],
          stages: [],
          preview_url: '/api/pipeline/runs/run-1/preview/index.html',
          artifacts: [
            {
              name: 'superdev-studio-quality-gate.md',
              path: 'output/superdev-studio-quality-gate.md',
              kind: 'markdown',
              size_bytes: 320,
              updated_at: '2026-03-11T00:00:00Z',
              preview_url: '/api/pipeline/runs/run-1/preview/superdev-studio-quality-gate.md',
              preview_type: 'markdown',
              stage: 'output',
            },
            {
              name: 'superdev-studio-task-execution.md',
              path: 'output/superdev-studio-task-execution.md',
              kind: 'text',
              size_bytes: 280,
              updated_at: '2026-03-11T00:00:10Z',
              preview_url: '/api/pipeline/runs/run-1/preview/superdev-studio-task-execution.md',
              preview_type: 'text',
              stage: 'output',
            },
          ],
        }}
      />,
    );

    const card = screen.getByTestId('delivery-process-preview-card');
    expect(card).toHaveTextContent('Process docs & final preview');
    expect(card).toHaveTextContent('superdev-studio-quality-gate.md');
    expect(card).toHaveTextContent('superdev-studio-task-execution.md');
    expect(card).toHaveTextContent('Open final preview');

    await waitFor(() => {
      expect(screen.getByText('Quality gate passed')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole('button', { name: 'superdev-studio-task-execution.md' }));

    await waitFor(() => {
      expect(screen.getByText('Execution report body')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('delivery-process-toggle-final-preview'));

    expect(screen.getByTitle('delivery-process-final-preview')).toHaveAttribute(
      'src',
      'http://localhost:8080/api/pipeline/runs/run-1/preview/index.html',
    );
  });

  it('shows a pending state when process docs and final preview are unavailable', () => {
    renderWithProviders(
      <DeliveryProcessPreviewCard
        apiBase="http://localhost:8080"
        completion={{
          run_id: 'run-2',
          status: 'running',
          output_dir: 'D:/Work/output',
          checklist: [],
          stages: [],
          artifacts: [],
        }}
      />,
    );

    const card = screen.getByTestId('delivery-process-preview-card');
    expect(card).toHaveTextContent('No process documents are ready yet.');
    expect(card).toHaveTextContent('The final preview is not ready yet.');
  });
});
