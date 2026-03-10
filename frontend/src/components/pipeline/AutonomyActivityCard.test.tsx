import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import AutonomyActivityCard from './AutonomyActivityCard';

describe('AutonomyActivityCard', () => {
  it('filters and summarizes autonomous delivery activity', () => {
    render(
      <AutonomyActivityCard
        events={[
          {
            id: 1,
            run_id: 'run-1',
            stage: 'starting',
            status: 'running',
            message: 'Pipeline started',
            created_at: '2026-03-11T00:00:00Z',
          },
          {
            id: 2,
            run_id: 'run-1',
            stage: 'auto-advance',
            status: 'log',
            message: 'Auto advance executed rerun_delivery and started run run-2',
            created_at: '2026-03-11T00:01:00Z',
          },
          {
            id: 3,
            run_id: 'run-1',
            stage: 'backlog-reconcile',
            status: 'completed',
            message: 'Residual backlog re-evaluated: closed 2 historical items.',
            created_at: '2026-03-11T00:02:00Z',
          },
          {
            id: 4,
            run_id: 'run-1',
            stage: 'lifecycle-quality',
            status: 'completed',
            message: 'Quality gate passed on iteration 2',
            created_at: '2026-03-11T00:03:00Z',
          },
          {
            id: 5,
            run_id: 'run-1',
            stage: 'done',
            status: 'completed',
            message: 'One-click lifecycle finished',
            created_at: '2026-03-11T00:04:00Z',
          },
        ]}
      />,
    );

    const card = screen.getByTestId('simple-delivery-autonomy-card');

    expect(screen.getByTestId('simple-delivery-autonomy-summary')).toHaveTextContent('Delivery reached the final completed state');
    expect(card).toHaveTextContent('Auto-advance');
    expect(card).toHaveTextContent('Backlog shrink');
    expect(card).toHaveTextContent('Latest quality');
    expect(card).toHaveTextContent('Quality gate passed on iteration 2');
    expect(card).toHaveTextContent('One-click lifecycle finished');
    expect(card).not.toHaveTextContent('Pipeline started');
  });

  it('shows an empty state when no relevant autonomous events exist', () => {
    render(<AutonomyActivityCard events={[]} />);

    expect(screen.getByText('No autonomous progress events yet')).toBeInTheDocument();
  });
});
