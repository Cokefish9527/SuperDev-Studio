package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/pipeline"
	"superdevstudio/internal/store"
)

type fakeRunner struct{}

func (f *fakeRunner) RunPipeline(_ context.Context, _ pipeline.RunRequest) ([]string, error) {
	return []string{"ok"}, nil
}

func (f *fakeRunner) RunCommand(_ context.Context, _ pipeline.RunRequest, commandArgs []string) ([]string, error) {
	if len(commandArgs) > 0 && commandArgs[0] == "create" {
		return []string{"✓ 变更 ID: api-step-change"}, nil
	}
	return []string{"ok"}, nil
}

type apiTestEnv struct {
	handler http.Handler
	store   *store.Store
}

func newAPITestEnv(t *testing.T) apiTestEnv {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	co := contextopt.NewService(s)
	pm := pipeline.NewManager(s, &fakeRunner{}, co)
	pm.SetPhaseDelay(5 * time.Millisecond)
	return apiTestEnv{
		handler: NewServer(s, pm, co).Router(),
		store:   s,
	}
}

func createProjectViaAPI(t *testing.T, handler http.Handler, name string) store.Project {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{
		"name":        name,
		"description": "test project",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(payload))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	var project store.Project
	if err := json.Unmarshal(createRes.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	return project
}

func waitForRunCompletion(t *testing.T, s *store.Store, runID string) store.PipelineRun {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := s.GetPipelineRun(ctx, runID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if run.Status == "completed" || run.Status == "failed" {
			return run
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("run %s did not finish before timeout", runID)
	return store.PipelineRun{}
}

func TestCreateAndListProjects(t *testing.T) {
	env := newAPITestEnv(t)
	handler := env.handler

	payload, _ := json.Marshal(map[string]string{
		"name":        "Demo",
		"description": "test project",
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(payload))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRes.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	listRes := httptest.NewRecorder()
	handler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRes.Code)
	}

	var response struct {
		Items []store.Project `json:"items"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 project, got %d", len(response.Items))
	}
}

func TestAutoScheduleTasksEndpoint(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "ScheduleProject")
	ctx := context.Background()

	seedTasks := []store.Task{
		{ProjectID: project.ID, Title: "进行中任务", Status: "in_progress", Priority: "medium"},
		{ProjectID: project.ID, Title: "高优先级任务", Status: "todo", Priority: "high"},
		{ProjectID: project.ID, Title: "完成任务", Status: "done", Priority: "high"},
	}
	for _, task := range seedTasks {
		if _, err := env.store.CreateTask(ctx, task); err != nil {
			t.Fatalf("seed task %q: %v", task.Title, err)
		}
	}

	payload, _ := json.Marshal(map[string]string{
		"start_date": "2026-03-10",
	})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/tasks/auto-schedule",
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var response struct {
		Items          []store.Task `json:"items"`
		ScheduledCount int          `json:"scheduled_count"`
		StartDate      string       `json:"start_date"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ScheduledCount != 2 {
		t.Fatalf("expected scheduled_count=2, got %d", response.ScheduledCount)
	}
	if response.StartDate != "2026-03-10" {
		t.Fatalf("expected response start_date=2026-03-10, got %s", response.StartDate)
	}

	taskByTitle := map[string]store.Task{}
	for _, task := range response.Items {
		taskByTitle[task.Title] = task
	}

	inProgress := taskByTitle["进行中任务"]
	if inProgress.StartDate == nil || inProgress.DueDate == nil {
		t.Fatalf("expected in-progress task to have schedule")
	}
	if inProgress.StartDate.Format("2006-01-02") != "2026-03-10" {
		t.Fatalf("expected in-progress start date 2026-03-10, got %s", inProgress.StartDate.Format("2006-01-02"))
	}

	high := taskByTitle["高优先级任务"]
	if high.StartDate == nil || high.DueDate == nil {
		t.Fatalf("expected high priority task to have schedule")
	}
	if high.StartDate.Format("2006-01-02") != "2026-03-13" {
		t.Fatalf("expected high priority task start date 2026-03-13, got %s", high.StartDate.Format("2006-01-02"))
	}

	done := taskByTitle["完成任务"]
	if done.StartDate != nil || done.DueDate != nil {
		t.Fatalf("expected done task to keep empty schedule")
	}
}

func TestMemoryKnowledgeAndContextPackEndpoints(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "ContextModules")

	memoryPayload, _ := json.Marshal(map[string]any{
		"role":       "note",
		"content":    "发布前必须准备回滚预案",
		"tags":       []string{"release", "risk"},
		"importance": 0.9,
	})
	memoryReq := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/memories",
		bytes.NewReader(memoryPayload),
	)
	memoryReq.Header.Set("Content-Type", "application/json")
	memoryRes := httptest.NewRecorder()
	env.handler.ServeHTTP(memoryRes, memoryReq)
	if memoryRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 for memory create, got %d", memoryRes.Code)
	}

	knowledgePayload, _ := json.Marshal(map[string]any{
		"title":      "部署规范",
		"source":     "runbook",
		"content":    "上线流程必须包含灰度验证和回滚演练",
		"chunk_size": 80,
	})
	knowledgeReq := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/knowledge/documents",
		bytes.NewReader(knowledgePayload),
	)
	knowledgeReq.Header.Set("Content-Type", "application/json")
	knowledgeRes := httptest.NewRecorder()
	env.handler.ServeHTTP(knowledgeRes, knowledgeReq)
	if knowledgeRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 for knowledge create, got %d", knowledgeRes.Code)
	}

	searchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/projects/"+project.ID+"/knowledge/search?q=回滚&limit=5",
		nil,
	)
	searchRes := httptest.NewRecorder()
	env.handler.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for knowledge search, got %d", searchRes.Code)
	}

	var searchResponse struct {
		Items []store.KnowledgeChunk `json:"items"`
	}
	if err := json.Unmarshal(searchRes.Body.Bytes(), &searchResponse); err != nil {
		t.Fatalf("decode knowledge search response: %v", err)
	}
	if len(searchResponse.Items) == 0 {
		t.Fatalf("expected at least one knowledge search result")
	}

	contextPayload, _ := json.Marshal(map[string]any{
		"query":        "上线回滚策略",
		"token_budget": 1200,
		"max_items":    8,
	})
	contextReq := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/context-pack",
		bytes.NewReader(contextPayload),
	)
	contextReq.Header.Set("Content-Type", "application/json")
	contextRes := httptest.NewRecorder()
	env.handler.ServeHTTP(contextRes, contextReq)
	if contextRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for context pack, got %d", contextRes.Code)
	}

	var pack store.ContextPack
	if err := json.Unmarshal(contextRes.Body.Bytes(), &pack); err != nil {
		t.Fatalf("decode context pack: %v", err)
	}
	if len(pack.Memories) == 0 {
		t.Fatalf("expected memories in context pack")
	}
	if strings.TrimSpace(pack.Summary) == "" {
		t.Fatalf("expected non-empty context summary")
	}
}

