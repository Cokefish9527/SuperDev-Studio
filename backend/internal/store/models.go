package store

import "time"

type Project struct {
	ID                        string    `json:"id"`
	Name                      string    `json:"name"`
	Description               string    `json:"description"`
	RepoPath                  string    `json:"repo_path"`
	Status                    string    `json:"status"`
	DefaultPlatform           string    `json:"default_platform"`
	DefaultFrontend           string    `json:"default_frontend"`
	DefaultBackend            string    `json:"default_backend"`
	DefaultDomain             string    `json:"default_domain"`
	DefaultAgentName          string    `json:"default_agent_name"`
	DefaultAgentMode          string    `json:"default_agent_mode"`
	DefaultContextMode        string    `json:"default_context_mode"`
	DefaultContextTokenBudget int       `json:"default_context_token_budget"`
	DefaultContextMaxItems    int       `json:"default_context_max_items"`
	DefaultContextDynamic     bool      `json:"default_context_dynamic"`
	DefaultMemoryWriteback    bool      `json:"default_memory_writeback"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type ChangeBatch struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	Title            string    `json:"title"`
	Goal             string    `json:"goal"`
	Status           string    `json:"status"`
	Mode             string    `json:"mode"`
	ExternalChangeID string    `json:"external_change_id"`
	LatestRunID      string    `json:"latest_run_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Task struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	Priority      string     `json:"priority"`
	Assignee      string     `json:"assignee"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	EstimatedDays int        `json:"estimated_days"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PipelineRun struct {
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	ChangeBatchID      string     `json:"change_batch_id"`
	ExternalChangeID   string     `json:"external_change_id"`
	Prompt             string     `json:"prompt"`
	LLMEnhancedLoop    bool       `json:"llm_enhanced_loop"`
	MultimodalAssets   []string   `json:"multimodal_assets,omitempty"`
	Simulate           bool       `json:"simulate"`
	ProjectDir         string     `json:"project_dir"`
	Platform           string     `json:"platform"`
	Frontend           string     `json:"frontend"`
	Backend            string     `json:"backend"`
	Domain             string     `json:"domain"`
	ContextMode        string     `json:"context_mode"`
	ContextQuery       string     `json:"context_query"`
	ContextTokenBudget int        `json:"context_token_budget"`
	ContextMaxItems    int        `json:"context_max_items"`
	ContextDynamic     bool       `json:"context_dynamic"`
	MemoryWriteback    bool       `json:"memory_writeback"`
	FullCycle          bool       `json:"full_cycle"`
	StepByStep         bool       `json:"step_by_step"`
	IterationLimit     int        `json:"iteration_limit"`
	RetryOf            string     `json:"retry_of"`
	Status             string     `json:"status"`
	Progress           int        `json:"progress"`
	Stage              string     `json:"stage"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
}

type RunEvent struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentRun struct {
	ID            string     `json:"id"`
	PipelineRunID string     `json:"pipeline_run_id"`
	ProjectID     string     `json:"project_id"`
	ChangeBatchID string     `json:"change_batch_id"`
	AgentName     string     `json:"agent_name"`
	ModeName      string     `json:"mode_name"`
	Status        string     `json:"status"`
	CurrentNode   string     `json:"current_node"`
	Summary       string     `json:"summary"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AgentStep struct {
	ID              string     `json:"id"`
	AgentRunID      string     `json:"agent_run_id"`
	StepIndex       int        `json:"step_index"`
	NodeName        string     `json:"node_name"`
	Title           string     `json:"title"`
	InputJSON       string     `json:"input_json"`
	OutputJSON      string     `json:"output_json"`
	DecisionSummary string     `json:"decision_summary"`
	Status          string     `json:"status"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AgentToolCall struct {
	ID           string    `json:"id"`
	AgentStepID  string    `json:"agent_step_id"`
	ToolName     string    `json:"tool_name"`
	RequestJSON  string    `json:"request_json"`
	ResponseJSON string    `json:"response_json"`
	Success      bool      `json:"success"`
	LatencyMS    int       `json:"latency_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

type AgentEvidence struct {
	ID           string    `json:"id"`
	AgentStepID  string    `json:"agent_step_id"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	Title        string    `json:"title"`
	Snippet      string    `json:"snippet"`
	Score        float64   `json:"score"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type AgentEvaluation struct {
	ID             string    `json:"id"`
	AgentStepID    string    `json:"agent_step_id"`
	EvaluationType string    `json:"evaluation_type"`
	Verdict        string    `json:"verdict"`
	Reason         string    `json:"reason"`
	NextAction     string    `json:"next_action"`
	CreatedAt      time.Time `json:"created_at"`
}

type Memory struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags"`
	Importance float64   `json:"importance"`
	CreatedAt  time.Time `json:"created_at"`
}

type KnowledgeDocument struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type KnowledgeChunk struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"document_id"`
	ProjectID  string    `json:"project_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	Score      float64   `json:"score,omitempty"`
}

type ContextPack struct {
	Query           string           `json:"query"`
	TokenBudget     int              `json:"token_budget"`
	EstimatedTokens int              `json:"estimated_tokens"`
	Summary         string           `json:"summary"`
	Memories        []Memory         `json:"memories"`
	Knowledge       []KnowledgeChunk `json:"knowledge"`
}

// RequirementSession tracks requirement intake to confirmation.
type RequirementSession struct {
	ID                  string     `json:"id"`
	ProjectID           string     `json:"project_id"`
	Title               string     `json:"title"`
	RawInput            string     `json:"raw_input"`
	Status              string     `json:"status"` // draft | awaiting_confirm | confirmed
	LatestSummary       string     `json:"latest_summary"`
	LatestPRD           string     `json:"latest_prd"`
	LatestPlan          string     `json:"latest_plan"`
	LatestRisks         string     `json:"latest_risks"`
	LatestChangeBatchID string     `json:"latest_change_batch_id,omitempty"`
	LatestRunID         string     `json:"latest_run_id,omitempty"`
	ConfirmedAt         *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// RequirementDocVersion stores versioned artifacts produced during intake.
type RequirementDocVersion struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Type      string    `json:"type"` // summary | prd | plan | risks
	Content   string    `json:"content"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// RequirementConfirmation records user confirmation.
type RequirementConfirmation struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}
