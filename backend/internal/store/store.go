package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("record not found")

var sqliteIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var allowedSchemaMutations = map[string]map[string]string{
	"pipeline_runs": {
		"change_batch_id":      "TEXT NOT NULL DEFAULT ''",
		"external_change_id":   "TEXT NOT NULL DEFAULT ''",
		"llm_enhanced_loop":    "INTEGER NOT NULL DEFAULT 0",
		"multimodal_assets":    "TEXT NOT NULL DEFAULT '[]'",
		"simulate":             "INTEGER NOT NULL DEFAULT 1",
		"project_dir":          "TEXT NOT NULL DEFAULT ''",
		"platform":             "TEXT NOT NULL DEFAULT ''",
		"frontend":             "TEXT NOT NULL DEFAULT ''",
		"backend":              "TEXT NOT NULL DEFAULT ''",
		"domain":               "TEXT NOT NULL DEFAULT ''",
		"context_mode":         "TEXT NOT NULL DEFAULT 'off'",
		"context_query":        "TEXT NOT NULL DEFAULT ''",
		"context_token_budget": "INTEGER NOT NULL DEFAULT 0",
		"context_max_items":    "INTEGER NOT NULL DEFAULT 0",
		"context_dynamic":      "INTEGER NOT NULL DEFAULT 0",
		"memory_writeback":     "INTEGER NOT NULL DEFAULT 1",
		"full_cycle":           "INTEGER NOT NULL DEFAULT 0",
		"step_by_step":         "INTEGER NOT NULL DEFAULT 0",
		"iteration_limit":      "INTEGER NOT NULL DEFAULT 3",
		"retry_of":             "TEXT NOT NULL DEFAULT ''",
	},
	"projects": {
		"default_platform":             "TEXT NOT NULL DEFAULT 'web'",
		"default_frontend":             "TEXT NOT NULL DEFAULT 'react'",
		"default_backend":              "TEXT NOT NULL DEFAULT 'go'",
		"default_domain":               "TEXT NOT NULL DEFAULT ''",
		"default_agent_name":           "TEXT NOT NULL DEFAULT 'delivery-agent'",
		"default_agent_mode":           "TEXT NOT NULL DEFAULT 'step_by_step'",
		"default_context_mode":         "TEXT NOT NULL DEFAULT 'auto'",
		"default_context_token_budget": "INTEGER NOT NULL DEFAULT 1200",
		"default_context_max_items":    "INTEGER NOT NULL DEFAULT 8",
		"default_context_dynamic":      "INTEGER NOT NULL DEFAULT 1",
		"default_memory_writeback":     "INTEGER NOT NULL DEFAULT 1",
	},
	"tasks": {
		"start_date":     "TEXT",
		"estimated_days": "INTEGER NOT NULL DEFAULT 0",
	},
	"agent_evaluations": {
		"missing_items_json": "TEXT NOT NULL DEFAULT '[]'",
		"acceptance_delta":   "TEXT NOT NULL DEFAULT ''",
	},
	"requirement_sessions": {
		"latest_summary":         "TEXT NOT NULL DEFAULT ''",
		"latest_prd":             "TEXT NOT NULL DEFAULT ''",
		"latest_plan":            "TEXT NOT NULL DEFAULT ''",
		"latest_risks":           "TEXT NOT NULL DEFAULT ''",
		"latest_change_batch_id": "TEXT NOT NULL DEFAULT ''",
		"latest_run_id":          "TEXT NOT NULL DEFAULT ''",
		"status":                 "TEXT NOT NULL DEFAULT 'draft'",
	},
}

type Store struct {
	db         *sql.DB
	ftsEnabled bool
}