func TestAdvanceProjectStartsIterativeRunAndSeedsMemory(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "AdvanceProject")
	ctx := context.Background()

	_, err := env.store.CreateTask(ctx, store.Task{
		ProjectID:   project.ID,
		Title:       "完善任务看板推进链路",
		Description: "增加一键推进入口并联动 super-dev",
		Status:      "todo",
		Priority:    "high",
	})
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"goal":            "基于任务看板持续推进项目迭代并优化质量",
		"mode":            "step_by_step",
		"iteration_limit": 3,
	})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/advance",
		bytes.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var advanceResp advanceProjectResponse
	if err := json.Unmarshal(res.Body.Bytes(), &advanceResp); err != nil {
		t.Fatalf("decode advance response: %v", err)
	}
	if advanceResp.Mode != "step_by_step" {
		t.Fatalf("expected mode=step_by_step, got %s", advanceResp.Mode)
	}
	if !advanceResp.MemoryWritten {
		t.Fatalf("expected first advance to write super-dev usage memory")
	}
	if strings.TrimSpace(advanceResp.MemoryID) == "" {
		t.Fatalf("expected memory_id in advance response")
	}

	finished := waitForRunCompletion(t, env.store, advanceResp.Run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed run, got %s", finished.Status)
	}
	if !finished.StepByStep {
		t.Fatalf("expected step_by_step=true")
	}
	if finished.Simulate {
		t.Fatalf("expected simulate=false")
	}

	memories, err := env.store.ListMemories(ctx, project.ID, 50)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	foundUsageMemory := false
	for _, mem := range memories {
		for _, tag := range mem.Tags {
			if tag == superDevUsageMemoryTag {
				foundUsageMemory = true
				break
			}
		}
		if foundUsageMemory {
			break
		}
	}
	if !foundUsageMemory {
		t.Fatalf("expected super-dev usage memory to be created")
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/advance",
		bytes.NewReader([]byte(`{}`)),
	)
	secondReq.Header.Set("Content-Type", "application/json")
	secondRes := httptest.NewRecorder()
	env.handler.ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for second advance, got %d", secondRes.Code)
	}
	var secondResp advanceProjectResponse
	if err := json.Unmarshal(secondRes.Body.Bytes(), &secondResp); err != nil {
		t.Fatalf("decode second advance response: %v", err)
	}
	if secondResp.MemoryWritten {
		t.Fatalf("expected second advance to reuse existing usage memory")
	}
}

