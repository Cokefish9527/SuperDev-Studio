import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MemoryPage from './MemoryPage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      listMemories: vi.fn(),
      createMemory: vi.fn(),
    },
  };
});

describe('MemoryPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
    vi.mocked(apiClient.listMemories).mockResolvedValue(
      Array.from({ length: 8 }, (_, index) => ({
        id: `memory-${index + 1}`,
        project_id: 'project-1',
        role: index % 2 === 0 ? 'note' : 'assistant',
        content: index === 6 ? 'memory-risk-critical' : `memory-${index + 1}`,
        tags: index === 6 ? ['risk', 'priority'] : ['normal'],
        importance: 0.6 + index * 0.05,
        created_at: `2026-03-08T00:0${index}:00Z`,
      })),
    );
  });

  it('filters memory items and paginates long lists', async () => {
    renderWithProviders(<MemoryPage />);

    await waitFor(() => {
      expect(apiClient.listMemories).toHaveBeenCalledWith('project-1', 50);
      expect(screen.getByTestId('memory-summary')).toHaveTextContent('8');
      expect(screen.getByText('memory-6')).toBeInTheDocument();
    });

    expect(screen.queryByText('memory-risk-critical')).not.toBeInTheDocument();

    const pagination = document.querySelector('.ant-pagination');
    expect(pagination).not.toBeNull();
    await userEvent.click(within(pagination as HTMLElement).getByTitle('2'));

    await waitFor(() => {
      expect(screen.getByText('memory-risk-critical')).toBeInTheDocument();
    });

    const searchInput = within(screen.getByTestId('memory-search-box')).getByRole('searchbox') as HTMLInputElement;
    await userEvent.clear(searchInput);
    await userEvent.type(searchInput, 'risk');

    await waitFor(() => {
      expect(screen.getByTestId('memory-summary')).toHaveTextContent('1');
      expect(screen.getByText('memory-risk-critical')).toBeInTheDocument();
    });
  });
});
