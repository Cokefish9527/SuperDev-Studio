import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ContextPage from './ContextPage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      buildContextPack: vi.fn(),
    },
  };
});

describe('ContextPage', () => {
  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
  });

  it('submits query and renders context pack summary', async () => {
    vi.mocked(apiClient.buildContextPack).mockResolvedValue({
      query: 'rollback strategy',
      token_budget: 1200,
      estimated_tokens: 240,
      summary: '记忆模块提要：\n- 优先实现失败重试策略',
      memories: [
        {
          id: 'm1',
          project_id: 'project-1',
          role: 'note',
          content: '优先实现失败重试策略',
          tags: ['pipeline'],
          importance: 0.9,
          created_at: '2026-03-05T00:00:00Z',
        },
      ],
      knowledge: [],
    });

    renderWithProviders(<ContextPage />);

    await userEvent.type(screen.getByPlaceholderText('例如：我现在要实现流水线错误回滚，给我最相关上下文'), 'rollback strategy');
    await userEvent.click(screen.getByRole('button', { name: '生成上下文包' }));

    await waitFor(() => {
      expect(apiClient.buildContextPack).toHaveBeenCalledWith(
        'project-1',
        expect.objectContaining({ query: 'rollback strategy' }),
      );
      expect(screen.getByText('优先实现失败重试策略')).toBeInTheDocument();
    });
  });
});