func New(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			repo_path TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			default_platform TEXT NOT NULL DEFAULT 'web',
			default_frontend TEXT NOT NULL DEFAULT 'react',
			default_backend TEXT NOT NULL DEFAULT 'go',
			default_domain TEXT NOT NULL DEFAULT '',
			default_agent_name TEXT NOT NULL DEFAULT 'delivery-agent',
			default_agent_mode TEXT NOT NULL DEFAULT 'step_by_step',
			default_context_mode TEXT NOT NULL DEFAULT 'auto',
			default_context_token_budget INTEGER NOT NULL DEFAULT 1200,
			default_context_max_items INTEGER NOT NULL DEFAULT 8,
			default_context_dynamic INTEGER NOT NULL DEFAULT 1,
			default_memory_writeback INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS change_batches (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			goal TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			mode TEXT NOT NULL DEFAULT 'step_by_step',
			external_change_id TEXT NOT NULL DEFAULT '',
			latest_run_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'todo',
			priority TEXT NOT NULL DEFAULT 'medium',
			assignee TEXT NOT NULL DEFAULT '',
			start_date TEXT,
			due_date TEXT,
			estimated_days INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS pipeline_runs (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			change_batch_id TEXT NOT NULL DEFAULT '',
			external_change_id TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL,
			llm_enhanced_loop INTEGER NOT NULL DEFAULT 0,
			multimodal_assets TEXT NOT NULL DEFAULT '[]',
			simulate INTEGER NOT NULL DEFAULT 1,
			project_dir TEXT NOT NULL DEFAULT '',
			platform TEXT NOT NULL DEFAULT '',
			frontend TEXT NOT NULL DEFAULT '',
			backend TEXT NOT NULL DEFAULT '',
			domain TEXT NOT NULL DEFAULT '',
			context_mode TEXT NOT NULL DEFAULT 'off',
			context_query TEXT NOT NULL DEFAULT '',
			context_token_budget INTEGER NOT NULL DEFAULT 0,
			context_max_items INTEGER NOT NULL DEFAULT 0,
			context_dynamic INTEGER NOT NULL DEFAULT 0,
			memory_writeback INTEGER NOT NULL DEFAULT 1,
			full_cycle INTEGER NOT NULL DEFAULT 0,
			step_by_step INTEGER NOT NULL DEFAULT 0,
			iteration_limit INTEGER NOT NULL DEFAULT 3,
			retry_of TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			progress INTEGER NOT NULL DEFAULT 0,
			stage TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS run_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			stage TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_runs (
			id TEXT PRIMARY KEY,
			pipeline_run_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			change_batch_id TEXT NOT NULL DEFAULT '',
			agent_name TEXT NOT NULL,
			mode_name TEXT NOT NULL,
			status TEXT NOT NULL,
			current_node TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_steps (
			id TEXT PRIMARY KEY,
			agent_run_id TEXT NOT NULL,
			step_index INTEGER NOT NULL,
			node_name TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			input_json TEXT NOT NULL DEFAULT '{}',
			output_json TEXT NOT NULL DEFAULT '{}',
			decision_summary TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(agent_run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_tool_calls (
			id TEXT PRIMARY KEY,
			agent_step_id TEXT NOT NULL,
			tool_name TEXT NOT NULL,
			request_json TEXT NOT NULL DEFAULT '{}',
			response_json TEXT NOT NULL DEFAULT '{}',
			success INTEGER NOT NULL DEFAULT 0,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY(agent_step_id) REFERENCES agent_steps(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_evidence (
			id TEXT PRIMARY KEY,
			agent_step_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			snippet TEXT NOT NULL DEFAULT '',
			score REAL NOT NULL DEFAULT 0,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			FOREIGN KEY(agent_step_id) REFERENCES agent_steps(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS agent_evaluations (
			id TEXT PRIMARY KEY,
			agent_step_id TEXT NOT NULL,
			evaluation_type TEXT NOT NULL,
			verdict TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			next_action TEXT NOT NULL DEFAULT '',
			missing_items_json TEXT NOT NULL DEFAULT '[]',
			acceptance_delta TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(agent_step_id) REFERENCES agent_steps(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_pipeline_run_id ON agent_runs(pipeline_run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_steps_agent_run_id ON agent_steps(agent_run_id, step_index);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_agent_step_id ON agent_tool_calls(agent_step_id);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_evidence_agent_step_id ON agent_evidence(agent_step_id);`,
		`CREATE INDEX IF NOT EXISTS idx_agent_evaluations_agent_step_id ON agent_evaluations(agent_step_id);`,
		`CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '',
			importance REAL NOT NULL DEFAULT 0.5,
			created_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS knowledge_documents (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS knowledge_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(document_id) REFERENCES knowledge_documents(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS requirement_sessions (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			raw_input TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			latest_summary TEXT NOT NULL DEFAULT '',
			latest_prd TEXT NOT NULL DEFAULT '',
			latest_plan TEXT NOT NULL DEFAULT '',
			latest_risks TEXT NOT NULL DEFAULT '',
			latest_change_batch_id TEXT NOT NULL DEFAULT '',
			latest_run_id TEXT NOT NULL DEFAULT '',
			confirmed_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS requirement_doc_versions (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES requirement_sessions(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS requirement_confirmations (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES requirement_sessions(id) ON DELETE CASCADE,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS residual_items (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			pipeline_run_id TEXT NOT NULL,
			agent_run_id TEXT NOT NULL DEFAULT '',
			stage TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT 'dev',
			severity TEXT NOT NULL DEFAULT 'medium',
			summary TEXT NOT NULL DEFAULT '',
			evidence TEXT NOT NULL DEFAULT '',
			suggested_command TEXT NOT NULL DEFAULT '',
			source_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'open',
			resolution_note TEXT NOT NULL DEFAULT '',
			resolved_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS approval_gates (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			pipeline_run_id TEXT NOT NULL,
			change_batch_id TEXT NOT NULL DEFAULT '',
			gate_type TEXT NOT NULL DEFAULT 'tool_governance',
			title TEXT NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			tool_name TEXT NOT NULL DEFAULT '',
			risk_level TEXT NOT NULL DEFAULT '',
			source_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'open',
			resolved_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_change_batches_project_id ON change_batches(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project_id ON pipeline_runs(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_memories_project_id ON memories(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_documents_project_id ON knowledge_documents(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_project_id ON knowledge_chunks(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_requirement_sessions_project_id ON requirement_sessions(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_requirement_doc_versions_session_id ON requirement_doc_versions(session_id, version);`,
		`CREATE INDEX IF NOT EXISTS idx_requirement_confirmations_session_id ON requirement_confirmations(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_residual_items_project_run ON residual_items(project_id, pipeline_run_id, status, updated_at);`,
		`CREATE INDEX IF NOT EXISTS idx_approval_gates_project_run ON approval_gates(project_id, pipeline_run_id, status, updated_at);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	if err := s.ensurePipelineRunColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureProjectColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureTaskColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureAgentEvaluationColumns(ctx); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_chunks_fts
		USING fts5(content, project_id UNINDEXED, document_id UNINDEXED, tokenize='unicode61 porter');
	`); err == nil {
		s.ftsEnabled = true
	}

	return nil
}

func (s *Store) ensurePipelineRunColumns(ctx context.Context) error {
	columns := []struct {
		name       string
		definition string
	}{
		{name: "change_batch_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "external_change_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "llm_enhanced_loop", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "multimodal_assets", definition: "TEXT NOT NULL DEFAULT '[]'"},
		{name: "simulate", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "project_dir", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "platform", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "frontend", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "backend", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "domain", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "context_mode", definition: "TEXT NOT NULL DEFAULT 'off'"},
		{name: "context_query", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "context_token_budget", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "context_max_items", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "context_dynamic", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "memory_writeback", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "full_cycle", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "step_by_step", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "iteration_limit", definition: "INTEGER NOT NULL DEFAULT 3"},
		{name: "retry_of", definition: "TEXT NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		if err := s.ensureTableColumn(ctx, "pipeline_runs", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureProjectColumns(ctx context.Context) error {
	columns := []struct {
		name       string
		definition string
	}{
		{name: "default_platform", definition: "TEXT NOT NULL DEFAULT 'web'"},
		{name: "default_frontend", definition: "TEXT NOT NULL DEFAULT 'react'"},
		{name: "default_backend", definition: "TEXT NOT NULL DEFAULT 'go'"},
		{name: "default_domain", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "default_agent_name", definition: "TEXT NOT NULL DEFAULT 'delivery-agent'"},
		{name: "default_agent_mode", definition: "TEXT NOT NULL DEFAULT 'step_by_step'"},
		{name: "default_context_mode", definition: "TEXT NOT NULL DEFAULT 'auto'"},
		{name: "default_context_token_budget", definition: "INTEGER NOT NULL DEFAULT 1200"},
		{name: "default_context_max_items", definition: "INTEGER NOT NULL DEFAULT 8"},
		{name: "default_context_dynamic", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "default_memory_writeback", definition: "INTEGER NOT NULL DEFAULT 1"},
	}
	for _, column := range columns {
		if err := s.ensureTableColumn(ctx, "projects", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureTaskColumns(ctx context.Context) error {
	columns := []struct {
		name       string
		definition string
	}{
		{name: "start_date", definition: "TEXT"},
		{name: "estimated_days", definition: "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, column := range columns {
		if err := s.ensureTableColumn(ctx, "tasks", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureAgentEvaluationColumns(ctx context.Context) error {
	columns := []struct {
		name       string
		definition string
	}{
		{name: "missing_items_json", definition: "TEXT NOT NULL DEFAULT '[]'"},
		{name: "acceptance_delta", definition: "TEXT NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		if err := s.ensureTableColumn(ctx, "agent_evaluations", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureTableColumn(ctx context.Context, tableName, columnName, definition string) error {
	if err := validateSchemaMutation(tableName, columnName, definition); err != nil {
		return err
	}

	quotedTableName := quoteSQLiteIdentifier(tableName)
	quotedColumnName := quoteSQLiteIdentifier(columnName)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quotedTableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	exists := false
	for rows.Next() {
		var cid int
		var name string
		var ctype sql.NullString
		var notnull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == columnName {
			exists = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = s.db.ExecContext(
		ctx,
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quotedTableName, quotedColumnName, definition),
	)
	return err
}

func validateSchemaMutation(tableName, columnName, definition string) error {
	if !sqliteIdentifierPattern.MatchString(tableName) {
		return fmt.Errorf("invalid schema table identifier: %s", tableName)
	}
	if !sqliteIdentifierPattern.MatchString(columnName) {
		return fmt.Errorf("invalid schema column identifier: %s", columnName)
	}
	allowedColumns, ok := allowedSchemaMutations[tableName]
	if !ok {
		return fmt.Errorf("schema mutation not allowed for table %s", tableName)
	}
	expectedDefinition, ok := allowedColumns[columnName]
	if !ok {
		return fmt.Errorf("schema mutation not allowed for %s.%s", tableName, columnName)
	}
	if definition != expectedDefinition {
		return fmt.Errorf("schema definition mismatch for %s.%s", tableName, columnName)
	}
	return nil
}

func quoteSQLiteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, raw)
}

func normalizeDayUTC(t time.Time) time.Time {
	year, month, day := t.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value != 0
}

func splitTags(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(trimmed, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			tags = append(tags, v)
		}
	}
	return tags
}

func joinTags(tags []string) string {
	cleaned := make([]string, 0, len(tags))
	for _, t := range tags {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, ",")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func applyProjectDefaults(project *Project) {
	if strings.TrimSpace(project.Status) == "" {
		project.Status = "active"
	}
	if strings.TrimSpace(project.DefaultPlatform) == "" {
		project.DefaultPlatform = "web"
	}
	if strings.TrimSpace(project.DefaultFrontend) == "" {
		project.DefaultFrontend = "react"
	}
	if strings.TrimSpace(project.DefaultBackend) == "" {
		project.DefaultBackend = "go"
	}
	if strings.TrimSpace(project.DefaultAgentName) == "" {
		project.DefaultAgentName = "delivery-agent"
	}
	if strings.TrimSpace(project.DefaultAgentMode) == "" {
		project.DefaultAgentMode = "step_by_step"
	}
	if strings.TrimSpace(project.DefaultContextMode) == "" {
		project.DefaultContextMode = "auto"
	}
	if project.DefaultContextTokenBudget <= 0 {
		project.DefaultContextTokenBudget = 1200
	}
	if project.DefaultContextMaxItems <= 0 {
		project.DefaultContextMaxItems = 8
	}
}

func scanProject(scanner rowScanner, project *Project) error {
	var createdRaw, updatedRaw string
	var contextDynamicRaw, memoryWritebackRaw int
	if err := scanner.Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.RepoPath,
		&project.Status,
		&project.DefaultPlatform,
		&project.DefaultFrontend,
		&project.DefaultBackend,
		&project.DefaultDomain,
		&project.DefaultAgentName,
		&project.DefaultAgentMode,
		&project.DefaultContextMode,
		&project.DefaultContextTokenBudget,
		&project.DefaultContextMaxItems,
		&contextDynamicRaw,
		&memoryWritebackRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return err
	}
	project.DefaultContextDynamic = intToBool(contextDynamicRaw)
	project.DefaultMemoryWriteback = intToBool(memoryWritebackRaw)
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return err
	}
	project.CreatedAt = createdAt
	project.UpdatedAt = updatedAt
	applyProjectDefaults(project)
	return nil
}

func scanPipelineRun(scanner rowScanner, run *PipelineRun) error {
	var createdRaw, updatedRaw string
	var startedRaw, finishedRaw sql.NullString
	var multimodalAssetsRaw string
	var simulateRaw, llmEnhancedLoopRaw, contextDynamicRaw, memoryWritebackRaw, fullCycleRaw, stepByStepRaw int
	if err := scanner.Scan(
		&run.ID,
		&run.ProjectID,
		&run.ChangeBatchID,
		&run.ExternalChangeID,
		&run.Prompt,
		&llmEnhancedLoopRaw,
		&multimodalAssetsRaw,
		&simulateRaw,
		&run.ProjectDir,
		&run.Platform,
		&run.Frontend,
		&run.Backend,
		&run.Domain,
		&run.ContextMode,
		&run.ContextQuery,
		&run.ContextTokenBudget,
		&run.ContextMaxItems,
		&contextDynamicRaw,
		&memoryWritebackRaw,
		&fullCycleRaw,
		&stepByStepRaw,
		&run.IterationLimit,
		&run.RetryOf,
		&run.Status,
		&run.Progress,
		&run.Stage,
		&createdRaw,
		&updatedRaw,
		&startedRaw,
		&finishedRaw,
	); err != nil {
		return err
	}
	run.LLMEnhancedLoop = intToBool(llmEnhancedLoopRaw)
	run.MultimodalAssets = decodeStringSlice(multimodalAssetsRaw)
	run.Simulate = intToBool(simulateRaw)
	run.ContextDynamic = intToBool(contextDynamicRaw)
	run.MemoryWriteback = intToBool(memoryWritebackRaw)
	run.FullCycle = intToBool(fullCycleRaw)
	run.StepByStep = intToBool(stepByStepRaw)
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return err
	}
	run.CreatedAt = createdAt
	run.UpdatedAt = updatedAt
	if startedRaw.Valid {
		startedAt, parseErr := parseTime(startedRaw.String)
		if parseErr == nil {
			run.StartedAt = &startedAt
		}
	}
	if finishedRaw.Valid {
		finishedAt, parseErr := parseTime(finishedRaw.String)
		if parseErr == nil {
			run.FinishedAt = &finishedAt
		}
	}
	return nil
}

func scanChangeBatch(scanner rowScanner, batch *ChangeBatch) error {
	var createdRaw, updatedRaw string
	if err := scanner.Scan(
		&batch.ID,
		&batch.ProjectID,
		&batch.Title,
		&batch.Goal,
		&batch.Status,
		&batch.Mode,
		&batch.ExternalChangeID,
		&batch.LatestRunID,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return err
	}
	createdAt, err := parseTime(createdRaw)
	if err != nil {
		return err
	}
	updatedAt, err := parseTime(updatedRaw)
	if err != nil {
		return err
	}
	batch.CreatedAt = createdAt
	batch.UpdatedAt = updatedAt
	return nil
}

func (s *Store) CreateProject(ctx context.Context, p Project) (Project, error) {
	now := nowUTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	applyProjectDefaults(&p)
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO projects(
			id, name, description, repo_path, status,
			default_platform, default_frontend, default_backend, default_domain,
			default_agent_name, default_agent_mode,
			default_context_mode, default_context_token_budget, default_context_max_items,
			default_context_dynamic, default_memory_writeback,
			created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID,
		p.Name,
		p.Description,
		p.RepoPath,
		p.Status,
		p.DefaultPlatform,
		p.DefaultFrontend,
		p.DefaultBackend,
		p.DefaultDomain,
		p.DefaultAgentName,
		p.DefaultAgentMode,
		p.DefaultContextMode,
		p.DefaultContextTokenBudget,
		p.DefaultContextMaxItems,
		boolToInt(p.DefaultContextDynamic),
		boolToInt(p.DefaultMemoryWriteback),
		formatTime(p.CreatedAt),
		formatTime(p.UpdatedAt),
	)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id, name, description, repo_path, status,
			default_platform, default_frontend, default_backend, default_domain,
			default_agent_name, default_agent_mode,
			default_context_mode, default_context_token_budget, default_context_max_items,
			default_context_dynamic, default_memory_writeback,
			created_at, updated_at
		 FROM projects ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		if err := scanProject(rows, &p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, projectID string) (Project, error) {
	var p Project
	if err := scanProject(
		s.db.QueryRowContext(
			ctx,
			`SELECT
				id, name, description, repo_path, status,
				default_platform, default_frontend, default_backend, default_domain,
				default_agent_name, default_agent_mode,
				default_context_mode, default_context_token_budget, default_context_max_items,
				default_context_dynamic, default_memory_writeback,
				created_at, updated_at
			 FROM projects WHERE id = ?`,
			projectID,
		),
		&p,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return p, nil
}

func (s *Store) UpdateProject(ctx context.Context, projectID, name, description, repoPath, status string) (Project, error) {
	return s.UpdateProjectWithDefaults(ctx, projectID, Project{
		Name:        name,
		Description: description,
		RepoPath:    repoPath,
		Status:      status,
	})
}

func (s *Store) UpdateProjectWithDefaults(ctx context.Context, projectID string, patch Project) (Project, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return Project{}, err
	}
	if strings.TrimSpace(patch.Name) != "" {
		p.Name = patch.Name
	}
	p.Description = patch.Description
	p.RepoPath = patch.RepoPath
	if strings.TrimSpace(patch.Status) != "" {
		p.Status = patch.Status
	}
	if strings.TrimSpace(patch.DefaultPlatform) != "" {
		p.DefaultPlatform = patch.DefaultPlatform
	}
	if strings.TrimSpace(patch.DefaultFrontend) != "" {
		p.DefaultFrontend = patch.DefaultFrontend
	}
	if strings.TrimSpace(patch.DefaultBackend) != "" {
		p.DefaultBackend = patch.DefaultBackend
	}
	p.DefaultDomain = patch.DefaultDomain
	if strings.TrimSpace(patch.DefaultAgentName) != "" {
		p.DefaultAgentName = patch.DefaultAgentName
	}
	if strings.TrimSpace(patch.DefaultAgentMode) != "" {
		p.DefaultAgentMode = patch.DefaultAgentMode
	}
	if strings.TrimSpace(patch.DefaultContextMode) != "" {
		p.DefaultContextMode = patch.DefaultContextMode
	}
	if patch.DefaultContextTokenBudget > 0 {
		p.DefaultContextTokenBudget = patch.DefaultContextTokenBudget
	}
	if patch.DefaultContextMaxItems > 0 {
		p.DefaultContextMaxItems = patch.DefaultContextMaxItems
	}
	p.DefaultContextDynamic = patch.DefaultContextDynamic
	p.DefaultMemoryWriteback = patch.DefaultMemoryWriteback
	applyProjectDefaults(&p)
	p.UpdatedAt = nowUTC()

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE projects SET
			name=?, description=?, repo_path=?, status=?,
			default_platform=?, default_frontend=?, default_backend=?, default_domain=?,
			default_agent_name=?, default_agent_mode=?,
			default_context_mode=?, default_context_token_budget=?, default_context_max_items=?,
			default_context_dynamic=?, default_memory_writeback=?,
			updated_at=?
		 WHERE id=?`,
		p.Name,
		p.Description,
		p.RepoPath,
		p.Status,
		p.DefaultPlatform,
		p.DefaultFrontend,
		p.DefaultBackend,
		p.DefaultDomain,
		p.DefaultAgentName,
		p.DefaultAgentMode,
		p.DefaultContextMode,
		p.DefaultContextTokenBudget,
		p.DefaultContextMaxItems,
		boolToInt(p.DefaultContextDynamic),
		boolToInt(p.DefaultMemoryWriteback),
		formatTime(p.UpdatedAt),
		p.ID,
	)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) CreateChangeBatch(ctx context.Context, batch ChangeBatch) (ChangeBatch, error) {
	now := nowUTC()
	if batch.ID == "" {
		batch.ID = uuid.NewString()
	}
	if strings.TrimSpace(batch.Status) == "" {
		batch.Status = "draft"
	}
	if strings.TrimSpace(batch.Mode) == "" {
		batch.Mode = "step_by_step"
	}
	batch.CreatedAt = now
	batch.UpdatedAt = now
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO change_batches(
			id, project_id, title, goal, status, mode, external_change_id, latest_run_id, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		batch.ID,
		batch.ProjectID,
		batch.Title,
		batch.Goal,
		batch.Status,
		batch.Mode,
		batch.ExternalChangeID,
		batch.LatestRunID,
		formatTime(batch.CreatedAt),
		formatTime(batch.UpdatedAt),
	)
	if err != nil {
		return ChangeBatch{}, err
	}
	return batch, nil
}

func (s *Store) ListChangeBatches(ctx context.Context, projectID string) ([]ChangeBatch, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, project_id, title, goal, status, mode, external_change_id, latest_run_id, created_at, updated_at
		 FROM change_batches WHERE project_id=? ORDER BY updated_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ChangeBatch{}
	for rows.Next() {
		var batch ChangeBatch
		if err := scanChangeBatch(rows, &batch); err != nil {
			return nil, err
		}
		items = append(items, batch)
	}
	return items, rows.Err()
}

func (s *Store) GetChangeBatch(ctx context.Context, batchID string) (ChangeBatch, error) {
	var batch ChangeBatch
	if err := scanChangeBatch(
		s.db.QueryRowContext(
			ctx,
			`SELECT id, project_id, title, goal, status, mode, external_change_id, latest_run_id, created_at, updated_at
			 FROM change_batches WHERE id = ?`,
			batchID,
		),
		&batch,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ChangeBatch{}, ErrNotFound
		}
		return ChangeBatch{}, err
	}
	return batch, nil
}

func (s *Store) UpdateChangeBatch(ctx context.Context, batchID, status, latestRunID, externalChangeID string) (ChangeBatch, error) {
	batch, err := s.GetChangeBatch(ctx, batchID)
	if err != nil {
		return ChangeBatch{}, err
	}
	if strings.TrimSpace(status) != "" {
		batch.Status = status
	}
	if strings.TrimSpace(latestRunID) != "" {
		batch.LatestRunID = latestRunID
	}
	if strings.TrimSpace(externalChangeID) != "" {
		batch.ExternalChangeID = externalChangeID
	}
	batch.UpdatedAt = nowUTC()
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE change_batches SET status=?, latest_run_id=?, external_change_id=?, updated_at=? WHERE id=?`,
		batch.Status,
		batch.LatestRunID,
		batch.ExternalChangeID,
		formatTime(batch.UpdatedAt),
		batch.ID,
	)
	if err != nil {
		return ChangeBatch{}, err
	}
	return batch, nil
}

func (s *Store) DeleteProject(ctx context.Context, projectID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, projectID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateTask(ctx context.Context, t Task) (Task, error) {
	now := nowUTC()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Status == "" {
		t.Status = "todo"
	}
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.StartDate != nil {
		start := normalizeDayUTC(*t.StartDate)
		t.StartDate = &start
	}
	if t.DueDate != nil {
		due := normalizeDayUTC(*t.DueDate)
		t.DueDate = &due
	}
	if t.EstimatedDays < 0 {
		t.EstimatedDays = 0
	}
	if t.EstimatedDays == 0 && t.StartDate != nil && t.DueDate != nil && !t.DueDate.Before(*t.StartDate) {
		t.EstimatedDays = int(t.DueDate.Sub(*t.StartDate).Hours()/24) + 1
	}
	t.CreatedAt = now
	t.UpdatedAt = now

	var startRaw any
	var dueRaw any
	if t.StartDate != nil {
		startRaw = formatTime(*t.StartDate)
	}
	if t.DueDate != nil {
		dueRaw = formatTime(*t.DueDate)
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tasks(id, project_id, title, description, status, priority, assignee, start_date, due_date, estimated_days, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID,
		t.ProjectID,
		t.Title,
		t.Description,
		t.Status,
		t.Priority,
		t.Assignee,
		startRaw,
		dueRaw,
		t.EstimatedDays,
		formatTime(t.CreatedAt),
		formatTime(t.UpdatedAt),
	)
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) ListTasks(ctx context.Context, projectID string) ([]Task, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, project_id, title, description, status, priority, assignee, start_date, due_date, estimated_days, created_at, updated_at
		 FROM tasks WHERE project_id = ? ORDER BY updated_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Task{}
	for rows.Next() {
		var t Task
		var startRaw sql.NullString
		var dueRaw sql.NullString
		var createdRaw, updatedRaw string
		if err := rows.Scan(
			&t.ID,
			&t.ProjectID,
			&t.Title,
			&t.Description,
			&t.Status,
			&t.Priority,
			&t.Assignee,
			&startRaw,
			&dueRaw,
			&t.EstimatedDays,
			&createdRaw,
			&updatedRaw,
		); err != nil {
			return nil, err
		}
		t.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		t.UpdatedAt, err = parseTime(updatedRaw)
		if err != nil {
			return nil, err
		}
		if startRaw.Valid {
			startDate, parseErr := parseTime(startRaw.String)
			if parseErr == nil {
				t.StartDate = &startDate
			}
		}
		if dueRaw.Valid {
			dueDate, parseErr := parseTime(dueRaw.String)
			if parseErr == nil {
				t.DueDate = &dueDate
			}
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (s *Store) UpdateTask(ctx context.Context, taskID, status, priority, assignee string) (Task, error) {
	t, err := s.getTaskByID(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if status != "" {
		t.Status = status
	}
	if priority != "" {
		t.Priority = priority
	}
	if assignee != "" {
		t.Assignee = assignee
	}
	t.UpdatedAt = nowUTC()

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE tasks SET status=?, priority=?, assignee=?, updated_at=? WHERE id=?`,
		t.Status,
		t.Priority,
		t.Assignee,
		formatTime(t.UpdatedAt),
		t.ID,
	)
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) AutoScheduleTasks(ctx context.Context, projectID string, startDate time.Time) ([]Task, int, error) {
	tasks, err := s.ListTasks(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}

	openTasks := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if strings.EqualFold(strings.TrimSpace(task.Status), "done") {
			continue
		}
		openTasks = append(openTasks, task)
	}
	if len(openTasks) == 0 {
		return tasks, 0, nil
	}

	sort.SliceStable(openTasks, func(i, j int) bool {
		left := openTasks[i]
		right := openTasks[j]
		leftStatusRank := taskStatusRank(left.Status)
		rightStatusRank := taskStatusRank(right.Status)
		if leftStatusRank != rightStatusRank {
			return leftStatusRank < rightStatusRank
		}

		leftPriorityRank := taskPriorityRank(left.Priority)
		rightPriorityRank := taskPriorityRank(right.Priority)
		if leftPriorityRank != rightPriorityRank {
			return leftPriorityRank < rightPriorityRank
		}

		if left.StartDate != nil && right.StartDate != nil && !left.StartDate.Equal(*right.StartDate) {
			return left.StartDate.Before(*right.StartDate)
		}
		if left.StartDate != nil && right.StartDate == nil {
			return true
		}
		if left.StartDate == nil && right.StartDate != nil {
			return false
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	current := normalizeDayUTC(startDate)
	updatedAt := formatTime(nowUTC())
	for _, task := range openTasks {
		days := task.EstimatedDays
		if days <= 0 {
			days = defaultDurationByPriority(task.Priority)
		}
		if strings.EqualFold(strings.TrimSpace(task.Status), "in_progress") && days < 2 {
			days = 2
		}

		start := current
		due := current.AddDate(0, 0, days-1)
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE tasks SET start_date=?, due_date=?, estimated_days=?, updated_at=? WHERE id=?`,
			formatTime(start),
			formatTime(due),
			days,
			updatedAt,
			task.ID,
		); err != nil {
			return nil, 0, err
		}
		current = due.AddDate(0, 0, 1)
	}

	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}

	items, err := s.ListTasks(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	return items, len(openTasks), nil
}

func (s *Store) getTaskByID(ctx context.Context, taskID string) (Task, error) {
	var t Task
	var startRaw sql.NullString
	var dueRaw sql.NullString
	var createdRaw, updatedRaw string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, project_id, title, description, status, priority, assignee, start_date, due_date, estimated_days, created_at, updated_at FROM tasks WHERE id = ?`,
		taskID,
	).Scan(
		&t.ID,
		&t.ProjectID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.Priority,
		&t.Assignee,
		&startRaw,
		&dueRaw,
		&t.EstimatedDays,
		&createdRaw,
		&updatedRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	t.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return Task{}, err
	}
	t.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return Task{}, err
	}
	if startRaw.Valid {
		startDate, parseErr := parseTime(startRaw.String)
		if parseErr == nil {
			t.StartDate = &startDate
		}
	}
	if dueRaw.Valid {
		dueDate, parseErr := parseTime(dueRaw.String)
		if parseErr == nil {
			t.DueDate = &dueDate
		}
	}
	return t, nil
}

func taskStatusRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "in_progress":
		return 0
	case "todo":
		return 1
	default:
		return 2
	}
}

func taskPriorityRank(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}

func defaultDurationByPriority(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return 4
	case "low":
		return 2
	default:
		return 3
	}
}

func (s *Store) CreatePipelineRun(ctx context.Context, r PipelineRun) (PipelineRun, error) {
	now := nowUTC()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Status == "" {
		r.Status = "queued"
	}
	if r.IterationLimit <= 0 {
		r.IterationLimit = 3
	}
	r.CreatedAt = now
	r.UpdatedAt = now

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO pipeline_runs(
			id, project_id, change_batch_id, external_change_id, prompt,
			llm_enhanced_loop, multimodal_assets,
			simulate, project_dir, platform, frontend, backend, domain,
			context_mode, context_query, context_token_budget, context_max_items, context_dynamic, memory_writeback, full_cycle, step_by_step, iteration_limit, retry_of,
			status, progress, stage, created_at, updated_at, started_at, finished_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID,
		r.ProjectID,
		r.ChangeBatchID,
		r.ExternalChangeID,
		r.Prompt,
		boolToInt(r.LLMEnhancedLoop),
		encodeStringSlice(r.MultimodalAssets),
		boolToInt(r.Simulate),
		r.ProjectDir,
		r.Platform,
		r.Frontend,
		r.Backend,
		r.Domain,
		r.ContextMode,
		r.ContextQuery,
		r.ContextTokenBudget,
		r.ContextMaxItems,
		boolToInt(r.ContextDynamic),
		boolToInt(r.MemoryWriteback),
		boolToInt(r.FullCycle),
		boolToInt(r.StepByStep),
		r.IterationLimit,
		r.RetryOf,
		r.Status,
		r.Progress,
		r.Stage,
		formatTime(r.CreatedAt),
		formatTime(r.UpdatedAt),
		nullableTime(r.StartedAt),
		nullableTime(r.FinishedAt),
	)
	if err != nil {
		return PipelineRun{}, err
	}
	return r, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func (s *Store) UpdatePipelineRun(ctx context.Context, runID, status, stage string, progress int, startedAt, finishedAt *time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE pipeline_runs SET status=?, stage=?, progress=?, started_at=COALESCE(?, started_at), finished_at=?, updated_at=? WHERE id=?`,
		status,
		stage,
		progress,
		nullableTime(startedAt),
		nullableTime(finishedAt),
		formatTime(nowUTC()),
		runID,
	)
	return err
}

func (s *Store) SetPipelineRunExternalChangeID(ctx context.Context, runID, externalChangeID string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE pipeline_runs SET external_change_id=?, updated_at=? WHERE id=?`,
		strings.TrimSpace(externalChangeID),
		formatTime(nowUTC()),
		runID,
	)
	return err
}

func (s *Store) GetPipelineRun(ctx context.Context, runID string) (PipelineRun, error) {
	var r PipelineRun
	if err := scanPipelineRun(s.db.QueryRowContext(
		ctx,
		`SELECT
			id, project_id, change_batch_id, external_change_id, prompt,
			llm_enhanced_loop, multimodal_assets,
			simulate, project_dir, platform, frontend, backend, domain,
			context_mode, context_query, context_token_budget, context_max_items, context_dynamic, memory_writeback, full_cycle, step_by_step, iteration_limit, retry_of,
			status, progress, stage, created_at, updated_at, started_at, finished_at
		FROM pipeline_runs WHERE id = ?`,
		runID,
	), &r); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineRun{}, ErrNotFound
		}
		return PipelineRun{}, err
	}
	return r, nil
}

func (s *Store) ListPipelineRuns(ctx context.Context, projectID string, limit int) ([]PipelineRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			id, project_id, change_batch_id, external_change_id, prompt,
			llm_enhanced_loop, multimodal_assets,
			simulate, project_dir, platform, frontend, backend, domain,
			context_mode, context_query, context_token_budget, context_max_items, context_dynamic, memory_writeback, full_cycle, step_by_step, iteration_limit, retry_of,
			status, progress, stage, created_at, updated_at, started_at, finished_at
		 FROM pipeline_runs WHERE project_id=? ORDER BY created_at DESC LIMIT ?`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PipelineRun{}
	for rows.Next() {
		var r PipelineRun
		if err := scanPipelineRun(rows, &r); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) AppendRunEvent(ctx context.Context, event RunEvent) (RunEvent, error) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = nowUTC()
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO run_events(run_id, stage, status, message, created_at) VALUES(?,?,?,?,?)`,
		event.RunID,
		event.Stage,
		event.Status,
		event.Message,
		formatTime(event.CreatedAt),
	)
	if err != nil {
		return RunEvent{}, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		event.ID = id
	}
	return event, nil
}

func encodeStringSlice(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeStringSlice(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil
	}
	return items
}

func (s *Store) ListRunEvents(ctx context.Context, runID string) ([]RunEvent, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, run_id, stage, status, message, created_at FROM run_events WHERE run_id=? ORDER BY id ASC`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RunEvent{}
	for rows.Next() {
		var e RunEvent
		var createdRaw string
		if err := rows.Scan(&e.ID, &e.RunID, &e.Stage, &e.Status, &e.Message, &createdRaw); err != nil {
			return nil, err
		}
		e.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (s *Store) CreateMemory(ctx context.Context, m Memory) (Memory, error) {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Role == "" {
		m.Role = "note"
	}
	if m.Importance < 0 {
		m.Importance = 0
	}
	if m.Importance > 1 {
		m.Importance = 1
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = nowUTC()
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO memories(id, project_id, role, content, tags, importance, created_at) VALUES(?,?,?,?,?,?,?)`,
		m.ID,
		m.ProjectID,
		m.Role,
		m.Content,
		joinTags(m.Tags),
		m.Importance,
		formatTime(m.CreatedAt),
	)
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

func (s *Store) ListMemories(ctx context.Context, projectID string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, project_id, role, content, tags, importance, created_at
		 FROM memories WHERE project_id=? ORDER BY created_at DESC LIMIT ?`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Memory{}
	for rows.Next() {
		var m Memory
		var tagsRaw, createdRaw string
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Role, &m.Content, &tagsRaw, &m.Importance, &createdRaw); err != nil {
			return nil, err
		}
		m.Tags = splitTags(tagsRaw)
		m.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (s *Store) AddKnowledgeDocument(ctx context.Context, projectID, title, source, content string, chunkSize int) (KnowledgeDocument, []KnowledgeChunk, error) {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	doc := KnowledgeDocument{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Title:     title,
		Source:    source,
		Content:   content,
		CreatedAt: nowUTC(),
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return KnowledgeDocument{}, nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO knowledge_documents(id, project_id, title, source, content, created_at) VALUES(?,?,?,?,?,?)`,
		doc.ID,
		doc.ProjectID,
		doc.Title,
		doc.Source,
		doc.Content,
		formatTime(doc.CreatedAt),
	)
	if err != nil {
		return KnowledgeDocument{}, nil, err
	}

	chunksText := splitIntoChunks(content, chunkSize)
	chunks := make([]KnowledgeChunk, 0, len(chunksText))
	for idx, chunkText := range chunksText {
		created := nowUTC()
		res, execErr := tx.ExecContext(
			ctx,
			`INSERT INTO knowledge_chunks(document_id, project_id, chunk_index, content, created_at) VALUES(?,?,?,?,?)`,
			doc.ID,
			projectID,
			idx,
			chunkText,
			formatTime(created),
		)
		if execErr != nil {
			err = execErr
			return KnowledgeDocument{}, nil, err
		}
		id, _ := res.LastInsertId()
		chunk := KnowledgeChunk{
			ID:         id,
			DocumentID: doc.ID,
			ProjectID:  projectID,
			ChunkIndex: idx,
			Content:    chunkText,
			CreatedAt:  created,
		}
		chunks = append(chunks, chunk)
		if s.ftsEnabled {
			_, _ = tx.ExecContext(
				ctx,
				`INSERT INTO knowledge_chunks_fts(rowid, content, project_id, document_id) VALUES(?,?,?,?)`,
				id,
				chunkText,
				projectID,
				doc.ID,
			)
		}
	}

	if err = tx.Commit(); err != nil {
		return KnowledgeDocument{}, nil, err
	}
	return doc, chunks, nil
}

func splitIntoChunks(content string, chunkSize int) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []string{}
	}

	paragraphs := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	if len(paragraphs) == 0 {
		paragraphs = []string{trimmed}
	}

	chunks := []string{}
	var current strings.Builder
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		candidate := para
		if current.Len() > 0 {
			candidate = current.String() + "\n" + para
		}
		if utf8.RuneCountInString(candidate) <= chunkSize {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(para)
			continue
		}

		if current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}

		if utf8.RuneCountInString(para) <= chunkSize {
			current.WriteString(para)
			continue
		}

		for len(para) > 0 {
			runes := []rune(para)
			if len(runes) <= chunkSize {
				chunks = append(chunks, string(runes))
				break
			}
			chunks = append(chunks, string(runes[:chunkSize]))
			para = string(runes[chunkSize:])
		}
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func (s *Store) ListKnowledgeDocuments(ctx context.Context, projectID string) ([]KnowledgeDocument, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, project_id, title, source, content, created_at FROM knowledge_documents WHERE project_id=? ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []KnowledgeDocument{}
	for rows.Next() {
		var d KnowledgeDocument
		var createdRaw string
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Title, &d.Source, &d.Content, &createdRaw); err != nil {
			return nil, err
		}
		d.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

func (s *Store) SearchKnowledge(ctx context.Context, projectID, query string, limit int) ([]KnowledgeChunk, error) {
	if limit <= 0 {
		limit = 5
	}
	if strings.TrimSpace(query) == "" {
		return []KnowledgeChunk{}, nil
	}

	if s.ftsEnabled {
		ftsQuery := strings.Join(strings.Fields(query), " OR ")
		rows, err := s.db.QueryContext(
			ctx,
			`SELECT kc.id, kc.document_id, kc.project_id, kc.chunk_index, kc.content, kc.created_at, bm25(knowledge_chunks_fts) AS score
			 FROM knowledge_chunks_fts
			 JOIN knowledge_chunks kc ON kc.id = knowledge_chunks_fts.rowid
			 WHERE knowledge_chunks_fts MATCH ? AND kc.project_id = ?
			 ORDER BY score LIMIT ?`,
			ftsQuery,
			projectID,
			limit,
		)
		if err == nil {
			defer rows.Close()
			items := []KnowledgeChunk{}
			for rows.Next() {
				var c KnowledgeChunk
				var createdRaw string
				if scanErr := rows.Scan(&c.ID, &c.DocumentID, &c.ProjectID, &c.ChunkIndex, &c.Content, &createdRaw, &c.Score); scanErr != nil {
					return nil, scanErr
				}
				c.CreatedAt, err = parseTime(createdRaw)
				if err != nil {
					return nil, err
				}
				items = append(items, c)
			}
			return items, rows.Err()
		}
	}

	rows, err := s.db.QueryContext(
		ctx,
		func() string {
			terms := strings.Fields(query)
			if len(terms) == 0 {
				terms = []string{query}
			}
			clauses := make([]string, 0, len(terms))
			for range terms {
				clauses = append(clauses, "content LIKE ?")
			}
			return fmt.Sprintf(
				`SELECT id, document_id, project_id, chunk_index, content, created_at
				 FROM knowledge_chunks WHERE project_id = ? AND (%s)
				 ORDER BY created_at DESC LIMIT ?`,
				strings.Join(clauses, " OR "),
			)
		}(),
		func() []any {
			terms := strings.Fields(query)
			if len(terms) == 0 {
				terms = []string{query}
			}
			args := make([]any, 0, len(terms)+2)
			args = append(args, projectID)
			for _, term := range terms {
				args = append(args, "%"+term+"%")
			}
			args = append(args, limit)
			return args
		}()...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []KnowledgeChunk{}
	for rows.Next() {
		var c KnowledgeChunk
		var createdRaw string
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.ProjectID, &c.ChunkIndex, &c.Content, &createdRaw); err != nil {
			return nil, err
		}
		c.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (s *Store) CreateRequirementSession(ctx context.Context, session RequirementSession) (RequirementSession, error) {
	now := nowUTC()
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	if session.Title == "" {
		session.Title = strings.TrimSpace(session.RawInput)
	}
	if session.Status == "" {
		session.Status = "draft"
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO requirement_sessions(
			id, project_id, title, raw_input, status,
			latest_summary, latest_prd, latest_plan, latest_risks,
			latest_change_batch_id, latest_run_id,
			confirmed_at, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		session.ID,
		session.ProjectID,
		session.Title,
		session.RawInput,
		session.Status,
		session.LatestSummary,
		session.LatestPRD,
		session.LatestPlan,
		session.LatestRisks,
		session.LatestChangeBatchID,
		session.LatestRunID,
		nullableTime(session.ConfirmedAt),
		formatTime(session.CreatedAt),
		formatTime(session.UpdatedAt),
	)
	if err != nil {
		return RequirementSession{}, err
	}
	return session, nil
}

func (s *Store) UpdateRequirementSession(ctx context.Context, session RequirementSession) error {
	session.UpdatedAt = nowUTC()
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE requirement_sessions
		 SET title=?, raw_input=?, status=?,
		     latest_summary=?, latest_prd=?, latest_plan=?, latest_risks=?,
		     latest_change_batch_id=?, latest_run_id=?,
		     confirmed_at=?, updated_at=?
		 WHERE id=?`,
		session.Title,
		session.RawInput,
		session.Status,
		session.LatestSummary,
		session.LatestPRD,
		session.LatestPlan,
		session.LatestRisks,
		session.LatestChangeBatchID,
		session.LatestRunID,
		nullableTime(session.ConfirmedAt),
		formatTime(session.UpdatedAt),
		session.ID,
	)
	return err
}

func (s *Store) AddRequirementDocVersion(ctx context.Context, version RequirementDocVersion) (RequirementDocVersion, error) {
	if version.ID == "" {
		version.ID = uuid.NewString()
	}
	if version.Version <= 0 {
		version.Version = 1
	}
	if version.CreatedAt.IsZero() {
		version.CreatedAt = nowUTC()
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO requirement_doc_versions(id, session_id, project_id, type, content, version, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		version.ID,
		version.SessionID,
		version.ProjectID,
		version.Type,
		version.Content,
		version.Version,
		formatTime(version.CreatedAt),
	)
	if err != nil {
		return RequirementDocVersion{}, err
	}
	return version, nil
}

func (s *Store) CreateRequirementConfirmation(ctx context.Context, confirmation RequirementConfirmation) (RequirementConfirmation, error) {
	if confirmation.ID == "" {
		confirmation.ID = uuid.NewString()
	}
	if confirmation.CreatedAt.IsZero() {
		confirmation.CreatedAt = nowUTC()
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO requirement_confirmations(id, session_id, project_id, note, created_at)
		 VALUES(?,?,?,?,?)`,
		confirmation.ID,
		confirmation.SessionID,
		confirmation.ProjectID,
		confirmation.Note,
		formatTime(confirmation.CreatedAt),
	)
	if err != nil {
		return RequirementConfirmation{}, err
	}
	return confirmation, nil
}

func (s *Store) GetRequirementSession(ctx context.Context, id string) (RequirementSession, error) {
	var r RequirementSession
	var confirmedRaw sql.NullString
	var createdRaw, updatedRaw string
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, project_id, title, raw_input, status,
		        latest_summary, latest_prd, latest_plan, latest_risks,
		        latest_change_batch_id, latest_run_id,
		        confirmed_at, created_at, updated_at
		   FROM requirement_sessions WHERE id=?`,
		id,
	).Scan(
		&r.ID, &r.ProjectID, &r.Title, &r.RawInput, &r.Status,
		&r.LatestSummary, &r.LatestPRD, &r.LatestPlan, &r.LatestRisks,
		&r.LatestChangeBatchID, &r.LatestRunID,
		&confirmedRaw, &createdRaw, &updatedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RequirementSession{}, ErrNotFound
		}
		return RequirementSession{}, err
	}
	var parseErr error
	r.CreatedAt, parseErr = parseTime(createdRaw)
	if parseErr != nil {
		return RequirementSession{}, parseErr
	}
	r.UpdatedAt, parseErr = parseTime(updatedRaw)
	if parseErr != nil {
		return RequirementSession{}, parseErr
	}
	if confirmedRaw.Valid && strings.TrimSpace(confirmedRaw.String) != "" {
		if ts, err := parseTime(confirmedRaw.String); err == nil {
			r.ConfirmedAt = &ts
		}
	}
	return r, nil
}

func (s *Store) ListRequirementSessions(ctx context.Context, projectID string, limit int) ([]RequirementSession, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, project_id, title, raw_input, status,
		        latest_summary, latest_prd, latest_plan, latest_risks,
		        latest_change_batch_id, latest_run_id,
		        confirmed_at, created_at, updated_at
		   FROM requirement_sessions
		   WHERE project_id=? ORDER BY created_at DESC LIMIT ?`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RequirementSession{}
	for rows.Next() {
		var r RequirementSession
		var confirmedRaw sql.NullString
		var createdRaw, updatedRaw string
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.Title, &r.RawInput, &r.Status,
			&r.LatestSummary, &r.LatestPRD, &r.LatestPlan, &r.LatestRisks,
			&r.LatestChangeBatchID, &r.LatestRunID,
			&confirmedRaw, &createdRaw, &updatedRaw,
		); err != nil {
			return nil, err
		}
		var parseErr error
		r.CreatedAt, parseErr = parseTime(createdRaw)
		if parseErr != nil {
			return nil, parseErr
		}
		r.UpdatedAt, parseErr = parseTime(updatedRaw)
		if parseErr != nil {
			return nil, parseErr
		}
		if confirmedRaw.Valid && strings.TrimSpace(confirmedRaw.String) != "" {
			if ts, err := parseTime(confirmedRaw.String); err == nil {
				r.ConfirmedAt = &ts
			}
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (s *Store) ListRequirementDocVersions(ctx context.Context, sessionID string) ([]RequirementDocVersion, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, session_id, project_id, type, content, version, created_at
		   FROM requirement_doc_versions
		   WHERE session_id=?
		   ORDER BY version DESC, created_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RequirementDocVersion{}
	for rows.Next() {
		var v RequirementDocVersion
		var createdRaw string
		if err := rows.Scan(&v.ID, &v.SessionID, &v.ProjectID, &v.Type, &v.Content, &v.Version, &createdRaw); err != nil {
			return nil, err
		}
		parsed, parseErr := parseTime(createdRaw)
		if parseErr != nil {
			return nil, parseErr
		}
		v.CreatedAt = parsed
		items = append(items, v)
	}
	return items, rows.Err()
}

func (s *Store) GetLatestRequirementConfirmation(ctx context.Context, sessionID string) (RequirementConfirmation, error) {
	var c RequirementConfirmation
	var createdRaw string
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, session_id, project_id, note, created_at
		   FROM requirement_confirmations
		   WHERE session_id=?
		   ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&c.ID, &c.SessionID, &c.ProjectID, &c.Note, &createdRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RequirementConfirmation{}, ErrNotFound
		}
		return RequirementConfirmation{}, err
	}
	parsed, err := parseTime(createdRaw)
	if err != nil {
		return RequirementConfirmation{}, err
	}
	c.CreatedAt = parsed
	return c, nil
}

func (s *Store) NextRequirementDocVersion(ctx context.Context, sessionID, docType string) (int, error) {
	var maxVersion sql.NullInt64
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT MAX(version) FROM requirement_doc_versions WHERE session_id=? AND type=?`,
		sessionID,
		docType,
	).Scan(&maxVersion); err != nil {
		return 0, err
	}
	if !maxVersion.Valid {
		return 1, nil
	}
	return int(maxVersion.Int64) + 1, nil
}

func (s *Store) DashboardStats(ctx context.Context, projectID string) (map[string]int, error) {
	stats := map[string]int{
		"projects": 0,
		"tasks":    0,
		"runs":     0,
		"memories": 0,
		"docs":     0,
	}

	var projectsCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&projectsCount); err != nil {
		return nil, err
	}
	stats["projects"] = projectsCount
	if projectID == "" {
		return stats, nil
	}

	queries := map[string]string{
		"tasks":    `SELECT COUNT(*) FROM tasks WHERE project_id=?`,
		"runs":     `SELECT COUNT(*) FROM pipeline_runs WHERE project_id=?`,
		"memories": `SELECT COUNT(*) FROM memories WHERE project_id=?`,
		"docs":     `SELECT COUNT(*) FROM knowledge_documents WHERE project_id=?`,
	}
	for key, query := range queries {
		var count int
		if err := s.db.QueryRowContext(ctx, query, projectID).Scan(&count); err != nil {
			return nil, err
		}
		stats[key] = count
	}

	return stats, nil
}
