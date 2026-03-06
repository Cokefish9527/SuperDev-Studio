import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import PipelinePage from './PipelinePage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      listRuns: vi.fn(),
      getRun: vi.fn(),
      getRunCompletion: vi.fn(),
      listRunEvents: vi.fn(),
      startPipeline: vi.fn(),
      retryPipeline: vi.fn(),
    },
  };
});

describe('PipelinePage', () => {
  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
    vi.mocked(apiClient.listRuns).mockResolvedValue([]);
    vi.mocked(apiClient.getRun).mockResolvedValue({
      id: 'run-1',
      project_id: 'project-1',
      prompt: 'placeholder',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.listRunEvents).mockResolvedValue([]);
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-1',
      status: 'queued',
      output_dir: 'D:/Work/output',
      checklist: [],
      artifacts: [],
    });
    vi.mocked(apiClient.retryPipeline).mockResolvedValue({
      id: 'run-retry',
      project_id: 'project-1',
      prompt: 'retry',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
  });

  it('submits dynamic context and writeback options', async () => {
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-1',
      project_id: 'project-1',
      prompt: '实现知识库检索增强',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.type(
      screen.getByPlaceholderText('例如：实现一个支持知识库检索和项目任务管理的开发协作平台'),
      '实现知识库检索增强',
    );
    await userEvent.click(screen.getByRole('button', { name: '启动流水线' }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          project_id: 'project-1',
          prompt: '实现知识库检索增强',
          simulate: true,
          context_mode: 'auto',
          context_dynamic: true,
          memory_writeback: true,
        }),
      );
    });
  });

  it('submits full cycle options and forces real mode', async () => {
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-full-cycle',
      project_id: 'project-1',
      prompt: '全流程交付需求',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.click(screen.getByRole('switch', { name: '一键全流程交付' }));
    await userEvent.clear(screen.getByRole('spinbutton', { name: '开发-单测-修复迭代次数' }));
    await userEvent.type(screen.getByRole('spinbutton', { name: '开发-单测-修复迭代次数' }), '4');
    await userEvent.clear(screen.getByPlaceholderText('例如：实现一个支持知识库检索和项目任务管理的开发协作平台'));
    await userEvent.type(
      screen.getByPlaceholderText('例如：实现一个支持知识库检索和项目任务管理的开发协作平台'),
      '全流程交付需求',
    );
    await userEvent.click(screen.getByRole('button', { name: '启动流水线' }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: '全流程交付需求',
          full_cycle: true,
          iteration_limit: 4,
          simulate: false,
        }),
      );
    });
  }, 12000);

  it('submits step-by-step mode and forces real mode', async () => {
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-step',
      project_id: 'project-1',
      prompt: '逐步开发需求',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.click(screen.getByRole('switch', { name: '按 super-dev 原生步骤执行' }));
    await userEvent.clear(screen.getByPlaceholderText('例如：实现一个支持知识库检索和项目任务管理的开发协作平台'));
    await userEvent.type(
      screen.getByPlaceholderText('例如：实现一个支持知识库检索和项目任务管理的开发协作平台'),
      '逐步开发需求',
    );
    await userEvent.click(screen.getByRole('button', { name: '启动流水线' }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: '逐步开发需求',
          step_by_step: true,
          simulate: false,
        }),
      );
    });
  }, 12000);

  it('retries a failed run from run details', async () => {
    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-failed',
        project_id: 'project-1',
        prompt: '执行失败的需求',
        status: 'failed',
        progress: 100,
        stage: 'super-dev',
        created_at: '2026-03-05T00:00:00Z',
        updated_at: '2026-03-05T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.getRun).mockImplementation(async (runId: string) => ({
      id: runId,
      project_id: 'project-1',
      prompt: runId === 'run-failed' ? '执行失败的需求' : '执行失败的需求',
      retry_of: runId === 'run-retry' ? 'run-failed' : undefined,
      status: runId === 'run-retry' ? 'queued' : 'failed',
      progress: runId === 'run-retry' ? 0 : 100,
      stage: runId === 'run-retry' ? 'queued' : 'super-dev',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    }));
    vi.mocked(apiClient.retryPipeline).mockResolvedValue({
      id: 'run-retry',
      project_id: 'project-1',
      prompt: '执行失败的需求',
      retry_of: 'run-failed',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-failed',
      status: 'failed',
      output_dir: 'D:/Work/output',
      checklist: [{ key: 'run-status', title: '流水线状态', status: 'failed', note: 'failed' }],
      artifacts: [],
      preview_url: '/api/pipeline/runs/run-failed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '重试失败运行' })).toBeInTheDocument();
    }, { timeout: 6000 });

    await userEvent.click(screen.getByRole('button', { name: '重试失败运行' }));

    await waitFor(() => {
      expect(apiClient.retryPipeline).toHaveBeenCalled();
      const [runId] = vi.mocked(apiClient.retryPipeline).mock.calls[0];
      expect(runId).toBe('run-failed');
    }, { timeout: 6000 });
  }, 12000);

  it('shows completion checklist and preview button for finished runs', async () => {
    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-completed',
        project_id: 'project-1',
        prompt: '已完成运行',
        status: 'completed',
        progress: 100,
        stage: 'done',
        created_at: '2026-03-05T00:00:00Z',
        updated_at: '2026-03-05T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.getRun).mockResolvedValue({
      id: 'run-completed',
      project_id: 'project-1',
      prompt: '已完成运行',
      status: 'completed',
      progress: 100,
      stage: 'done',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-completed',
      status: 'completed',
      output_dir: 'D:/Work/output',
      checklist: [
        { key: 'run-status', title: '流水线状态', status: 'completed', note: 'completed' },
        { key: 'prd', title: 'PRD 文档', status: 'completed' },
      ],
      artifacts: [{ name: 'PRD 文档', path: 'output/demo-prd.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z' }],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getAllByText('PRD 文档').length).toBeGreaterThan(0);
      expect(screen.getByRole('button', { name: '预览页面' })).toBeInTheDocument();
    });
  });
});
