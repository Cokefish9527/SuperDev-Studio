import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SimpleDeliveryPage from './SimpleDeliveryPage';
import { apiClient } from '../api/client';
import { renderWithProviders } from '../test/render';

const TEXTAREA_PLACEHOLDER = '一句话或一段话描述需求';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    apiClient: {
      ...actual.apiClient,
      getProject: vi.fn(),
      createRequirementSession: vi.fn(),
      getRequirementSession: vi.fn(),
      reviseRequirementSession: vi.fn(),
      confirmRequirementSession: vi.fn(),
      getRun: vi.fn(),
      getRunCompletion: vi.fn(),
      getRunAgent: vi.fn(),
      listRunPreviewSessions: vi.fn(),
      listRunApprovalGates: vi.fn(),
      listRunResidualItems: vi.fn(),
      updatePreviewSession: vi.fn(),
      autoAdvancePipeline: vi.fn(),
    },
  };
});

describe('SimpleDeliveryPage', () => {
  beforeEach(() => {
    localStorage.setItem('superdev-studio-active-project', 'project-1');
    vi.clearAllMocks();
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
      default_agent_mode: 'step_by_step',
      default_context_mode: 'auto',
      default_context_token_budget: 1200,
      default_context_max_items: 8,
      default_context_dynamic: true,
      default_memory_writeback: true,
      created_at: '2026-03-10T00:00:00Z',
      updated_at: '2026-03-10T00:00:00Z',
    });
    vi.mocked(apiClient.getRequirementSession).mockResolvedValue(undefined as never);
    vi.mocked(apiClient.reviseRequirementSession).mockResolvedValue(undefined as never);
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-placeholder',
      status: 'queued',
      output_dir: 'D:/Work/output',
      checklist: [],
      artifacts: [],
      stages: [],
    });
    vi.mocked(apiClient.getRunAgent).mockResolvedValue(undefined as never);
    vi.mocked(apiClient.listRunPreviewSessions).mockResolvedValue([]);
    vi.mocked(apiClient.listRunApprovalGates).mockResolvedValue([]);
    vi.mocked(apiClient.listRunResidualItems).mockResolvedValue([]);
    vi.mocked(apiClient.updatePreviewSession).mockResolvedValue(undefined as never);
    vi.mocked(apiClient.autoAdvancePipeline).mockResolvedValue({
      action: 'complete_delivery',
      reason: 'Delivery is already complete.',
      executed: false,
      next_command: 'complete_delivery',
    });
  });

  it('automatically advances a failed delivery run into the next run', async () => {
    vi.mocked(apiClient.createRequirementSession).mockResolvedValue({
      session: {
        id: 'session-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        raw_input: 'Build a timeline knowledge notebook.',
        status: 'awaiting_confirm',
        latest_summary: 'summary',
        latest_prd: 'prd',
        latest_plan: 'plan',
        latest_risks: 'risks',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      doc_versions: [],
    });
    vi.mocked(apiClient.confirmRequirementSession).mockResolvedValue({
      session: {
        id: 'session-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        raw_input: 'Build a timeline knowledge notebook.',
        status: 'confirmed',
        latest_summary: 'summary',
        latest_prd: 'prd',
        latest_plan: 'plan',
        latest_risks: 'risks',
        latest_change_batch_id: 'change-1',
        latest_run_id: 'run-old',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      run: {
        id: 'run-old',
        project_id: 'project-1',
        prompt: 'Timeline notebook',
        status: 'failed',
        progress: 100,
        stage: 'quality',
        step_by_step: true,
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      change_batch: {
        id: 'change-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        goal: 'Build a timeline knowledge notebook.',
        status: 'running',
        mode: 'step_by_step',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
    });
    vi.mocked(apiClient.getRun).mockImplementation(async (runId: string) => {
      if (runId === 'run-new') {
        return {
          id: 'run-new',
          project_id: 'project-1',
          prompt: 'Timeline notebook',
          status: 'running',
          progress: 10,
          stage: 'queued',
          step_by_step: true,
          created_at: '2026-03-10T00:01:00Z',
          updated_at: '2026-03-10T00:01:00Z',
        };
      }
      return {
        id: 'run-old',
        project_id: 'project-1',
        prompt: 'Timeline notebook',
        status: 'failed',
        progress: 100,
        stage: 'quality',
        step_by_step: true,
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      };
    });
    vi.mocked(apiClient.getRunCompletion).mockImplementation(async (runId: string) => ({
      run_id: runId,
      status: runId === 'run-new' ? 'running' : 'failed',
      output_dir: 'D:/Work/output',
      checklist: [],
      artifacts: [],
      stages: [],
    }));
    vi.mocked(apiClient.getRunAgent).mockImplementation(async (runId: string) => {
      if (runId !== 'run-old') {
        return undefined as never;
      }
      return {
        run: {
          id: 'agent-run-1',
          pipeline_run_id: 'run-old',
          project_id: 'project-1',
          change_batch_id: 'change-1',
          agent_name: 'reviewer',
          mode_name: 'step_by_step',
          status: 'failed',
          current_node: 'quality',
          summary: 'Retry required',
          created_at: '2026-03-10T00:00:00Z',
          updated_at: '2026-03-10T00:00:00Z',
        },
        step_count: 1,
        tool_call_count: 0,
        evidence_count: 0,
        evaluation_count: 1,
        latest_evaluation: {
          id: 'eval-1',
          agent_step_id: 'step-1',
          evaluation_type: 'step-outcome',
          verdict: 'retry',
          reason: 'Run another repair attempt.',
          next_action: 'Run another repair attempt.',
          next_command: 'rerun_delivery',
          missing_items: [],
          acceptance_delta: '',
          created_at: '2026-03-10T00:00:00Z',
        },
      };
    });
    vi.mocked(apiClient.autoAdvancePipeline).mockResolvedValue({
      action: 'rerun_delivery',
      reason: 'A new delivery run has been started automatically.',
      executed: true,
      next_command: 'rerun_delivery',
      run: {
        id: 'run-new',
        project_id: 'project-1',
        prompt: 'Timeline notebook',
        status: 'queued',
        progress: 0,
        stage: 'queued',
        step_by_step: true,
        retry_of: 'run-old',
        created_at: '2026-03-10T00:01:00Z',
        updated_at: '2026-03-10T00:01:00Z',
      },
    });

    renderWithProviders(<SimpleDeliveryPage />);

    await userEvent.type(screen.getByPlaceholderText(TEXTAREA_PLACEHOLDER), 'Build a timeline knowledge notebook.');
    await userEvent.click(screen.getByRole('button', { name: '生成需求草案' }));

    const confirmButton = await screen.findByRole('button', { name: '确认并启动交付' });
    await userEvent.click(confirmButton);

    await waitFor(() => {
      expect(apiClient.autoAdvancePipeline).toHaveBeenCalledWith('run-old');
    });

    await waitFor(() => {
      expect(apiClient.getRun).toHaveBeenCalledWith('run-new');
    });
  }, 20000);

  it('shows preview review actions and continues auto advance after acceptance', async () => {
    vi.mocked(apiClient.createRequirementSession).mockResolvedValue({
      session: {
        id: 'session-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        raw_input: 'Build a timeline knowledge notebook.',
        status: 'awaiting_confirm',
        latest_summary: 'summary',
        latest_prd: 'prd',
        latest_plan: 'plan',
        latest_risks: 'risks',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      doc_versions: [],
    });
    vi.mocked(apiClient.confirmRequirementSession).mockResolvedValue({
      session: {
        id: 'session-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        raw_input: 'Build a timeline knowledge notebook.',
        status: 'confirmed',
        latest_summary: 'summary',
        latest_prd: 'prd',
        latest_plan: 'plan',
        latest_risks: 'risks',
        latest_change_batch_id: 'change-1',
        latest_run_id: 'run-preview',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      run: {
        id: 'run-preview',
        project_id: 'project-1',
        prompt: 'Timeline notebook',
        status: 'completed',
        progress: 100,
        stage: 'done',
        step_by_step: true,
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      change_batch: {
        id: 'change-1',
        project_id: 'project-1',
        title: 'Timeline notebook',
        goal: 'Build a timeline knowledge notebook.',
        status: 'running',
        mode: 'step_by_step',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
    });
    vi.mocked(apiClient.getRun).mockResolvedValue({
      id: 'run-preview',
      project_id: 'project-1',
      prompt: 'Timeline notebook',
      status: 'completed',
      progress: 100,
      stage: 'done',
      step_by_step: true,
      created_at: '2026-03-10T00:00:00Z',
      updated_at: '2026-03-10T00:00:00Z',
    });
    vi.mocked(apiClient.getRunCompletion).mockResolvedValue({
      run_id: 'run-preview',
      status: 'completed',
      output_dir: 'D:/Work/output',
      checklist: [],
      artifacts: [],
      stages: [],
      preview_url: '/api/pipeline/runs/run-preview/preview',
    });
    vi.mocked(apiClient.getRunAgent).mockResolvedValue({
      run: {
        id: 'agent-run-1',
        pipeline_run_id: 'run-preview',
        project_id: 'project-1',
        change_batch_id: 'change-1',
        agent_name: 'reviewer',
        mode_name: 'step_by_step',
        status: 'completed',
        current_node: 'done',
        summary: 'Preview needs review',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
      step_count: 1,
      tool_call_count: 0,
      evidence_count: 0,
      evaluation_count: 1,
      latest_evaluation: {
        id: 'eval-1',
        agent_step_id: 'step-1',
        evaluation_type: 'step-outcome',
        verdict: 'pass',
        reason: 'Preview is ready for review.',
        next_action: 'Review preview before final sign-off.',
        next_command: 'review_preview',
        missing_items: [],
        acceptance_delta: '',
        created_at: '2026-03-10T00:00:00Z',
      },
    });
    vi.mocked(apiClient.listRunPreviewSessions).mockResolvedValue([
      {
        id: 'preview-1',
        project_id: 'project-1',
        pipeline_run_id: 'run-preview',
        preview_url: '/api/pipeline/runs/run-preview/preview',
        preview_type: 'html',
        title: 'Preview build',
        source_key: 'preview:1',
        status: 'generated',
        created_at: '2026-03-10T00:00:00Z',
        updated_at: '2026-03-10T00:00:00Z',
      },
    ]);
    vi.mocked(apiClient.updatePreviewSession).mockResolvedValue({
      id: 'preview-1',
      project_id: 'project-1',
      pipeline_run_id: 'run-preview',
      preview_url: '/api/pipeline/runs/run-preview/preview',
      preview_type: 'html',
      title: 'Preview build',
      source_key: 'preview:1',
      status: 'accepted',
      created_at: '2026-03-10T00:00:00Z',
      updated_at: '2026-03-10T00:01:00Z',
    });
    vi.mocked(apiClient.autoAdvancePipeline)
      .mockResolvedValueOnce({
        action: 'review_preview',
        reason: 'Preview review is still pending; auto advance stops until a reviewer accepts or rejects the preview.',
        executed: false,
        blocking: 'preview_review',
        next_command: 'review_preview',
      })
      .mockResolvedValueOnce({
        action: 'complete_delivery',
        reason: 'Delivery is already complete; no further automatic action is required.',
        executed: false,
        next_command: 'complete_delivery',
      });

    renderWithProviders(<SimpleDeliveryPage />);

    await userEvent.type(screen.getByPlaceholderText(TEXTAREA_PLACEHOLDER), 'Build a timeline knowledge notebook.');
    await userEvent.click(screen.getByRole('button', { name: '生成需求草案' }));
    await userEvent.click(await screen.findByRole('button', { name: '确认并启动交付' }));

    await waitFor(() => {
      expect(apiClient.autoAdvancePipeline).toHaveBeenCalledTimes(1);
    });

    expect(await screen.findByTestId('simple-delivery-status-alert')).toHaveTextContent('请先完成预览验收');

    await userEvent.click(await screen.findByTestId('simple-delivery-preview-accept'));

    await waitFor(() => {
      expect(apiClient.updatePreviewSession).toHaveBeenCalledWith('preview-1', { status: 'accepted' });
      expect(apiClient.autoAdvancePipeline).toHaveBeenCalledTimes(2);
    });
  }, 20000);
});
