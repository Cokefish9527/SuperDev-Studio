import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
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
  afterEach(() => {
    vi.unstubAllGlobals();
  });

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
      stages: [],
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
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      text: async () => '',
    } as Response));
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
      stages: [],
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

  it('renders structured template preview for loop artifacts', async () => {
    const markdown = `# 构思增强稿

## 文档元数据
| 字段 | 值 |
| --- | --- |
| run_id | run-completed |
| stage | llm-idea |
| template_kind | concept |
| change_id | change-001 |
| generated_at | 2026-03-05T00:00:00Z |
| multimodal_assets | 1 |

## 输入快照
### 输入需求
为阶段产物提供更严格的模板化预览。

### 参考素材
- https://example.com/reference.png

## 执行摘要
构思阶段已形成可继续推进的结构化结论。

## 用户价值
- 让项目成员快速理解当前阶段结论。

## 验收检查点
- 页面必须稳定显示固定章节。

## 下一步动作
- 继续推进设计复核。

## LLM 原始输出
\`\`\`text
{"summary":"构思阶段已形成可继续推进的结构化结论。"}
\`\`\``;

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
      checklist: [{ key: 'run-status', title: '流水线状态', status: 'completed', note: 'completed' }],
      artifacts: [{ name: '构思增强稿', path: 'output/demo-concept.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-concept.md', preview_type: 'markdown', stage: 'idea' }],
      stages: [{ key: 'idea', title: '构思', status: 'completed', artifacts: [{ name: '构思增强稿', path: 'output/demo-concept.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-concept.md', preview_type: 'markdown', stage: 'idea' }] }],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      text: async () => markdown,
    } as Response));

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByText('模板化预览')).toBeInTheDocument();
      expect(screen.getAllByText('\u8fd0\u884c ID').length).toBeGreaterThan(0);
      expect(screen.getAllByText('\u8f93\u5165\u5feb\u7167').length).toBeGreaterThan(0);
      expect(screen.getAllByText('\u9a8c\u6536\u68c0\u67e5\u70b9').length).toBeGreaterThan(0);
      expect(screen.getByText('构思阶段已形成可继续推进的结构化结论。')).toBeInTheDocument();
    });
  });

  it('paginates long timeline events and supports back to top', async () => {
    const scrollToSpy = vi.fn();
    vi.stubGlobal('scrollTo', scrollToSpy);

    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-completed',
        project_id: 'project-1',
        prompt: '????????',
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
      prompt: '????????',
      status: 'completed',
      progress: 100,
      stage: 'done',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.listRunEvents).mockResolvedValue(
      Array.from({ length: 12 }, (_, index) => ({
        id: index + 1,
        run_id: 'run-completed',
        stage: 'superdev',
        status: 'completed',
        message: `?? ${index + 1}`,
        created_at: `2026-03-05T00:${String(index).padStart(2, '0')}:00Z`,
      })),
    );
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-completed',
      status: 'completed',
      output_dir: 'D:/Work/output',
      checklist: [{ key: 'run-status', title: '?????', status: 'completed', note: 'completed' }],
      artifacts: [],
      stages: [],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByText('?? 1-8 / 12')).toBeInTheDocument();
      expect(screen.getByText('?? 8')).toBeInTheDocument();
    });

    expect(screen.queryByText('?? 9')).not.toBeInTheDocument();

    const paginationBar = screen.getByText('? 1 / 2 ?').closest('div');
    expect(paginationBar).not.toBeNull();
    await userEvent.click(within(paginationBar as HTMLElement).getAllByRole('button')[1]);

    await waitFor(() => {
      expect(screen.getByText('?? 9-12 / 12')).toBeInTheDocument();
      expect(screen.getByText('?? 9')).toBeInTheDocument();
    });

    const summaryBar = screen.getByText('?? 9-12 / 12').closest('.ant-space');
    expect(summaryBar).not.toBeNull();
    await userEvent.click(within(summaryBar as HTMLElement).getByRole('button'));

    expect(scrollToSpy).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' });
  });

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
      artifacts: [{ name: 'PRD ??', path: 'output/demo-prd.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-prd.md', preview_type: 'markdown', stage: 'design' }],
      stages: [{ key: 'design', title: '??', status: 'completed', artifacts: [{ name: 'PRD ??', path: 'output/demo-prd.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-prd.md', preview_type: 'markdown', stage: 'design' }] }],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getAllByText('PRD 文档').length).toBeGreaterThan(0);
      expect(screen.getByRole('button', { name: '预览页面' })).toBeInTheDocument();
    });
  });
});
