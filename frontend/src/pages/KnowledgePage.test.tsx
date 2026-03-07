import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import KnowledgePage from './KnowledgePage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      listKnowledgeDocuments: vi.fn(),
      searchKnowledge: vi.fn(),
      createKnowledgeDocument: vi.fn(),
    },
  };
});

describe('KnowledgePage', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
    vi.mocked(apiClient.listKnowledgeDocuments).mockResolvedValue([
      {
        id: 'doc-1',
        project_id: 'project-1',
        title: 'Design Spec',
        source: 'Confluence',
        content: 'doc body',
        created_at: '2026-03-08T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.searchKnowledge).mockResolvedValue(
      Array.from({ length: 6 }, (_, index) => ({
        id: index + 1,
        document_id: 'doc-1',
        project_id: 'project-1',
        chunk_index: index + 1,
        content: `chunk-${index + 1}`,
        created_at: '2026-03-08T00:00:00Z',
      })),
    );
  });

  it('paginates long search results and supports back to top', async () => {
    const scrollToSpy = vi.fn();
    vi.stubGlobal('scrollTo', scrollToSpy);

    renderWithProviders(<KnowledgePage />);

    const searchInput = within(screen.getByTestId('knowledge-search-box')).getByRole('searchbox') as HTMLInputElement;
    await userEvent.type(searchInput, 'context');
    await userEvent.keyboard('{Enter}');

    await waitFor(() => {
      expect(apiClient.searchKnowledge).toHaveBeenCalledWith('project-1', 'context', 8);
      expect(screen.getByTestId('knowledge-hit-count')).toHaveTextContent('6');
      expect(screen.getByTestId('knowledge-summary')).toHaveTextContent('1-4');
      expect(screen.getByText('chunk-4')).toBeInTheDocument();
    });

    expect(screen.queryByText('chunk-5')).not.toBeInTheDocument();

    await userEvent.click(screen.getByTestId('knowledge-next-page'));

    await waitFor(() => {
      expect(screen.getByTestId('knowledge-summary')).toHaveTextContent('5-6');
      expect(screen.getByText('chunk-5')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('knowledge-back-top'));
    expect(scrollToSpy).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' });
  });
});