func TestAdvanceProjectRejectsInvalidMode(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "AdvanceInvalidMode")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/advance",
		bytes.NewReader([]byte(`{"mode":"invalid-mode"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAdvanceProjectFallsBackWhenRepoPathMissing(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "AdvanceFallbackDir")
	ctx := context.Background()

	if _, err := env.store.UpdateProject(
		ctx,
		project.ID,
		project.Name,
		project.Description,
		"D:/path/does-not-exist-12345",
		project.Status,
	); err != nil {
		t.Fatalf("update project repo_path: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+project.ID+"/advance",
		bytes.NewReader([]byte(`{"mode":"step_by_step"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var response advanceProjectResponse
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(response.Run.ProjectDir) != "" {
		t.Fatalf("expected project_dir fallback to empty, got %q", response.Run.ProjectDir)
	}
	finished := waitForRunCompletion(t, env.store, response.Run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed run, got %s", finished.Status)
	}
}

func TestStartPipelineManualModeRequiresQuery(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "ManualMode")

	payload, _ := json.Marshal(map[string]any{
		"project_id":   project.ID,
		"prompt":       "实现用户登录",
		"simulate":     true,
		"context_mode": "manual",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}

	var response errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if !strings.Contains(response.Error, "context_query is required") {
		t.Fatalf("expected context_query validation error, got %q", response.Error)
	}
}

func TestStartPipelineDynamicContextAndMemoryWriteback(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "DynamicContext")
	ctx := context.Background()

	_, err := env.store.CreateMemory(ctx, store.Memory{
		ProjectID:  project.ID,
		Role:       "note",
		Content:    "订单接口改造需要兼容旧版本客户端",
		Importance: 0.9,
	})
	if err != nil {
		t.Fatalf("seed memory: %v", err)
	}
	_, _, err = env.store.AddKnowledgeDocument(
		ctx,
		project.ID,
		"API Guide",
		"internal",
		"接口发布必须支持灰度和回滚策略",
		160,
	)
	if err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"project_id":           project.ID,
		"prompt":               "实现订单接口改造并补充回滚方案",
		"simulate":             true,
		"context_mode":         "auto",
		"context_dynamic":      true,
		"memory_writeback":     true,
		"context_token_budget": 1200,
		"context_max_items":    8,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var run store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode pipeline run: %v", err)
	}

	finished := waitForRunCompletion(t, env.store, run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed, got %s", finished.Status)
	}

	events, err := env.store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	foundPhaseContextEvent := false
	for _, event := range events {
		if event.Stage == "context-optimizer-phase" && event.Status == "completed" {
			foundPhaseContextEvent = true
			break
		}
	}
	if !foundPhaseContextEvent {
		t.Fatalf("expected context-optimizer-phase completed event")
	}

	memories, err := env.store.ListMemories(ctx, project.ID, 50)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	foundRunSummary := false
	for _, memory := range memories {
		if memory.Role == "run-summary" && strings.Contains(memory.Content, "run_id: "+run.ID) {
			foundRunSummary = true
			break
		}
	}
	if !foundRunSummary {
		t.Fatalf("expected run-summary memory writeback for run %s", run.ID)
	}
}

func TestStartPipelineCanDisableMemoryWriteback(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "NoWriteback")
	ctx := context.Background()

	payload, _ := json.Marshal(map[string]any{
		"project_id":       project.ID,
		"prompt":           "只执行模拟运行不回写记忆",
		"simulate":         true,
		"context_mode":     "off",
		"memory_writeback": false,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var run store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode pipeline run: %v", err)
	}

	waitForRunCompletion(t, env.store, run.ID)

	memories, err := env.store.ListMemories(ctx, project.ID, 20)
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	for _, memory := range memories {
		if memory.Role == "run-summary" {
			t.Fatalf("unexpected run-summary memory when writeback disabled")
		}
	}
}

