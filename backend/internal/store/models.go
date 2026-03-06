package store

import "time"

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RepoPath    string    `json:"repo_path"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	Prompt             string     `json:"prompt"`
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
