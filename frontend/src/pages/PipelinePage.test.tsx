import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import PipelinePage from './PipelinePage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

const PROMPT_PLACEHOLDER = '例如：实现一个支持知识库检索和项目任务管理的开发协作平台';
const START_BUTTON = '启动流水线';
const FULL_CYCLE_SWITCH = '一键全流程交付';
const STEP_BY_STEP_SWITCH = '按 super-dev 原生步骤执行';
const ITERATION_LABEL = '开发-单测-修复迭代次数';
const RETRY_LABEL = '重试失败运行';
const APPROVE_LABEL = '确认高风险动作并继续';
const TAB_PREVIEW = '产物预览';
const TAB_STAGES = '阶段产物';
const TAB_EXECUTION = '执行轨迹';
const PREVIEW_PAGE_LABEL = '预览页面';
const AGENT_DEFAULT_FALLBACK_ALERT = '检测到项目默认 Agent 配置已失效';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      listRuns: vi.fn(),
      getProject: vi.fn(),
      getProjectAgentBundle: vi.fn(),
      listChangeBatches: vi.fn(),
      getRun: vi.fn(),
      getRunCompletion: vi.fn(),
      listRunEvents: vi.fn(),
      getRunAgent: vi.fn(),
      listRunAgentSteps: vi.fn(),
      listRunAgentToolCalls: vi.fn(),
      listRunAgentEvidence: vi.fn(),
      listRunAgentEvaluations: vi.fn(),
      startPipeline: vi.fn(),
      retryPipeline: vi.fn(),
      resumePipeline: vi.fn(),
      approvePipelineTool: vi.fn(),
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
    vi.mocked(apiClient.listChangeBatches).mockResolvedValue([]);
    vi.mocked(apiClient.getProject).mockResolvedValue({
      id: 'project-1',
      name: 'Studio',
      description: 'test',
      repo_path: 'D:/Work/agent-demo/SuperDev-Studio',
      status: 'active',
      default_platform: 'web',
      default_frontend: 'react',
      default_backend: 'go',
      default_domain: 'saas',
      default_agent_name: 'reviewer',
      default_agent_mode: 'review',
      default_context_mode: 'auto',
      default_context_token_budget: 1200,
      default_context_max_items: 8,
      default_context_dynamic: true,
      default_memory_writeback: true,
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.getProjectAgentBundle).mockResolvedValue({
      project_id: 'project-1',
      project_dir: 'D:/Work/agent-demo/SuperDev-Studio',
      default_agent_name: 'reviewer',
      default_agent_mode: 'review',
      agents: [{ name: 'reviewer', description: 'Review agent' }],
      modes: [{ name: 'review', description: 'Review mode' }, { name: 'full_cycle', description: 'Full cycle mode' }, { name: 'step_by_step', description: 'Step mode' }],
    });
    vi.mocked(apiClient.listChangeBatches).mockResolvedValue([]);
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
    vi.mocked(apiClient.getRunAgent).mockResolvedValue(undefined as never);
    vi.mocked(apiClient.listRunAgentSteps).mockResolvedValue([]);
    vi.mocked(apiClient.listRunAgentToolCalls).mockResolvedValue([]);
    vi.mocked(apiClient.listRunAgentEvidence).mockResolvedValue([]);
    vi.mocked(apiClient.listRunAgentEvaluations).mockResolvedValue([]);
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
    vi.mocked(apiClient.resumePipeline).mockResolvedValue({
      id: 'run-resume',
      project_id: 'project-1',
      prompt: 'resume',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.approvePipelineTool).mockResolvedValue({
      id: 'run-awaiting',
      project_id: 'project-1',
      prompt: 'approve',
      status: 'running',
      progress: 88,
      stage: 'lifecycle-release',
      full_cycle: true,
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        text: async () => '',
      } as Response),
    );
  });

  it('submits dynamic context and writeback options', async () => {
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-1',
      project_id: 'project-1',
      prompt: 'context-enhanced-run',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.type(screen.getByPlaceholderText(PROMPT_PLACEHOLDER), 'context-enhanced-run');
    await userEvent.click(screen.getByRole('button', { name: START_BUTTON }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          project_id: 'project-1',
          prompt: 'context-enhanced-run',
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
      prompt: 'full-cycle-run',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.click(screen.getByRole('switch', { name: FULL_CYCLE_SWITCH }));
    await userEvent.clear(screen.getByRole('spinbutton', { name: ITERATION_LABEL }));
    await userEvent.type(screen.getByRole('spinbutton', { name: ITERATION_LABEL }), '4');
    await userEvent.clear(screen.getByPlaceholderText(PROMPT_PLACEHOLDER));
    await userEvent.type(screen.getByPlaceholderText(PROMPT_PLACEHOLDER), 'full-cycle-run');
    await userEvent.click(screen.getByRole('button', { name: START_BUTTON }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: 'full-cycle-run',
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
      prompt: 'step-run',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.click(screen.getByRole('switch', { name: STEP_BY_STEP_SWITCH }));
    await userEvent.clear(screen.getByPlaceholderText(PROMPT_PLACEHOLDER));
    await userEvent.type(screen.getByPlaceholderText(PROMPT_PLACEHOLDER), 'step-run');
    await userEvent.click(screen.getByRole('button', { name: START_BUTTON }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls[0];
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: 'step-run',
          step_by_step: true,
          simulate: false,
        }),
      );
    });
  }, 12000);

  it('submits step_by_step with default agent strategy', async () => {
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-agent',
      project_id: 'project-1',
      prompt: 'step-run-with-default-agent',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await userEvent.click(screen.getByRole('switch', { name: STEP_BY_STEP_SWITCH }));
    await userEvent.clear(screen.getByPlaceholderText(PROMPT_PLACEHOLDER));
    await userEvent.type(screen.getByPlaceholderText(PROMPT_PLACEHOLDER), 'step-run-with-default-agent');
    await userEvent.click(screen.getByRole('button', { name: START_BUTTON }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls.at(-1)!;
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: 'step-run-with-default-agent',
          step_by_step: true,
          agent_name: 'reviewer',
          agent_mode: 'review',
        }),
      );
    });
  }, 12000);

  it('falls back stale project agent defaults to bundle defaults and shows warning', async () => {
    vi.mocked(apiClient.getProject).mockResolvedValue({
      id: 'project-1',
      name: 'Studio',
      description: 'test',
      repo_path: 'D:/Work/agent-demo/SuperDev-Studio',
      status: 'active',
      default_platform: 'web',
      default_frontend: 'react',
      default_backend: 'go',
      default_domain: 'saas',
      default_agent_name: 'legacy-reviewer',
      default_agent_mode: 'review',
      default_context_mode: 'auto',
      default_context_token_budget: 1200,
      default_context_max_items: 8,
      default_context_dynamic: true,
      default_memory_writeback: true,
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.getProjectAgentBundle).mockResolvedValue({
      project_id: 'project-1',
      project_dir: 'D:/Work/agent-demo/SuperDev-Studio',
      default_agent_name: 'legacy-reviewer',
      default_agent_mode: 'review',
      agents: [{ name: 'delivery-agent', description: 'Delivery agent' }],
      modes: [{ name: 'step_by_step', description: 'Step mode' }, { name: 'full_cycle', description: 'Full cycle mode' }],
    });
    vi.mocked(apiClient.startPipeline).mockResolvedValue({
      id: 'run-fallback',
      project_id: 'project-1',
      prompt: 'fallback-agent-run',
      status: 'queued',
      progress: 0,
      stage: 'queued',
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByTestId('pipeline-agent-default-fallback-alert')).toBeInTheDocument();
      expect(screen.getByText(AGENT_DEFAULT_FALLBACK_ALERT)).toBeInTheDocument();
    });

    await userEvent.clear(screen.getByPlaceholderText(PROMPT_PLACEHOLDER));
    await userEvent.type(screen.getByPlaceholderText(PROMPT_PLACEHOLDER), 'fallback-agent-run');
    await userEvent.click(screen.getByRole('button', { name: START_BUTTON }));

    await waitFor(() => {
      expect(apiClient.startPipeline).toHaveBeenCalled();
      const [payload] = vi.mocked(apiClient.startPipeline).mock.calls.at(-1)!;
      expect(payload).toEqual(
        expect.objectContaining({
          prompt: 'fallback-agent-run',
          agent_name: 'delivery-agent',
          agent_mode: 'step_by_step',
        }),
      );
    });
  }, 12000);


  it('retries a failed run from compact summary', async () => {
    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-failed',
        project_id: 'project-1',
        prompt: 'failed-run',
        status: 'failed',
        progress: 100,
        stage: 'super-dev',
        created_at: '2026-03-05T00:00:00Z',
        updated_at: '2026-03-05T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.getRun).mockResolvedValue({
      id: 'run-failed',
      project_id: 'project-1',
      prompt: 'failed-run',
      status: 'failed',
      progress: 100,
      stage: 'super-dev',
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
      expect(screen.getByTestId('pipeline-run-details-open')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('pipeline-run-details-open'));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(within(dialog).getByRole('button', { name: RETRY_LABEL }));

    await waitFor(() => {
      expect(apiClient.retryPipeline).toHaveBeenCalled();
      expect(vi.mocked(apiClient.retryPipeline).mock.calls[0][0]).toBe('run-failed');
    });
  }, 12000);

  it('approves pending high-risk full-cycle action from summary', async () => {
    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-awaiting',
        project_id: 'project-1',
        prompt: 'full-cycle-awaiting-approval',
        status: 'awaiting_human',
        progress: 88,
        stage: 'lifecycle-release-approval',
        full_cycle: true,
        created_at: '2026-03-05T00:00:00Z',
        updated_at: '2026-03-05T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.getRun).mockResolvedValue({
      id: 'run-awaiting',
      project_id: 'project-1',
      prompt: 'full-cycle-awaiting-approval',
      status: 'awaiting_human',
      progress: 88,
      stage: 'lifecycle-release-approval',
      full_cycle: true,
      created_at: '2026-03-05T00:00:00Z',
      updated_at: '2026-03-05T00:00:00Z',
    });
    vi.mocked(apiClient.getRunAgent).mockResolvedValue({
      run: {
        id: 'agent-run-1',
        pipeline_run_id: 'run-awaiting',
        project_id: 'project-1',
        change_batch_id: '',
        agent_name: 'reviewer',
        mode_name: 'full_cycle',
        status: 'awaiting_human',
        current_node: 'lifecycle-release-approval',
        summary: 'Awaiting high-risk deploy approval',
        created_at: '2026-03-05T00:00:00Z',
        updated_at: '2026-03-05T00:00:00Z',
      },
      step_count: 3,
      tool_call_count: 1,
      evidence_count: 0,
      evaluation_count: 1,
      latest_evaluation: {
        id: 'eval-1',
        agent_step_id: 'step-1',
        evaluation_type: 'tool_governance',
        verdict: 'need_human',
        reason: 'High-risk deploy action requires human confirmation before continuing.',
        next_action: 'Confirm deploy and continue execution.',
        created_at: '2026-03-05T00:00:00Z',
      },
    });
    vi.mocked(apiClient.listRunAgentToolCalls).mockResolvedValue([
      {
        id: 'tool-1',
        agent_step_id: 'step-1',
        tool_name: 'run_superdev_deploy',
        request_json: '{"stage":"lifecycle-release-approval","args":["deploy","--docker","--cicd","all"]}',
        response_json: '{"status":"awaiting_approval","risk_level":"high","requires_confirmation":true,"approved":false,"reason":"High-risk deploy action requires human confirmation before continuing."}',
        success: false,
        latency_ms: 0,
        created_at: '2026-03-05T00:00:00Z',
      },
    ]);

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(apiClient.listRuns).toHaveBeenCalledWith('project-1');
      expect(apiClient.getRun).toHaveBeenCalledWith('run-awaiting');
    }, { timeout: 4000 });

    await waitFor(() => {
      expect(apiClient.listRunAgentToolCalls).toHaveBeenCalledWith('run-awaiting');
    }, { timeout: 4000 });

    const approveButton = await screen.findByRole('button', { name: APPROVE_LABEL }, { timeout: 4000 });
    await userEvent.click(approveButton);

    await waitFor(() => {
      expect(apiClient.approvePipelineTool).toHaveBeenCalled();
      expect(vi.mocked(apiClient.approvePipelineTool).mock.calls[0][0]).toBe('run-awaiting');
      expect(vi.mocked(apiClient.approvePipelineTool).mock.calls[0][1]).toBe('run_superdev_deploy');
    });
  }, 12000);

  it('renders structured template preview inside run details modal tabs', async () => {
    const markdown = [
      '# 构思增强稿',
      '',
      '## 文档元数据',
      '| 字段 | 值 |',
      '| --- | --- |',
      '| run_id | run-completed |',
      '| stage | llm-idea |',
      '| template_kind | concept |',
      '| change_id | change-001 |',
      '| generated_at | 2026-03-05T00:00:00Z |',
      '| multimodal_assets | 1 |',
      '',
      '## 输入快照',
      '### 输入需求',
      '为阶段产物提供更严格的模板化预览。',
      '',
      '## 参考素材',
      '- https://example.com/reference.png',
      '',
      '## 执行摘要',
      '构思阶段已形成可继续推进的结构化结论。',
      '',
      '## 用户价值',
      '- 让项目成员快速理解当前阶段结论。',
      '',
      '## 验收检查点',
      '- 页面必须稳定显示固定章节。',
      '',
      '## 下一步动作',
      '- 继续推进设计复核。',
      '',
      '## LLM 原始输出',
      '\`\`\`text',
      '{"summary":"构思阶段已形成可继续推进的结构化结论。"}',
      '\`\`\`',
    ].join(String.fromCharCode(10));

    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-completed',
        project_id: 'project-1',
        prompt: 'completed-run',
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
      prompt: 'completed-run',
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
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        text: async () => markdown,
      } as Response),
    );

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByTestId('pipeline-run-details-open')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('pipeline-run-details-open'));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(within(dialog).getByRole('tab', { name: TAB_PREVIEW }));

    await waitFor(() => {
      expect(within(dialog).getByText('模板化预览')).toBeInTheDocument();
      expect(within(dialog).getAllByText('运行 ID').length).toBeGreaterThan(0);
      expect(within(dialog).getAllByText('输入快照').length).toBeGreaterThan(0);
      expect(within(dialog).getAllByText('验收检查点').length).toBeGreaterThan(0);
      expect(within(dialog).getByText('构思阶段已形成可继续推进的结构化结论。')).toBeInTheDocument();
    });
  });

  it('paginates long timeline events and supports back to top', async () => {
    const scrollToSpy = vi.fn();
    vi.stubGlobal('scrollTo', scrollToSpy);

    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-completed',
        project_id: 'project-1',
        prompt: 'timeline-run',
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
      prompt: 'timeline-run',
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
        message: `event-${index + 1}`,
        created_at: `2026-03-05T00:${String(index).padStart(2, '0')}:00Z`,
      })),
    );
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-completed',
      status: 'completed',
      output_dir: 'D:/Work/output',
      checklist: [{ key: 'run-status', title: '流水线状态', status: 'completed', note: 'completed' }],
      artifacts: [],
      stages: [],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByTestId('pipeline-run-details-open')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('pipeline-run-details-open'));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(within(dialog).getByRole('tab', { name: TAB_EXECUTION }));

    await waitFor(() => {
      expect(within(dialog).getByTestId('pipeline-timeline-summary')).toHaveTextContent('1-8 / 12');
      expect(within(dialog).getByText('event-8')).toBeInTheDocument();
    });

    expect(within(dialog).queryByText('event-9')).not.toBeInTheDocument();

    await userEvent.click(within(dialog).getByTestId('pipeline-timeline-next'));

    await waitFor(() => {
      expect(within(dialog).getByTestId('pipeline-timeline-summary')).toHaveTextContent('9-12 / 12');
      expect(within(dialog).getByText('event-9')).toBeInTheDocument();
    });

    await userEvent.click(within(dialog).getByTestId('pipeline-timeline-back-top'));
    expect(scrollToSpy).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' });
  });

  it('shows completion checklist and preview button inside run details modal', async () => {
    vi.mocked(apiClient.listRuns).mockResolvedValue([
      {
        id: 'run-completed',
        project_id: 'project-1',
        prompt: 'finished-run',
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
      prompt: 'finished-run',
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
      artifacts: [{ name: 'PRD 文档', path: 'output/demo-prd.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-prd.md', preview_type: 'markdown', stage: 'design' }],
      stages: [{ key: 'design', title: '设计', status: 'completed', artifacts: [{ name: 'PRD 文档', path: 'output/demo-prd.md', kind: 'markdown', size_bytes: 128, updated_at: '2026-03-05T00:00:00Z', preview_url: '/api/pipeline/runs/run-completed/preview/demo-prd.md', preview_type: 'markdown', stage: 'design' }] }],
      preview_url: '/api/pipeline/runs/run-completed/preview/index.html',
    });

    renderWithProviders(<PipelinePage />);

    await waitFor(() => {
      expect(screen.getByTestId('pipeline-run-details-open')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId('pipeline-run-details-open'));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(within(dialog).getByRole('tab', { name: TAB_STAGES }));

    await waitFor(() => {
      expect(within(dialog).getAllByText('PRD 文档').length).toBeGreaterThan(0);
      expect(within(dialog).getByRole('button', { name: PREVIEW_PAGE_LABEL })).toBeInTheDocument();
    });
  }, 12000);
});