func TestStartPipelineFullCycleMode(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "FullCycle")

	payload, _ := json.Marshal(map[string]any{
		"project_id":      project.ID,
		"prompt":          "一键完成项目交付",
		"simulate":        true,
		"full_cycle":      true,
		"iteration_limit": 4,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var run store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode pipeline run: %v", err)
	}
	finished := waitForRunCompletion(t, env.store, run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed, got %s", finished.Status)
	}
	if !finished.FullCycle {
		t.Fatalf("expected full_cycle=true")
	}
	if finished.Simulate {
		t.Fatalf("expected simulate=false when full_cycle enabled")
	}
	if finished.IterationLimit != 4 {
		t.Fatalf("expected iteration_limit=4, got %d", finished.IterationLimit)
	}
}

func TestStartPipelineFullCycleDefaultsIterationLimit(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "FullCycleDefaultIter")

	payload, _ := json.Marshal(map[string]any{
		"project_id": project.ID,
		"prompt":     "一键完成项目交付",
		"simulate":   true,
		"full_cycle": true,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var run store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode pipeline run: %v", err)
	}
	finished := waitForRunCompletion(t, env.store, run.ID)
	if !finished.FullCycle {
		t.Fatalf("expected full_cycle=true")
	}
	if finished.Simulate {
		t.Fatalf("expected simulate=false when full_cycle enabled")
	}
	if finished.IterationLimit != 3 {
		t.Fatalf("expected default iteration_limit=3, got %d", finished.IterationLimit)
	}
}

func TestStartPipelineStepByStepForcesRealMode(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "StepByStepMode")

	payload, _ := json.Marshal(map[string]any{
		"project_id":   project.ID,
		"prompt":       "按 super-dev 原生命令逐步开发",
		"simulate":     true,
		"full_cycle":   true,
		"step_by_step": true,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var run store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode pipeline run: %v", err)
	}

	finished := waitForRunCompletion(t, env.store, run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed, got %s", finished.Status)
	}
	if !finished.StepByStep {
		t.Fatalf("expected step_by_step=true")
	}
	if finished.FullCycle {
		t.Fatalf("expected full_cycle=false when step_by_step is enabled")
	}
	if finished.Simulate {
		t.Fatalf("expected simulate=false when step_by_step enabled")
	}
}

func TestRetryFailedPipelineRun(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "Retryable")
	ctx := context.Background()

	failedRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:          project.ID,
		Prompt:             "重试：修复失败流水线",
		Simulate:           false,
		Platform:           "web",
		Frontend:           "react",
		Backend:            "go",
		StepByStep:         true,
		ContextMode:        "auto",
		ContextTokenBudget: 1200,
		ContextMaxItems:    8,
		ContextDynamic:     true,
		MemoryWriteback:    true,
		Status:             "failed",
		Progress:           100,
		Stage:              "super-dev",
	})
	if err != nil {
		t.Fatalf("seed failed run: %v", err)
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/"+failedRun.ID+"/retry", nil)
	retryRes := httptest.NewRecorder()
	env.handler.ServeHTTP(retryRes, retryReq)
	if retryRes.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", retryRes.Code)
	}

	var retriedRun store.PipelineRun
	if err := json.Unmarshal(retryRes.Body.Bytes(), &retriedRun); err != nil {
		t.Fatalf("decode retried run: %v", err)
	}
	if retriedRun.RetryOf != failedRun.ID {
		t.Fatalf("expected retry_of=%s, got %s", failedRun.ID, retriedRun.RetryOf)
	}

	finished := waitForRunCompletion(t, env.store, retriedRun.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected retried run completed, got %s", finished.Status)
	}
	if finished.RetryOf != failedRun.ID {
		t.Fatalf("expected persisted retry_of=%s, got %s", failedRun.ID, finished.RetryOf)
	}
	if !finished.StepByStep {
		t.Fatalf("expected retried run to inherit step_by_step=true")
	}

	events, err := env.store.ListRunEvents(ctx, retriedRun.ID)
	if err != nil {
		t.Fatalf("list retry run events: %v", err)
	}
	foundRetryEvent := false
	for _, event := range events {
		if event.Stage == "retry" && strings.Contains(event.Message, failedRun.ID) {
			foundRetryEvent = true
			break
		}
	}
	if !foundRetryEvent {
		t.Fatalf("expected retry log event referencing previous run")
	}
}

