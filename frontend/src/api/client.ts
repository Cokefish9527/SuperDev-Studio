import axios from 'axios';
import type {
  ContextPack,
  DashboardResponse,
  KnowledgeChunk,
  KnowledgeDocument,
  Memory,
  PipelineCompletion,
  ProjectAdvanceResponse,
  PipelineRun,
  Project,
  RunEvent,
  Task,
} from '../types';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 15000,
});

const unwrapItems = async <T>(promise: Promise<{ data: { items: T[] } }>) => {
  const { data } = await promise;
  return data.items;
};

export const apiClient = {
  health: async () => (await api.get<{ status: string }>('/api/health')).data,

  listProjects: async () => unwrapItems<Project>(api.get('/api/projects')),
  createProject: async (payload: Partial<Project>) => (await api.post<Project>('/api/projects', payload)).data,
  updateProject: async (id: string, payload: Partial<Project>) => (await api.put<Project>(`/api/projects/${id}`, payload)).data,
  deleteProject: async (id: string) => (await api.delete<{ status: string }>(`/api/projects/${id}`)).data,

  listTasks: async (projectId: string) => unwrapItems<Task>(api.get(`/api/projects/${projectId}/tasks`)),
  createTask: async (projectId: string, payload: Partial<Task>) =>
    (await api.post<Task>(`/api/projects/${projectId}/tasks`, payload)).data,
  updateTask: async (taskId: string, payload: Partial<Task>) =>
    (await api.patch<Task>(`/api/tasks/${taskId}`, payload)).data,
  autoScheduleTasks: async (projectId: string, payload?: { start_date?: string }) =>
    (
      await api.post<{ items: Task[]; scheduled_count: number; start_date: string }>(
        `/api/projects/${projectId}/tasks/auto-schedule`,
        payload ?? {},
      )
    ).data,
  advanceProject: async (
    projectId: string,
    payload?: {
      goal?: string;
      mode?: 'step_by_step' | 'full_cycle';
      iteration_limit?: number;
      platform?: string;
      frontend?: string;
      backend?: string;
      domain?: string;
    },
  ) =>
    (
      await api.post<ProjectAdvanceResponse>(`/api/projects/${projectId}/advance`, payload ?? {})
    ).data,

  getDashboard: async (projectId?: string) =>
    (
      await api.get<DashboardResponse>('/api/dashboard', {
        params: projectId ? { project_id: projectId } : undefined,
      })
    ).data,

  listMemories: async (projectId: string, limit = 50) =>
    unwrapItems<Memory>(api.get(`/api/projects/${projectId}/memories`, { params: { limit } })),
  createMemory: async (projectId: string, payload: Partial<Memory>) =>
    (await api.post<Memory>(`/api/projects/${projectId}/memories`, payload)).data,

  listKnowledgeDocuments: async (projectId: string) =>
    unwrapItems<KnowledgeDocument>(api.get(`/api/projects/${projectId}/knowledge/documents`)),
  createKnowledgeDocument: async (
    projectId: string,
    payload: {
      title: string;
      source: string;
      content: string;
      chunk_size?: number;
    },
  ) =>
    (
      await api.post<{ document: KnowledgeDocument; chunks: KnowledgeChunk[] }>(
        `/api/projects/${projectId}/knowledge/documents`,
        payload,
      )
    ).data,
  searchKnowledge: async (projectId: string, query: string, limit = 8) =>
    unwrapItems<KnowledgeChunk>(
      api.get(`/api/projects/${projectId}/knowledge/search`, {
        params: { q: query, limit },
      }),
    ),

  buildContextPack: async (
    projectId: string,
    payload: { query: string; token_budget?: number; max_items?: number },
  ) => (await api.post<ContextPack>(`/api/projects/${projectId}/context-pack`, payload)).data,

  startPipeline: async (payload: {
    project_id: string;
    prompt: string;
    simulate: boolean;
    full_cycle?: boolean;
    step_by_step?: boolean;
    iteration_limit?: number;
    project_dir?: string;
    platform?: string;
    frontend?: string;
    backend?: string;
    domain?: string;
    context_mode?: 'off' | 'auto' | 'manual';
    context_query?: string;
    context_token_budget?: number;
    context_max_items?: number;
    context_dynamic?: boolean;
    memory_writeback?: boolean;
  }) => (await api.post<PipelineRun>('/api/pipeline/runs', payload)).data,
  retryPipeline: async (runId: string) =>
    (await api.post<PipelineRun>(`/api/pipeline/runs/${runId}/retry`)).data,
  getRunCompletion: async (runId: string) =>
    (await api.get<PipelineCompletion>(`/api/pipeline/runs/${runId}/completion`)).data,
  getRun: async (runId: string) => (await api.get<PipelineRun>(`/api/pipeline/runs/${runId}`)).data,
  listRunEvents: async (runId: string) => unwrapItems<RunEvent>(api.get(`/api/pipeline/runs/${runId}/events`)),
  listRuns: async (projectId: string, limit = 20) =>
    unwrapItems<PipelineRun>(api.get(`/api/projects/${projectId}/pipeline-runs`, { params: { limit } })),
};

export default api;
