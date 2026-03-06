package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("record not found")

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
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
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
			prompt TEXT NOT NULL,
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
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project_id ON pipeline_runs(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_memories_project_id ON memories(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_documents_project_id ON knowledge_documents(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_project_id ON knowledge_chunks(project_id);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	if err := s.ensurePipelineRunColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureTaskColumns(ctx); err != nil {
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

func (s *Store) ensureTableColumn(ctx context.Context, tableName, columnName, definition string) error {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
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
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition),
	)
	return err
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

func (s *Store) CreateProject(ctx context.Context, p Project) (Project, error) {
	now := nowUTC()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Status == "" {
		p.Status = "active"
	}
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO projects(id, name, description, repo_path, status, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`,
		p.ID,
		p.Name,
		p.Description,
		p.RepoPath,
		p.Status,
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
		`SELECT id, name, description, repo_path, status, created_at, updated_at FROM projects ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		var createdRaw, updatedRaw string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.RepoPath, &p.Status, &createdRaw, &updatedRaw); err != nil {
			return nil, err
		}
		p.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		p.UpdatedAt, err = parseTime(updatedRaw)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, projectID string) (Project, error) {
	var p Project
	var createdRaw, updatedRaw string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, description, repo_path, status, created_at, updated_at FROM projects WHERE id = ?`,
		projectID,
	).Scan(&p.ID, &p.Name, &p.Description, &p.RepoPath, &p.Status, &createdRaw, &updatedRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	p.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return Project{}, err
	}
	p.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Store) UpdateProject(ctx context.Context, projectID, name, description, repoPath, status string) (Project, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return Project{}, err
	}
	if name != "" {
		p.Name = name
	}
	p.Description = description
	p.RepoPath = repoPath
	if status != "" {
		p.Status = status
	}
	p.UpdatedAt = nowUTC()

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE projects SET name=?, description=?, repo_path=?, status=?, updated_at=? WHERE id=?`,
		p.Name,
		p.Description,
		p.RepoPath,
		p.Status,
		formatTime(p.UpdatedAt),
		p.ID,
	)
	if err != nil {
		return Project{}, err
	}
	return p, nil
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
			id, project_id, prompt,
			simulate, project_dir, platform, frontend, backend, domain,
			context_mode, context_query, context_token_budget, context_max_items, context_dynamic, memory_writeback, full_cycle, step_by_step, iteration_limit, retry_of,
			status, progress, stage, created_at, updated_at, started_at, finished_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID,
		r.ProjectID,
		r.Prompt,
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

func (s *Store) GetPipelineRun(ctx context.Context, runID string) (PipelineRun, error) {
	var r PipelineRun
	var createdRaw, updatedRaw string
	var startedRaw, finishedRaw sql.NullString
	var simulateRaw, contextDynamicRaw, memoryWritebackRaw, fullCycleRaw, stepByStepRaw int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			id, project_id, prompt,
			simulate, project_dir, platform, frontend, backend, domain,
			context_mode, context_query, context_token_budget, context_max_items, context_dynamic, memory_writeback, full_cycle, step_by_step, iteration_limit, retry_of,
			status, progress, stage, created_at, updated_at, started_at, finished_at
		FROM pipeline_runs WHERE id = ?`,
		runID,
	).Scan(
		&r.ID,
		&r.ProjectID,
		&r.Prompt,
		&simulateRaw,
		&r.ProjectDir,
		&r.Platform,
		&r.Frontend,
		&r.Backend,
		&r.Domain,
		&r.ContextMode,
		&r.ContextQuery,
		&r.ContextTokenBudget,
		&r.ContextMaxItems,
		&contextDynamicRaw,
		&memoryWritebackRaw,
		&fullCycleRaw,
		&stepByStepRaw,
		&r.IterationLimit,
		&r.RetryOf,
		&r.Status,
		&r.Progress,
		&r.Stage,
		&createdRaw,
		&updatedRaw,
		&startedRaw,
		&finishedRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineRun{}, ErrNotFound
		}
		return PipelineRun{}, err
	}
	r.Simulate = intToBool(simulateRaw)
	r.ContextDynamic = intToBool(contextDynamicRaw)
	r.MemoryWriteback = intToBool(memoryWritebackRaw)
	r.FullCycle = intToBool(fullCycleRaw)
	r.StepByStep = intToBool(stepByStepRaw)
	r.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return PipelineRun{}, err
	}
	r.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return PipelineRun{}, err
	}
	if startedRaw.Valid {
		started, parseErr := parseTime(startedRaw.String)
		if parseErr == nil {
			r.StartedAt = &started
		}
	}
	if finishedRaw.Valid {
		finished, parseErr := parseTime(finishedRaw.String)
		if parseErr == nil {
			r.FinishedAt = &finished
		}
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
			id, project_id, prompt,
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
		var createdRaw, updatedRaw string
		var startedRaw, finishedRaw sql.NullString
		var simulateRaw, contextDynamicRaw, memoryWritebackRaw, fullCycleRaw, stepByStepRaw int
		if err := rows.Scan(
			&r.ID,
			&r.ProjectID,
			&r.Prompt,
			&simulateRaw,
			&r.ProjectDir,
			&r.Platform,
			&r.Frontend,
			&r.Backend,
			&r.Domain,
			&r.ContextMode,
			&r.ContextQuery,
			&r.ContextTokenBudget,
			&r.ContextMaxItems,
			&contextDynamicRaw,
			&memoryWritebackRaw,
			&fullCycleRaw,
			&stepByStepRaw,
			&r.IterationLimit,
			&r.RetryOf,
			&r.Status,
			&r.Progress,
			&r.Stage,
			&createdRaw,
			&updatedRaw,
			&startedRaw,
			&finishedRaw,
		); err != nil {
			return nil, err
		}
		r.Simulate = intToBool(simulateRaw)
		r.ContextDynamic = intToBool(contextDynamicRaw)
		r.MemoryWriteback = intToBool(memoryWritebackRaw)
		r.FullCycle = intToBool(fullCycleRaw)
		r.StepByStep = intToBool(stepByStepRaw)
		r.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		r.UpdatedAt, err = parseTime(updatedRaw)
		if err != nil {
			return nil, err
		}
		if startedRaw.Valid {
			started, parseErr := parseTime(startedRaw.String)
			if parseErr == nil {
				r.StartedAt = &started
			}
		}
		if finishedRaw.Valid {
			finished, parseErr := parseTime(finishedRaw.String)
			if parseErr == nil {
				r.FinishedAt = &finished
			}
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