func TestRetryPipelineFallsBackFromMissingProjectDir(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "RetryDirFallback")
	ctx := context.Background()

	if _, err := env.store.UpdateProject(
		ctx,
		project.ID,
		project.Name,
		project.Description,
		"D:/path/does-not-exist-67890",
		project.Status,
	); err != nil {
		t.Fatalf("update project repo_path: %v", err)
	}

	failedRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:          project.ID,
		Prompt:             "重试目录回退验证",
		ProjectDir:         "D:/path/does-not-exist-67890",
		Simulate:           false,
		Platform:           "web",
		Frontend:           "react",
		Backend:            "go",
		StepByStep:         true,
		ContextMode:        "auto",
		ContextTokenBudget: 1200,
		ContextMaxItems:    8,
		ContextDynamic:     true,
		MemoryWriteback:    true,
		Status:             "failed",
		Progress:           100,
		Stage:              "step-create",
	})
	if err != nil {
		t.Fatalf("seed failed run: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/"+failedRun.ID+"/retry", nil)
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", res.Code)
	}

	var retried store.PipelineRun
	if err := json.Unmarshal(res.Body.Bytes(), &retried); err != nil {
		t.Fatalf("decode retried run: %v", err)
	}
	if strings.TrimSpace(retried.ProjectDir) != "" {
		t.Fatalf("expected retried project_dir fallback to empty, got %q", retried.ProjectDir)
	}
}

func TestRetryPipelineRejectsNonFailedRun(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "RetryReject")
	ctx := context.Background()

	completedRun, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:       project.ID,
		Prompt:          "已成功运行，无需重试",
		Simulate:        true,
		MemoryWriteback: true,
		Status:          "completed",
		Progress:        100,
		Stage:           "done",
	})
	if err != nil {
		t.Fatalf("seed completed run: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/runs/"+completedRun.ID+"/retry", nil)
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}

	var response errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if !strings.Contains(response.Error, "only failed runs can be retried") {
		t.Fatalf("expected non-failed retry rejection, got %q", response.Error)
	}
}

func TestPipelineCompletionAndPreviewEndpoints(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "CompletionPreview")
	projectDir := filepath.Join(t.TempDir(), "preview-project")
	frontendDir := filepath.Join(projectDir, "output", "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("create frontend output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "index.html"), []byte("<!doctype html><html><body>preview</body></html>"), 0o644); err != nil {
		t.Fatalf("write preview index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "styles.css"), []byte("body{font-family:sans-serif;}"), 0o644); err != nil {
		t.Fatalf("write preview css: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "app.js"), []byte("console.log('preview');"), 0o644); err != nil {
		t.Fatalf("write preview js: %v", err)
	}

	ctx := context.Background()
	run, err := env.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:       project.ID,
		Prompt:          "研发一款具有时间线的记事本软件，优化移动端的体验",
		ProjectDir:      projectDir,
		Simulate:        false,
		MemoryWriteback: true,
		Status:          "completed",
		Progress:        100,
		Stage:           "done",
	})
	if err != nil {
		t.Fatalf("seed completed run: %v", err)
	}

	completionReq := httptest.NewRequest(http.MethodGet, "/api/pipeline/runs/"+run.ID+"/completion", nil)
	completionRes := httptest.NewRecorder()
	env.handler.ServeHTTP(completionRes, completionReq)
	if completionRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", completionRes.Code)
	}

	var completion pipelineCompletionResponse
	if err := json.Unmarshal(completionRes.Body.Bytes(), &completion); err != nil {
		t.Fatalf("decode completion response: %v", err)
	}
	if completion.RunID != run.ID {
		t.Fatalf("expected run_id=%s, got %s", run.ID, completion.RunID)
	}
	if completion.PreviewURL == "" {
		t.Fatalf("expected preview_url in completion response")
	}
	if len(completion.Checklist) == 0 {
		t.Fatalf("expected completion checklist items")
	}

	previewReq := httptest.NewRequest(http.MethodGet, completion.PreviewURL, nil)
	previewRes := httptest.NewRecorder()
	env.handler.ServeHTTP(previewRes, previewReq)
	if previewRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for preview endpoint, got %d", previewRes.Code)
	}
	if !strings.Contains(previewRes.Body.String(), "preview") {
		t.Fatalf("expected preview content body")
	}
}
