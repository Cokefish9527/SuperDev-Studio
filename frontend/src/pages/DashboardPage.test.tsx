import { screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DashboardPage from './DashboardPage';
import { renderWithProviders } from '../test/render';
import { apiClient } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      getDashboard: vi.fn(),
    },
  };
});

describe('DashboardPage', () => {
  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
  });

  it('renders dashboard stats and recent runs', async () => {
    vi.mocked(apiClient.getDashboard).mockResolvedValue({
      stats: {
        projects: 2,
        tasks: 5,
        runs: 3,
        memories: 12,
        docs: 4,
      },
      recent_runs: [
        {
          id: 'run-1',
          project_id: 'project-1',
          prompt: 'Build auth service',
          status: 'completed',
          progress: 100,
          stage: 'done',
          created_at: '2026-03-05T00:00:00Z',
          updated_at: '2026-03-05T00:00:00Z',
        },
      ],
    });

    renderWithProviders(<DashboardPage />);

    await waitFor(() => {
      expect(screen.getByText('工作台总览')).toBeInTheDocument();
      expect(screen.getByText('Build auth service')).toBeInTheDocument();
      expect(screen.getByText('项目总数')).toBeInTheDocument();
    });
  });
});
