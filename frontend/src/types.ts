export type Project = {
  id: string;
  name: string;
  description: string;
  repo_path: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type Task = {
  id: string;
  project_id: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  assignee: string;
  start_date?: string;
  due_date?: string;
  estimated_days: number;
  created_at: string;
  updated_at: string;
};

export type PipelineRun = {
  id: string;
  project_id: string;
  prompt: string;
  full_cycle?: boolean;
  step_by_step?: boolean;
  iteration_limit?: number;
  retry_of?: string;
  status: string;
  progress: number;
  stage: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
};

export type RunEvent = {
  id: number;
  run_id: string;
  stage: string;
  status: string;
  message: string;
  created_at: string;
};

export type Memory = {
  id: string;
  project_id: string;
  role: string;
  content: string;
  tags: string[];
  importance: number;
  created_at: string;
};

export type KnowledgeDocument = {
  id: string;
  project_id: string;
  title: string;
  source: string;
  content: string;
  created_at: string;
};

export type KnowledgeChunk = {
  id: number;
  document_id: string;
  project_id: string;
  chunk_index: number;
  content: string;
  created_at: string;
  score?: number;
};

export type ContextPack = {
  query: string;
  token_budget: number;
  estimated_tokens: number;
  summary: string;
  memories: Memory[];
  knowledge: KnowledgeChunk[];
};

export type DashboardResponse = {
  stats: {
    projects: number;
    tasks: number;
    runs: number;
    memories: number;
    docs: number;
  };
  recent_runs?: PipelineRun[];
};

export type PipelineCompletionItem = {
  key: string;
  title: string;
  status: 'completed' | 'missing' | 'failed' | 'in_progress' | string;
  note?: string;
};

export type PipelineArtifact = {
  name: string;
  path: string;
  kind: string;
  size_bytes: number;
  updated_at: string;
};

export type PipelineCompletion = {
  run_id: string;
  status: string;
  output_dir: string;
  checklist: PipelineCompletionItem[];
  artifacts: PipelineArtifact[];
  preview_url?: string;
};

export type ProjectAdvanceResponse = {
  run: PipelineRun;
  mode: 'step_by_step' | 'full_cycle' | string;
  memory_written: boolean;
  memory_id?: string;
};
