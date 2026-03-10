export type Project = {
  id: string;
  name: string;
  description: string;
  repo_path: string;
  status: string;
  default_platform: string;
  default_frontend: string;
  default_backend: string;
  default_domain: string;
  default_agent_name: string;
  default_agent_mode: string;
  default_context_mode: 'off' | 'auto' | 'manual' | string;
  default_context_token_budget: number;
  default_context_max_items: number;
  default_context_dynamic: boolean;
  default_memory_writeback: boolean;
  created_at: string;
  updated_at: string;
};

export type ChangeBatch = {
  id: string;
  project_id: string;
  title: string;
  goal: string;
  status: string;
  mode: string;
  external_change_id?: string;
  latest_run_id?: string;
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
  change_batch_id?: string;
  external_change_id?: string;
  prompt: string;
  llm_enhanced_loop?: boolean;
  multimodal_assets?: string[];
  simulate?: boolean;
  project_dir?: string;
  platform?: string;
  frontend?: string;
  backend?: string;
  domain?: string;
  context_mode?: 'off' | 'auto' | 'manual' | string;
  context_query?: string;
  context_token_budget?: number;
  context_max_items?: number;
  context_dynamic?: boolean;
  memory_writeback?: boolean;
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

export type AgentRun = {
  id: string;
  pipeline_run_id: string;
  project_id: string;
  change_batch_id?: string;
  agent_name: string;
  mode_name: string;
  status: string;
  current_node: string;
  summary?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
};

export type AgentStep = {
  id: string;
  agent_run_id: string;
  step_index: number;
  node_name: string;
  title: string;
  input_json: string;
  output_json: string;
  decision_summary: string;
  status: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
};

export type AgentToolCall = {
  id: string;
  agent_step_id: string;
  tool_name: string;
  request_json: string;
  response_json: string;
  success: boolean;
  latency_ms: number;
  created_at: string;
};

export type AgentEvidence = {
  id: string;
  agent_step_id: string;
  source_type: string;
  source_id: string;
  title: string;
  snippet: string;
  score: number;
  metadata_json: string;
  created_at: string;
};

export type AgentEvaluation = {
  id: string;
  agent_step_id: string;
  evaluation_type: string;
  verdict: string;
  reason: string;
  next_action: string;
  next_command?: string;
  missing_items: string[];
  acceptance_delta: string;
  created_at: string;
};

export type PipelineAutoAdvanceResult = {
  action: string;
  reason: string;
  executed: boolean;
  blocking?: string;
  next_command?: string;
  run?: PipelineRun;
};

export type ResidualItem = {
  id: string;
  project_id: string;
  pipeline_run_id: string;
  agent_run_id?: string;
  stage: string;
  category: string;
  severity: string;
  summary: string;
  evidence: string;
  suggested_command: string;
  source_key: string;
  status: 'open' | 'resolved' | 'waived' | string;
  resolution_note?: string;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
};

export type PreviewSession = {
  id: string;
  project_id: string;
  pipeline_run_id: string;
  change_batch_id?: string;
  preview_url: string;
  preview_type: string;
  title: string;
  source_key: string;
  status: 'generated' | 'accepted' | 'rejected' | string;
  reviewer_note?: string;
  created_at: string;
  updated_at: string;
  reviewed_at?: string;
};

export type DeliveryAcceptance = {
  id: string;
  project_id: string;
  pipeline_run_id: string;
  change_batch_id?: string;
  status: 'accepted' | 'revoked' | string;
  reviewer_note?: string;
  created_at: string;
  updated_at: string;
  reviewed_at?: string;
};

export type ApprovalGate = {
  id: string;
  project_id: string;
  pipeline_run_id: string;
  change_batch_id?: string;
  gate_type: string;
  title: string;
  detail: string;
  tool_name?: string;
  risk_level?: string;
  source_key: string;
  status: 'open' | 'resolved' | string;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
};

export type PipelineRunAgent = {
  run: AgentRun;
  step_count: number;
  tool_call_count: number;
  evidence_count: number;
  evaluation_count: number;
  latest_evaluation?: AgentEvaluation;
};

export type AgentProfile = {
  name: string;
  description: string;
  default_model?: string;
  allowed_tools?: string[];
  default_skills?: string[];
  max_steps?: number;
};

export type AgentModeProfile = {
  name: string;
  description: string;
  allow_deploy?: boolean;
  max_retries?: number;
  require_approval?: boolean;
};

export type ProjectAgentBundle = {
  project_id: string;
  project_dir: string;
  default_agent_name: string;
  default_agent_mode: string;
  agents: AgentProfile[];
  modes: AgentModeProfile[];
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

export type RequirementSession = {
  id: string;
  project_id: string;
  title: string;
  raw_input: string;
  status: string;
  latest_summary: string;
  latest_prd: string;
  latest_plan: string;
  latest_risks: string;
  latest_change_batch_id?: string;
  latest_run_id?: string;
  confirmed_at?: string;
  created_at: string;
  updated_at: string;
};

export type RequirementDocVersion = {
  id: string;
  session_id: string;
  project_id: string;
  type: string;
  content: string;
  version: number;
  created_at: string;
};

export type RequirementConfirmation = {
  id: string;
  session_id: string;
  project_id: string;
  note: string;
  created_at: string;
};

export type RequirementSessionBundle = {
  session: RequirementSession;
  doc_versions?: RequirementDocVersion[];
  confirmation?: RequirementConfirmation;
  run?: PipelineRun;
  change_batch?: ChangeBatch;
  delivery_error?: string;
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
  preview_url?: string;
  preview_type?: 'markdown' | 'html' | 'text' | 'image' | 'binary' | string;
  stage?: 'idea' | 'design' | 'superdev' | 'output' | 'rethink' | string;
};

export type PipelineStage = {
  key: 'idea' | 'design' | 'superdev' | 'output' | 'rethink' | string;
  title: string;
  status: 'completed' | 'missing' | 'failed' | 'in_progress' | 'pending' | string;
  artifacts: PipelineArtifact[];
};

export type PipelineCompletion = {
  run_id: string;
  status: string;
  output_dir: string;
  checklist: PipelineCompletionItem[];
  artifacts: PipelineArtifact[];
  stages: PipelineStage[];
  preview_url?: string;
};

export type ProjectAdvanceResponse = {
  run: PipelineRun;
  change_batch?: ChangeBatch;
  mode: 'step_by_step' | 'full_cycle' | string;
  memory_written: boolean;
  memory_id?: string;
};
