import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import DeliveryLedgerCard from './DeliveryLedgerCard';

describe('DeliveryLedgerCard', () => {
  it('shows the attempt history and richer signals for a change batch', () => {
    render(
      <DeliveryLedgerCard
        batchId="change-1"
        batchTitle="Timeline notebook"
        mode="step_by_step"
        currentRunId="run-2"
        runSignals={{
          'run-1': {
            preview: 'rejected',
            quality: 'failed',
            openApprovals: 1,
            openResiduals: 2,
          },
          'run-2': {
            preview: 'accepted',
            quality: 'passed',
            openApprovals: 0,
            openResiduals: 0,
          },
        }}
        runs={[
          {
            id: 'run-1',
            project_id: 'project-1',
            change_batch_id: 'change-1',
            prompt: 'Initial delivery attempt',
            status: 'failed',
            progress: 100,
            stage: 'failed',
            step_by_step: true,
            created_at: '2026-03-11T00:00:00Z',
            updated_at: '2026-03-11T00:05:00Z',
          },
          {
            id: 'run-2',
            project_id: 'project-1',
            change_batch_id: 'change-1',
            prompt: 'Recovered delivery attempt',
            status: 'completed',
            progress: 100,
            stage: 'done',
            step_by_step: true,
            retry_of: 'run-1',
            created_at: '2026-03-11T00:06:00Z',
            updated_at: '2026-03-11T00:10:00Z',
          },
        ]}
      />,
    );

    const card = screen.getByTestId('simple-delivery-ledger-card');

    expect(screen.getByTestId('simple-delivery-ledger-summary')).toHaveTextContent('Timeline notebook: 2 delivery attempt(s)');
    expect(card).toHaveTextContent('Attempt 1');
    expect(card).toHaveTextContent('Attempt 2');
    expect(card).toHaveTextContent('Retried from run-1');
    expect(card).toHaveTextContent('Preview accepted');
    expect(card).toHaveTextContent('Quality passed');
    expect(card).toHaveTextContent('Preview rejected');
    expect(card).toHaveTextContent('Quality failed');
    expect(card).toHaveTextContent('Approvals 1');
    expect(card).toHaveTextContent('Residuals 2');
    expect(card).toHaveTextContent('current');
    expect(card).toHaveTextContent('latest');
  });

  it('shows an empty state when no runs exist for the change batch', () => {
    render(<DeliveryLedgerCard runs={[]} />);

    expect(screen.getByText('No delivery attempts have been recorded for this change batch yet')).toBeInTheDocument();
  });
});
