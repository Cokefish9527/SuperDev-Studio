package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/store"
)

type fakeRunner struct {
	mu             sync.Mutex
	lines          []string
	err            error
	capturedPrompt string
	commandCalls   [][]string
	commandFn      func(req RunRequest, commandArgs []string) ([]string, error)
}

func (f *fakeRunner) RunPipeline(_ context.Context, req RunRequest) ([]string, error) {
	f.mu.Lock()
	f.capturedPrompt = req.Prompt
	f.mu.Unlock()
	return f.lines, f.err
}

func (f *fakeRunner) RunCommand(_ context.Context, req RunRequest, commandArgs []string) ([]string, error) {
	f.mu.Lock()
	if len(commandArgs) > 0 {
		copied := append([]string{}, commandArgs...)
		f.commandCalls = append(f.commandCalls, copied)
	}
	commandFn := f.commandFn
	lines := append([]string{}, f.lines...)
	err := f.err
	f.mu.Unlock()

	if commandFn != nil {
		return commandFn(req, commandArgs)
	}
	return lines, err
}

func (f *fakeRunner) Prompt() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.capturedPrompt
}

func (f *fakeRunner) Commands() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := make([][]string, 0, len(f.commandCalls))
	for _, call := range f.commandCalls {
		copied = append(copied, append([]string{}, call...))
	}
	return copied
}

func newPipelineTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "pipeline.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestManagerStartSimulation(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "Pipeline"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	manager := NewManager(s, &fakeRunner{}, contextopt.NewService(s))
	manager.phaseDelay = 5 * time.Millisecond
	manager.phases = []string{"phase-a", "phase-b"}

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "build feature",
		Simulate:  true,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Progress != 100 {
		t.Fatalf("expected 100 progress, got %d", updated.Progress)
	}
}

func TestManagerStartWithRunner(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "Pipeline"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	manager := NewManager(s, &fakeRunner{lines: []string{"line1", "line2"}}, contextopt.NewService(s))
	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "run real",
		Simulate:  false,
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "completed" {
		t.Fatalf("expected completed, got %s", updated.Status)
	}

	events, listErr := s.ListRunEvents(ctx, run.ID)
	if listErr != nil {
		t.Fatalf("list events: %v", listErr)
	}
	if len(events) < 3 {
		t.Fatalf("expected run events, got %d", len(events))
	}
}

func TestManagerInjectsContextIntoRunnerPrompt(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "ContextPipeline"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = s.CreateMemory(ctx, store.Memory{
		ProjectID:  project.ID,
		Role:       "note",
		Content:    "接口字段必须兼容旧版本客户端",
		Importance: 0.9,
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}
	_, _, err = s.AddKnowledgeDocument(
		ctx,
		project.ID,
		"Design Doc",
		"internal",
		"对外 API 变更必须先走灰度发布和回滚预案。",
		120,
	)
	if err != nil {
		t.Fatalf("add knowledge doc: %v", err)
	}

	runner := &fakeRunner{lines: []string{"done"}}
	manager := NewManager(s, runner, contextopt.NewService(s))
	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "实现订单接口改造",
		Simulate:  false,
		Context: ContextOptions{
			Mode:            ContextModeAuto,
			TokenBudget:     1200,
			MaxItems:        6,
			DynamicByPhase:  true,
			MemoryWriteback: true,
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	waitForRunCompletion(t, s, run.ID)

	capturedPrompt := runner.Prompt()
	if !strings.Contains(capturedPrompt, "上下文优化摘要") {
		t.Fatalf("expected injected context summary in prompt, got: %s", capturedPrompt)
	}
	if !strings.Contains(capturedPrompt, "阶段动态上下文") {
		t.Fatalf("expected dynamic phase context in prompt, got: %s", capturedPrompt)
	}

	events, err := s.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	foundContextEvent := false
	for _, event := range events {
		if event.Stage == "context-optimizer" && event.Status == "completed" {
			foundContextEvent = true
			break
		}
	}
	if !foundContextEvent {
		t.Fatalf("expected context optimizer event in run events")
	}
	memories, err := s.ListMemories(ctx, project.ID, 20)
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

func TestManagerWritebackKnowledgeFromPlansAndOutputDocs(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "KnowledgeWriteback"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()
	changeID := "knowledge-writeback-change"
	projectDocName := "knowledge-writeback-prd.md"
	projectDocContent := "# PRD\n\n本次迭代的产品需求文档内容。"

	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "create":
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				docPath := filepath.Join(outputDir, projectDocName)
				if writeErr := os.WriteFile(docPath, []byte(projectDocContent), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{
					"项目创建完成",
					"✓ 变更 ID: " + changeID,
				}, nil
			case "spec":
				return []string{"spec validate passed"}, nil
			case "task":
				return []string{"task command passed"}, nil
			case "quality":
				return []string{"quality passed"}, nil
			case "preview":
				outputFile := filepath.Join(req.ProjectDir, "output", "preview.html")
				if writeErr := os.WriteFile(outputFile, []byte("<html>preview</html>"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"preview ok"}, nil
			case "deploy":
				return []string{"deploy ok"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}

	manager := NewManager(s, runner, contextopt.NewService(s))
	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "将本次迭代方案和输出文档沉淀进知识库",
		Simulate:  false,
		Context: ContextOptions{
			MemoryWriteback: true,
		},
		Lifecycle: LifecycleOptions{
			StepByStep:     true,
			IterationLimit: 2,
		},
		Options: RunRequest{
			Prompt:     "将本次迭代方案和输出文档沉淀进知识库",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	finished := waitForRunCompletion(t, s, run.ID)
	if finished.Status != "completed" {
		t.Fatalf("expected completed, got %s", finished.Status)
	}

	docs, err := s.ListKnowledgeDocuments(ctx, project.ID)
	if err != nil {
		t.Fatalf("list knowledge docs: %v", err)
	}
	foundPlan := false
	foundProjectDoc := false
	for _, doc := range docs {
		if doc.Source == "volcengine-plan:"+run.ID {
			foundPlan = true
		}
		if strings.HasPrefix(doc.Source, "super-dev-output:"+run.ID+":") && strings.Contains(doc.Content, "产品需求文档内容") {
			foundProjectDoc = true
		}
	}
	if !foundPlan {
		t.Fatalf("expected volcengine plan knowledge document for run %s", run.ID)
	}
	if !foundProjectDoc {
		t.Fatalf("expected super-dev output markdown knowledge document for run %s", run.ID)
	}
}

func TestManagerOneClickLifecycleCompletesAfterQualityRetry(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "OneClick"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()

	qualityAttempts := 0
	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "pipeline":
				return []string{"项目: one-click", "pipeline ok"}, nil
			case "quality":
				qualityAttempts++
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				reportPath := filepath.Join(outputDir, "one-click-quality-gate.md")
				if qualityAttempts == 1 {
					if writeErr := os.WriteFile(reportPath, []byte("质量门禁未通过"), 0o644); writeErr != nil {
						return nil, writeErr
					}
					return []string{"quality failed"}, errors.New("quality gate failed")
				}
				if writeErr := os.WriteFile(reportPath, []byte("质量门禁通过"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"quality passed"}, nil
			case "deploy":
				return []string{"deploy ok"}, nil
			case "preview":
				outputFile := filepath.Join(req.ProjectDir, "output", "preview.html")
				if writeErr := os.WriteFile(outputFile, []byte("<html>preview</html>"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"preview ok"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}
	manager := NewManager(s, runner, contextopt.NewService(s))

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "一键交付订单系统",
		Simulate:  false,
		Lifecycle: LifecycleOptions{
			OneClickDelivery: true,
			IterationLimit:   3,
		},
		Options: RunRequest{
			Prompt:     "一键交付订单系统",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "completed" {
		t.Fatalf("expected completed, got %s", updated.Status)
	}
	projectTasks, listErr := s.ListTasks(ctx, project.ID)
	if listErr != nil {
		t.Fatalf("list tasks: %v", listErr)
	}
	if len(projectTasks) == 0 {
		t.Fatalf("expected one-click lifecycle to create project tasks")
	}

	calls := runner.Commands()
	if len(calls) < 6 {
		t.Fatalf("expected at least 6 command calls, got %d", len(calls))
	}

	qualityCount := 0
	deployCount := 0
	configSyncCount := 0
	for _, call := range calls {
		if len(call) == 0 {
			continue
		}
		if call[0] == "quality" {
			qualityCount++
		}
		if call[0] == "deploy" {
			deployCount++
		}
		if len(call) >= 4 && call[0] == "config" && call[1] == "set" && call[2] == "name" {
			configSyncCount++
		}
	}
	if qualityCount < 2 {
		t.Fatalf("expected at least 2 quality checks, got %d", qualityCount)
	}
	if deployCount != 1 {
		t.Fatalf("expected deploy command once, got %d", deployCount)
	}
	if configSyncCount == 0 {
		t.Fatalf("expected lifecycle config sync command")
	}
}

func TestManagerOneClickLifecycleFailsWhenQualityNeverPasses(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "OneClickFail"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()

	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "pipeline":
				return []string{"pipeline ok"}, nil
			case "quality":
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				reportPath := filepath.Join(outputDir, "one-click-quality-gate.md")
				if writeErr := os.WriteFile(reportPath, []byte("质量门禁未通过"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"quality failed"}, errors.New("quality gate failed")
			case "deploy", "preview":
				return []string{"unexpected"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}
	manager := NewManager(s, runner, contextopt.NewService(s))

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "一键交付失败场景",
		Simulate:  false,
		Lifecycle: LifecycleOptions{
			OneClickDelivery: true,
			IterationLimit:   2,
		},
		Options: RunRequest{
			Prompt:     "一键交付失败场景",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "failed" {
		t.Fatalf("expected failed, got %s", updated.Status)
	}
	if updated.Stage != "lifecycle-quality" {
		t.Fatalf("expected lifecycle-quality stage, got %s", updated.Stage)
	}

	calls := runner.Commands()
	for _, call := range calls {
		if len(call) == 0 {
			continue
		}
		if call[0] == "deploy" || call[0] == "preview" {
			t.Fatalf("did not expect %s command after quality failure", call[0])
		}
	}
}

func TestManagerOneClickLifecycleUsesRawRequestPrompt(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "OneClickPrompt"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()

	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "pipeline":
				return []string{
					"项目: one-click-prompt",
					"pipeline ok",
				}, nil
			case "quality":
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				reportPath := filepath.Join(outputDir, "one-click-prompt-quality-gate.md")
				if writeErr := os.WriteFile(reportPath, []byte("质量门禁通过"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"quality passed"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}
	manager := NewManager(s, runner, contextopt.NewService(s))

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "原始需求提示词",
		Simulate:  false,
		Lifecycle: LifecycleOptions{
			OneClickDelivery: true,
			IterationLimit:   1,
		},
		Options: RunRequest{
			Prompt:     "原始需求提示词\n---\n上下文优化摘要（不应直接透传）",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "completed" {
		t.Fatalf("expected completed, got %s", updated.Status)
	}

	calls := runner.Commands()
	foundPipeline := false
	for _, call := range calls {
		if len(call) < 2 || call[0] != "pipeline" {
			continue
		}
		foundPipeline = true
		if !strings.Contains(call[1], "原始需求提示词") {
			t.Fatalf("expected pipeline prompt to contain raw prompt, got %s", call[1])
		}
		if strings.Contains(call[1], "上下文优化摘要") {
			t.Fatalf("expected pipeline prompt not to contain injected context payload, got %s", call[1])
		}
		break
	}
	if !foundPipeline {
		t.Fatalf("expected pipeline command call")
	}
}

func TestManagerStepByStepLifecycleCommandOrder(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "StepByStep"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()

	changeID := "add-order-workflow"
	qualityAttempts := 0
	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "create":
				return []string{
					"项目创建完成",
					"✓ 变更 ID: " + changeID,
				}, nil
			case "spec":
				return []string{"spec validate passed"}, nil
			case "task":
				if len(commandArgs) >= 2 && commandArgs[1] == "status" {
					return []string{"task status: 3/5 completed"}, nil
				}
				return []string{"task run completed"}, nil
			case "quality":
				qualityAttempts++
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				reportPath := filepath.Join(outputDir, "step-by-step-quality-gate.md")
				if qualityAttempts == 1 {
					if writeErr := os.WriteFile(reportPath, []byte("质量门禁未通过"), 0o644); writeErr != nil {
						return nil, writeErr
					}
					return []string{"quality failed"}, errors.New("quality gate failed")
				}
				if writeErr := os.WriteFile(reportPath, []byte("质量门禁通过"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"quality passed"}, nil
			case "preview":
				outputFile := filepath.Join(req.ProjectDir, "output", "preview.html")
				if writeErr := os.WriteFile(outputFile, []byte("<html>preview</html>"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"preview ok"}, nil
			case "deploy":
				return []string{"deploy ok"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}
	manager := NewManager(s, runner, contextopt.NewService(s))

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "按 super-dev 原生步骤开发订单系统",
		Simulate:  false,
		Lifecycle: LifecycleOptions{
			StepByStep: true,
		},
		Options: RunRequest{
			Prompt:     "按 super-dev 原生步骤开发订单系统",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "completed" {
		t.Fatalf("expected completed, got %s", updated.Status)
	}
	if !updated.StepByStep {
		t.Fatalf("expected step_by_step=true")
	}
	projectTasks, listErr := s.ListTasks(ctx, project.ID)
	if listErr != nil {
		t.Fatalf("list project tasks: %v", listErr)
	}
	if len(projectTasks) == 0 {
		t.Fatalf("expected project tasks to be generated from initial docs")
	}
	completedCount := 0
	for _, item := range projectTasks {
		if item.Status == "done" {
			completedCount++
		}
	}
	if completedCount == 0 {
		t.Fatalf("expected at least one project task completed")
	}

	calls := runner.Commands()
	if len(calls) < 9 {
		t.Fatalf("expected at least 9 command calls, got %d", len(calls))
	}
	if calls[0][0] != "create" {
		t.Fatalf("expected first command create, got %s", calls[0][0])
	}
	if len(calls[1]) < 3 || calls[1][0] != "spec" || calls[1][1] != "validate" || calls[1][2] != changeID {
		t.Fatalf("expected second command spec validate %s, got %v", changeID, calls[1])
	}
	if len(calls[2]) < 3 || calls[2][0] != "task" || calls[2][1] != "status" || calls[2][2] != changeID {
		t.Fatalf("expected third command task status %s, got %v", changeID, calls[2])
	}

	taskRunCount := 0
	qualityCount := 0
	previewCount := 0
	deployCount := 0
	for _, call := range calls {
		if len(call) == 0 {
			continue
		}
		if call[0] == "task" && len(call) >= 3 && call[1] == "run" {
			taskRunCount++
			if call[2] != changeID {
				t.Fatalf("expected task run to use change_id=%s, got %v", changeID, call)
			}
		}
		if call[0] == "quality" {
			qualityCount++
		}
		if call[0] == "preview" {
			previewCount++
		}
		if call[0] == "deploy" {
			deployCount++
		}
	}
	if taskRunCount < 2 {
		t.Fatalf("expected at least 2 task run iterations, got %d", taskRunCount)
	}
	if qualityCount < 2 {
		t.Fatalf("expected at least 2 quality checks, got %d", qualityCount)
	}
	if previewCount != 1 {
		t.Fatalf("expected preview command once, got %d", previewCount)
	}
	if deployCount != 1 {
		t.Fatalf("expected deploy command once, got %d", deployCount)
	}
}

func TestManagerStepByStepLifecycleBuildsNextIterationTasks(t *testing.T) {
	s := newPipelineTestStore(t)
	ctx := context.Background()
	project, err := s.CreateProject(ctx, store.Project{Name: "StepByStepNextTasks"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectDir := t.TempDir()

	changeID := "next-iteration-change"
	runner := &fakeRunner{
		commandFn: func(req RunRequest, commandArgs []string) ([]string, error) {
			if len(commandArgs) == 0 {
				return nil, errors.New("empty command args")
			}
			switch commandArgs[0] {
			case "create":
				return []string{
					"项目创建完成",
					"✓ 变更 ID: " + changeID,
				}, nil
			case "spec":
				return []string{"spec validate passed"}, nil
			case "task":
				return []string{"task command passed"}, nil
			case "quality":
				outputDir := filepath.Join(req.ProjectDir, "output")
				if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
					return nil, mkErr
				}
				reportPath := filepath.Join(outputDir, "step-by-step-quality-gate.md")
				if writeErr := os.WriteFile(reportPath, []byte("质量门禁通过"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"quality passed"}, nil
			case "preview":
				outputFile := filepath.Join(req.ProjectDir, "output", "preview.html")
				if writeErr := os.WriteFile(outputFile, []byte("<html>preview</html>"), 0o644); writeErr != nil {
					return nil, writeErr
				}
				return []string{"preview ok"}, nil
			case "deploy":
				return []string{"deploy ok"}, nil
			default:
				return []string{"ok"}, nil
			}
		},
	}
	manager := NewManager(s, runner, contextopt.NewService(s))

	run, err := manager.Start(ctx, StartRequest{
		ProjectID: project.ID,
		Prompt:    "基于任务看板持续推进项目迭代",
		Simulate:  false,
		Lifecycle: LifecycleOptions{
			StepByStep:     true,
			IterationLimit: 2,
		},
		Options: RunRequest{
			Prompt:     "基于任务看板持续推进项目迭代",
			ProjectDir: projectDir,
			Platform:   "web",
			Frontend:   "react",
			Backend:    "go",
		},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}

	updated := waitForRunCompletion(t, s, run.ID)
	if updated.Status != "completed" {
		t.Fatalf("expected completed, got %s", updated.Status)
	}

	projectTasks, listErr := s.ListTasks(ctx, project.ID)
	if listErr != nil {
		t.Fatalf("list project tasks: %v", listErr)
	}
	doneCount := 0
	todoCount := 0
	nextIterationTaskCount := 0
	for _, item := range projectTasks {
		switch item.Status {
		case "done":
			doneCount++
		case "todo":
			todoCount++
		}
		if strings.HasPrefix(item.Title, "下一迭代：") {
			nextIterationTaskCount++
		}
	}
	if doneCount == 0 {
		t.Fatalf("expected completed tasks after step-by-step run")
	}
	if todoCount == 0 {
		t.Fatalf("expected next-iteration todo tasks to be generated")
	}
	if nextIterationTaskCount == 0 {
		t.Fatalf("expected generated tasks with 下一迭代 prefix")
	}
}

func TestParseTaskVerificationAnswer(t *testing.T) {
	passed, reason, ok := parseTaskVerificationAnswer("PASS: 满足需求并通过验收")
	if !ok || !passed {
		t.Fatalf("expected PASS to be parsed")
	}
	if !strings.Contains(reason, "满足需求") {
		t.Fatalf("unexpected reason: %s", reason)
	}

	passed, reason, ok = parseTaskVerificationAnswer("FAIL: 缺少验收标准验证")
	if !ok || passed {
		t.Fatalf("expected FAIL to be parsed")
	}
	if !strings.Contains(reason, "缺少验收标准") {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestAllowQualitySoftPass(t *testing.T) {
	report := `
# 质量门禁报告
**总分**: 64/100

## 检查结果摘要
- 通过: 4 项
- 警告: 6 项
- 失败: 1 项

### code_quality
| 检查项 | 状态 | 得分 | 说明 |
|:---|:---:|:---:|:---|
| Python 语法检查 | ✗ | 20/100 | compileall 语法检查 |
`
	passed, reason := allowQualitySoftPass(report, "go", t.TempDir(), "")
	if !passed {
		t.Fatalf("expected soft pass for non-python backend")
	}
	if !strings.Contains(reason, "soft-pass") {
		t.Fatalf("expected soft-pass reason, got %s", reason)
	}
}

func TestAllowQualitySoftPassRejectsCriticalFailures(t *testing.T) {
	report := `
# 质量门禁报告
**总分**: 41/100

## 检查结果摘要
- 通过: 1 项
- 警告: 7 项
- 失败: 3 项

## 关键失败项
- [documentation] 产品需求文档存在性
- [documentation] 架构设计文档存在性

### code_quality
| 检查项 | 状态 | 得分 | 说明 |
|:---|:---:|:---:|:---|
| Python 语法检查 | ✗ | 20/100 | compileall 语法检查 |
`
	passed, _ := allowQualitySoftPass(report, "go", t.TempDir(), "")
	if passed {
		t.Fatalf("expected soft pass to be rejected when critical failures exist")
	}
}

func TestAllowQualitySoftPassForSpecTaskFalsePositive(t *testing.T) {
	projectDir := t.TempDir()
	changeID := "demo-change"
	taskDir := filepath.Join(projectDir, ".super-dev", "changes", changeID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	tasksContent := `# Tasks

- [x] 1. done
- [x] 2. done
`
	if err := os.WriteFile(filepath.Join(taskDir, "tasks.md"), []byte(tasksContent), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	report := `
# 质量门禁报告
**总分**: 62/100

## 检查结果摘要
- 通过: 3 项
- 警告: 6 项
- 失败: 2 项

## 关键失败项
- [testing] Spec 任务闭环状态

### testing
| 检查项 | 状态 | 得分 | 说明 |
|:---|:---:|:---:|:---|
| Spec任务完成度 | ✗ | 75/100 | Spec 任务闭环状态 |

### code_quality
| 检查项 | 状态 | 得分 | 说明 |
|:---|:---:|:---:|:---|
| Python 语法检查 | ✗ | 20/100 | compileall 语法检查 |
`
	reportPath := filepath.Join(projectDir, "output", changeID+"-quality-gate.md")
	passed, reason := allowQualitySoftPass(report, "go", projectDir, reportPath)
	if !passed {
		t.Fatalf("expected soft pass for closed spec tasks + non-python compileall failure")
	}
	if !strings.Contains(reason, "Spec task closure") {
		t.Fatalf("expected reason to mention spec task closure, got %s", reason)
	}
}

func TestAllowQualitySoftPassRejectsOpenSpecTasks(t *testing.T) {
	projectDir := t.TempDir()
	changeID := "demo-change"
	taskDir := filepath.Join(projectDir, ".super-dev", "changes", changeID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	tasksContent := `# Tasks

- [x] 1. done
- [ ] 2. todo
`
	if err := os.WriteFile(filepath.Join(taskDir, "tasks.md"), []byte(tasksContent), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	report := `
# 质量门禁报告
**总分**: 62/100

## 检查结果摘要
- 通过: 3 项
- 警告: 6 项
- 失败: 1 项

## 关键失败项
- [testing] Spec 任务闭环状态

### testing
| 检查项 | 状态 | 得分 | 说明 |
|:---|:---:|:---:|:---|
| Spec任务完成度 | ✗ | 75/100 | Spec 任务闭环状态 |
`
	reportPath := filepath.Join(projectDir, "output", changeID+"-quality-gate.md")
	passed, _ := allowQualitySoftPass(report, "go", projectDir, reportPath)
	if passed {
		t.Fatalf("expected soft pass rejection when current change tasks are not closed")
	}
}

func TestExtractChangeIDSupportsUnicode(t *testing.T) {
	lines := []string{
		"✓ 变更 ID: 实现一款提醒事项工具-使用适配移动端的方式开发-提供网页版本",
	}
	changeID := extractChangeID(lines)
	if changeID == "" {
		t.Fatalf("expected change_id to be extracted from unicode line")
	}
	if !strings.Contains(changeID, "实现一款提醒事项工具") {
		t.Fatalf("unexpected change_id: %s", changeID)
	}
}

func waitForRunCompletion(t *testing.T, s *store.Store, runID string) store.PipelineRun {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		updated, getErr := s.GetPipelineRun(ctx, runID)
		if getErr != nil {
			t.Fatalf("get run: %v", getErr)
		}
		if updated.Status == "completed" || updated.Status == "failed" {
			return updated
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("run did not complete before timeout")
	return store.PipelineRun{}
}
