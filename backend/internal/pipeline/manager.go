package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/store"
)

type Manager struct {
	store      *store.Store
	runner     Runner
	contextOpt *contextopt.Service
	llmAdvisor LLMAdvisor
	phases     []string
	phaseDelay time.Duration
}

type LLMAdvisor interface {
	Advise(ctx context.Context, prompt string) (string, error)
}

type ContextMode string

const (
	ContextModeOff    ContextMode = "off"
	ContextModeAuto   ContextMode = "auto"
	ContextModeManual ContextMode = "manual"
)

type ContextOptions struct {
	Mode            ContextMode
	Query           string
	TokenBudget     int
	MaxItems        int
	DynamicByPhase  bool
	MemoryWriteback bool
}

type StartRequest struct {
	ProjectID string
	Prompt    string
	Simulate  bool
	RetryOf   string
	Context   ContextOptions
	Lifecycle LifecycleOptions
	Options   RunRequest
}

type LifecycleOptions struct {
	OneClickDelivery bool
	StepByStep       bool
	IterationLimit   int
}

func NewManager(s *store.Store, runner Runner, contextOpt *contextopt.Service) *Manager {
	return &Manager{
		store:      s,
		runner:     runner,
		contextOpt: contextOpt,
		phases: []string{
			"phase-0-discovery",
			"phase-1-intelligence",
			"phase-2-document-drafting",
			"phase-3-frontend-scaffold",
			"phase-4-spec-generation",
			"phase-5-implementation-pack",
			"phase-6-redteam",
			"phase-7-quality-gate",
			"phase-8-code-review-guide",
			"phase-9-ai-prompts",
			"phase-10-cicd",
			"phase-11-delivery",
		},
		phaseDelay: 600 * time.Millisecond,
	}
}

func (m *Manager) SetPhaseDelay(delay time.Duration) {
	if delay < 0 {
		return
	}
	m.phaseDelay = delay
}

func (m *Manager) SetLLMAdvisor(advisor LLMAdvisor) {
	m.llmAdvisor = advisor
}

func (m *Manager) Start(ctx context.Context, req StartRequest) (store.PipelineRun, error) {
	mode := req.Context.Mode
	if mode == "" {
		mode = ContextModeOff
	}

	run, err := m.store.CreatePipelineRun(ctx, store.PipelineRun{
		ProjectID:          req.ProjectID,
		Prompt:             req.Prompt,
		Simulate:           req.Simulate,
		ProjectDir:         strings.TrimSpace(req.Options.ProjectDir),
		Platform:           strings.TrimSpace(req.Options.Platform),
		Frontend:           strings.TrimSpace(req.Options.Frontend),
		Backend:            strings.TrimSpace(req.Options.Backend),
		Domain:             strings.TrimSpace(req.Options.Domain),
		ContextMode:        string(mode),
		ContextQuery:       strings.TrimSpace(req.Context.Query),
		ContextTokenBudget: req.Context.TokenBudget,
		ContextMaxItems:    req.Context.MaxItems,
		ContextDynamic:     req.Context.DynamicByPhase,
		MemoryWriteback:    req.Context.MemoryWriteback,
		FullCycle:          req.Lifecycle.OneClickDelivery,
		StepByStep:         req.Lifecycle.StepByStep,
		IterationLimit:     req.Lifecycle.IterationLimit,
		RetryOf:            strings.TrimSpace(req.RetryOf),
		Status:             "queued",
		Progress:           0,
		Stage:              "queued",
	})
	if err != nil {
		return store.PipelineRun{}, err
	}

	go m.execute(run.ID, req)
	return run, nil
}

func (m *Manager) execute(runID string, req StartRequest) {
	ctx := context.Background()
	started := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", "starting", 1, &started, nil)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "starting",
		Status:  "running",
		Message: "Pipeline started",
	})

	executionPrompt, pack, contextErr := m.preparePromptWithContext(ctx, req)
	if contextErr != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "context-optimizer",
			Status:  "failed",
			Message: fmt.Sprintf("Context build failed: %v", contextErr),
		})
	} else if pack != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:  runID,
			Stage:  "context-optimizer",
			Status: "completed",
			Message: fmt.Sprintf(
				"Context injected (memories=%d, knowledge=%d, estimated_tokens=%d)",
				len(pack.Memories),
				len(pack.Knowledge),
				pack.EstimatedTokens,
			),
		})
	}

	phasePacks, phaseErr := m.buildPhaseContextPacks(ctx, req, strings.TrimSpace(req.Prompt))
	if phaseErr != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "context-optimizer-phase",
			Status:  "failed",
			Message: fmt.Sprintf("Phase context build failed: %v", phaseErr),
		})
	}
	if len(phasePacks) > 0 {
		executionPrompt = appendPhaseContextsToPrompt(executionPrompt, phasePacks)
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:  runID,
			Stage:  "context-optimizer-phase",
			Status: "completed",
			Message: fmt.Sprintf(
				"Dynamic phase context injected for %d phases",
				len(phasePacks),
			),
		})
	}

	req.Options.Prompt = executionPrompt

	if req.Simulate {
		m.runSimulation(ctx, runID, req, phasePacks)
		return
	}
	if req.Lifecycle.StepByStep {
		m.runStepByStepLifecycle(ctx, runID, req, phasePacks)
		return
	}
	if req.Lifecycle.OneClickDelivery {
		m.runOneClickLifecycle(ctx, runID, req, phasePacks)
		return
	}
	m.runWithSuperDev(ctx, runID, req, phasePacks)
}

func (m *Manager) preparePromptWithContext(ctx context.Context, req StartRequest) (string, *store.ContextPack, error) {
	basePrompt := strings.TrimSpace(req.Prompt)
	if basePrompt == "" {
		basePrompt = strings.TrimSpace(req.Options.Prompt)
	}
	if basePrompt == "" {
		basePrompt = "Please execute the planned software delivery pipeline."
	}

	mode := req.Context.Mode
	if mode == "" {
		mode = ContextModeOff
	}
	if mode == ContextModeOff || m.contextOpt == nil {
		return basePrompt, nil, nil
	}

	query := strings.TrimSpace(req.Context.Query)
	if query == "" || mode == ContextModeAuto {
		query = basePrompt
	}

	pack, err := m.contextOpt.BuildContextPack(ctx, contextopt.BuildRequest{
		ProjectID:   req.ProjectID,
		Query:       query,
		TokenBudget: req.Context.TokenBudget,
		MaxItems:    req.Context.MaxItems,
	})
	if err != nil {
		return basePrompt, nil, err
	}

	composed := composePrompt(basePrompt, pack)
	return composed, &pack, nil
}

type PhaseContextPack struct {
	Stage string
	Query string
	Pack  store.ContextPack
}

func (m *Manager) buildPhaseContextPacks(ctx context.Context, req StartRequest, basePrompt string) ([]PhaseContextPack, error) {
	mode := req.Context.Mode
	if mode == "" || mode == ContextModeOff || m.contextOpt == nil || !req.Context.DynamicByPhase {
		return nil, nil
	}

	selectedPhases := []string{
		"phase-0-discovery",
		"phase-2-document-drafting",
		"phase-5-implementation-pack",
		"phase-6-redteam",
		"phase-7-quality-gate",
		"phase-11-delivery",
	}

	packs := make([]PhaseContextPack, 0, len(selectedPhases))
	var errs []string
	for _, phase := range selectedPhases {
		query := m.resolvePhaseQuery(phase, mode, basePrompt, req.Context.Query)
		pack, err := m.contextOpt.BuildContextPack(ctx, contextopt.BuildRequest{
			ProjectID:   req.ProjectID,
			Query:       query,
			TokenBudget: req.Context.TokenBudget,
			MaxItems:    req.Context.MaxItems,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", phase, err))
			continue
		}
		if len(pack.Memories) == 0 && len(pack.Knowledge) == 0 {
			continue
		}
		packs = append(packs, PhaseContextPack{
			Stage: phase,
			Query: query,
			Pack:  pack,
		})
	}

	if len(errs) > 0 {
		return packs, errors.New(strings.Join(errs, "; "))
	}
	return packs, nil
}

func (m *Manager) resolvePhaseQuery(phase string, mode ContextMode, basePrompt, manualQuery string) string {
	phaseHints := map[string]string{
		"phase-0-discovery":           "业务目标 用户价值 关键需求",
		"phase-2-document-drafting":   "PRD 架构 文档 边界约束",
		"phase-5-implementation-pack": "代码实现 接口设计 数据模型",
		"phase-6-redteam":             "安全风险 攻击面 漏洞预防",
		"phase-7-quality-gate":        "测试策略 验收标准 质量门禁",
		"phase-11-delivery":           "发布流程 部署回滚 交付清单",
	}
	hint := phaseHints[phase]
	if mode == ContextModeManual && strings.TrimSpace(manualQuery) != "" {
		return strings.TrimSpace(manualQuery + " " + hint)
	}
	return strings.TrimSpace(basePrompt + " " + hint)
}

func composePrompt(base string, pack store.ContextPack) string {
	parts := []string{
		strings.TrimSpace(base),
		"",
		"---",
		"上下文优化摘要（SuperDev Studio 自动注入）:",
		pack.Summary,
	}

	if len(pack.Memories) > 0 {
		parts = append(parts, "", "记忆片段:")
		for idx, m := range pack.Memories {
			if idx >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("- [%s|importance=%.1f] %s", m.Role, m.Importance, m.Content))
		}
	}
	if len(pack.Knowledge) > 0 {
		parts = append(parts, "", "知识片段:")
		for idx, k := range pack.Knowledge {
			if idx >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("- [doc=%s chunk=%d] %s", k.DocumentID, k.ChunkIndex, k.Content))
		}
	}

	parts = append(parts, "", "---", "请优先遵循以上上下文进行实现。")
	return strings.Join(parts, "\n")
}

func appendPhaseContextsToPrompt(prompt string, phasePacks []PhaseContextPack) string {
	if len(phasePacks) == 0 {
		return prompt
	}
	lines := []string{prompt, "", "阶段动态上下文（按关键阶段自动召回）:"}
	for _, phasePack := range phasePacks {
		lines = append(lines, "", fmt.Sprintf("### %s", phasePack.Stage))
		lines = append(lines, fmt.Sprintf("- 查询: %s", phasePack.Query))
		lines = append(lines, fmt.Sprintf("- 摘要: %s", phasePack.Pack.Summary))
	}
	lines = append(lines, "", "请在各阶段优先参考对应上下文。")
	return strings.Join(lines, "\n")
}

func (m *Manager) runOneClickLifecycle(ctx context.Context, runID string, req StartRequest, phasePacks []PhaseContextPack) {
	iterationLimit := req.Lifecycle.IterationLimit
	if iterationLimit <= 0 {
		iterationLimit = 3
	}
	lifecyclePrompt := resolveLifecyclePrompt(req)
	resolvedProjectName := ""

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "lifecycle",
		Status:  "running",
		Message: "One-click full lifecycle started (design -> iterate -> test -> acceptance -> release)",
	})

	designLines, err := m.runCommandStageWithLines(
		ctx,
		runID,
		"lifecycle-design",
		req.Options,
		buildPipelineCommandArgs(req.Options, lifecyclePrompt, true, true, true),
		15,
	)
	if err != nil {
		m.failRun(ctx, runID, req, "lifecycle-design", err, phasePacks)
		return
	}
	resolvedProjectName = extractPipelineProjectName(designLines)
	changeID := resolveChangeIDFromLinesOrLatest(req.Options.ProjectDir, designLines)
	if strings.TrimSpace(changeID) == "" {
		changeID = buildChangeID(lifecyclePrompt)
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "lifecycle-task-backlog",
			Status:  "log",
			Message: "Fallback change_id generated for backlog planning: " + changeID,
		})
	}
	docsBrief := buildCreateDocsBrief(req.Options.ProjectDir, designLines)
	if _, taskErr := m.ensureProjectTasksFromDocs(ctx, runID, req, changeID, docsBrief, "lifecycle-task-backlog"); taskErr != nil {
		m.failRun(ctx, runID, req, "lifecycle-task-backlog", taskErr, phasePacks)
		return
	}

	qualityPassed := false
	lastQualitySummary := ""
	for idx := 1; idx <= iterationLimit; idx++ {
		progress := 15 + int(float64(idx)/float64(iterationLimit)*50)
		iterationStage := fmt.Sprintf("lifecycle-iteration-%d", idx)
		guidance := m.generateIterationGuidance(ctx, req, idx, lastQualitySummary)
		if strings.TrimSpace(guidance) != "" {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   iterationStage,
				Status:  "log",
				Message: "LLM iteration guidance: " + guidance,
			})
		}

		iterationPrompt := lifecyclePrompt
		if strings.TrimSpace(guidance) != "" {
			iterationPrompt = strings.TrimSpace(iterationPrompt + "\n\n迭代修复目标:\n" + guidance)
		}
		iterationLines, err := m.runCommandStageWithLines(
			ctx,
			runID,
			iterationStage,
			req.Options,
			buildPipelineCommandArgs(req.Options, iterationPrompt, true, true, false),
			progress,
		)
		if err != nil {
			m.failRun(ctx, runID, req, iterationStage, err, phasePacks)
			return
		}
		if parsedName := extractPipelineProjectName(iterationLines); strings.TrimSpace(parsedName) != "" {
			resolvedProjectName = parsedName
		}
		m.syncSuperDevProjectName(ctx, runID, req.Options, resolvedProjectName)

		qualityLines, qualityErr := m.runCommandStageWithLines(
			ctx,
			runID,
			fmt.Sprintf("lifecycle-quality-%d", idx),
			req.Options,
			[]string{"quality", "--type", "all"},
			progress+5,
		)
		lastQualitySummary = summarizeQualityOutput(qualityLines)
		var qualityDecision string
		qualityPassed, qualityDecision = m.isQualityGatePassed(
			req.Options.ProjectDir,
			req.Prompt,
			req.Options.Backend,
			qualityErr,
			qualityLines,
		)
		if strings.TrimSpace(qualityDecision) != "" {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "lifecycle-quality",
				Status:  "log",
				Message: qualityDecision,
			})
		}

		if qualityPassed {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "lifecycle-quality",
				Status:  "completed",
				Message: fmt.Sprintf("Quality gate passed on iteration %d", idx),
			})
			break
		}
		if qualityErr != nil {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "lifecycle-quality",
				Status:  "log",
				Message: fmt.Sprintf("Quality gate still failing on iteration %d: %v", idx, qualityErr),
			})
		} else {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "lifecycle-quality",
				Status:  "log",
				Message: fmt.Sprintf("Quality gate not passed on iteration %d, continue fixing", idx),
			})
		}
	}

	if !qualityPassed {
		m.failRun(
			ctx,
			runID,
			req,
			"lifecycle-quality",
			fmt.Errorf("quality gate not passed after %d iterations", iterationLimit),
			phasePacks,
		)
		return
	}

	acceptanceSummary := m.generateAcceptanceSummary(ctx, req, lastQualitySummary)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "lifecycle-acceptance",
		Status:  "completed",
		Message: acceptanceSummary,
	})
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", "lifecycle-acceptance", 80, nil, nil)

	if err := m.runCommandStage(
		ctx,
		runID,
		"lifecycle-release",
		req.Options,
		[]string{"deploy", "--docker", "--cicd", "all"},
		90,
	); err != nil {
		m.failRun(ctx, runID, req, "lifecycle-release", err, phasePacks)
		return
	}

	if err := m.runCommandStage(
		ctx,
		runID,
		"lifecycle-preview",
		req.Options,
		[]string{"preview", "--output", "output/preview.html"},
		95,
	); err != nil {
		m.failRun(ctx, runID, req, "lifecycle-preview", err, phasePacks)
		return
	}

	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "done",
		Status:  "completed",
		Message: "One-click lifecycle finished",
	})
}

func (m *Manager) runStepByStepLifecycle(ctx context.Context, runID string, req StartRequest, phasePacks []PhaseContextPack) {
	iterationLimit := req.Lifecycle.IterationLimit
	if iterationLimit <= 0 {
		iterationLimit = 3
	}
	lifecyclePrompt := resolveLifecyclePrompt(req)

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-by-step",
		Status:  "running",
		Message: "Step-by-step lifecycle started (create -> spec validate -> task status -> task run -> quality -> preview -> deploy)",
	})

	createLines, err := m.runCommandStageWithLines(
		ctx,
		runID,
		"step-create",
		req.Options,
		buildCreateCommandArgs(req.Options, lifecyclePrompt),
		15,
	)
	if err != nil {
		m.failRun(ctx, runID, req, "step-create", err, phasePacks)
		return
	}

	changeID := resolveChangeIDFromLinesOrLatest(req.Options.ProjectDir, createLines)
	if strings.TrimSpace(changeID) == "" {
		m.failRun(
			ctx,
			runID,
			req,
			"step-create",
			errors.New("unable to resolve change_id from create output"),
			phasePacks,
		)
		return
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-create",
		Status:  "log",
		Message: "Resolved change_id: " + changeID,
	})

	docsBrief := buildCreateDocsBrief(req.Options.ProjectDir, createLines)
	if strings.TrimSpace(docsBrief) != "" {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-docs",
			Status:  "completed",
			Message: "Initial design documents loaded for task planning",
		})
	}

	if err := m.runCommandStage(
		ctx,
		runID,
		"step-spec-validate",
		req.Options,
		buildSpecValidateCommandArgs(changeID),
		25,
	); err != nil {
		m.failRun(ctx, runID, req, "step-spec-validate", err, phasePacks)
		return
	}

	if err := m.runCommandStage(
		ctx,
		runID,
		"step-task-status-init",
		req.Options,
		buildTaskStatusCommandArgs(changeID),
		30,
	); err != nil {
		m.failRun(ctx, runID, req, "step-task-status-init", err, phasePacks)
		return
	}

	projectTasks, taskErr := m.ensureProjectTasksFromDocs(ctx, runID, req, changeID, docsBrief, "step-task-backlog")
	if taskErr != nil {
		m.failRun(ctx, runID, req, "step-task-backlog", taskErr, phasePacks)
		return
	}
	if len(projectTasks) == 0 {
		m.failRun(ctx, runID, req, "step-task-backlog", errors.New("no project tasks available for execution"), phasePacks)
		return
	}

	kickoffGuidance := m.generateStepByStepKickoffGuidance(ctx, req, changeID, docsBrief)
	if strings.TrimSpace(kickoffGuidance) != "" {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-agent",
			Status:  "log",
			Message: kickoffGuidance,
		})
	}

	qualitySummary := ""
	for taskIndex, task := range projectTasks {
		if strings.EqualFold(strings.TrimSpace(task.Status), "done") {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "step-project-task",
				Status:  "log",
				Message: fmt.Sprintf("Skip completed project task: %s", task.Title),
			})
			continue
		}

		if _, updateErr := m.store.UpdateTask(ctx, task.ID, "in_progress", "", ""); updateErr == nil {
			task.Status = "in_progress"
		}
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-project-task",
			Status:  "running",
			Message: fmt.Sprintf("Task %d/%d: %s", taskIndex+1, len(projectTasks), task.Title),
		})

		taskPassed := false
		for attempt := 1; attempt <= iterationLimit; attempt++ {
			progress := calcTaskProgress(taskIndex, len(projectTasks), attempt, iterationLimit)
			taskStage := fmt.Sprintf("step-task-run-%d-%d", taskIndex+1, attempt)
			taskStatusStage := fmt.Sprintf("step-task-status-%d-%d", taskIndex+1, attempt)
			qualityStage := fmt.Sprintf("step-quality-%d-%d", taskIndex+1, attempt)

			taskGuidance := m.generateTaskExecutionGuidance(ctx, req, changeID, task, docsBrief, qualitySummary)
			if strings.TrimSpace(taskGuidance) != "" {
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   "step-agent",
					Status:  "log",
					Message: fmt.Sprintf("Task %d attempt %d guidance: %s", taskIndex+1, attempt, taskGuidance),
				})
			}

			if err := m.runCommandStage(
				ctx,
				runID,
				taskStage,
				req.Options,
				buildTaskRunCommandArgsWithRetries(req.Options, changeID, minInt(attempt, 3)),
				progress,
			); err != nil {
				m.failRun(ctx, runID, req, taskStage, err, phasePacks)
				return
			}

			if err := m.runCommandStage(
				ctx,
				runID,
				taskStatusStage,
				req.Options,
				buildTaskStatusCommandArgs(changeID),
				progress+2,
			); err != nil {
				m.failRun(ctx, runID, req, taskStatusStage, err, phasePacks)
				return
			}

			qualityLines, qualityErr := m.runCommandStageWithLines(
				ctx,
				runID,
				qualityStage,
				req.Options,
				[]string{"quality", "--type", "all"},
				progress+5,
			)
			qualitySummary = summarizeQualityOutput(qualityLines)
			qualityDecisionPassed, qualityDecision := m.isQualityGatePassed(
				req.Options.ProjectDir,
				req.Prompt,
				req.Options.Backend,
				qualityErr,
				qualityLines,
			)
			if strings.TrimSpace(qualityDecision) != "" {
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   qualityStage,
					Status:  "log",
					Message: qualityDecision,
				})
			}

			if qualityDecisionPassed {
				verified, verifyReason := m.verifyTaskCompletionAgainstRequirement(ctx, req, task, qualitySummary)
				verifyStatus := "completed"
				if !verified {
					verifyStatus = "failed"
				}
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   "step-task-verify",
					Status:  verifyStatus,
					Message: fmt.Sprintf("Task verify %s (attempt %d): %s", task.Title, attempt, verifyReason),
				})

				if verified {
					taskPassed = true
					if _, updateErr := m.store.UpdateTask(ctx, task.ID, "done", "", ""); updateErr == nil {
						task.Status = "done"
					}
					_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
						RunID:   runID,
						Stage:   "step-project-task",
						Status:  "completed",
						Message: fmt.Sprintf("Task completed: %s", task.Title),
					})
					break
				}

				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   qualityStage,
					Status:  "log",
					Message: fmt.Sprintf("Task requirement verification not passed on task %d attempt %d", taskIndex+1, attempt),
				})
			}

			repairGuidance := m.generateStepByStepRepairGuidance(
				ctx,
				req,
				changeID,
				docsBrief,
				taskIndex*iterationLimit+attempt,
				qualitySummary,
			)
			if strings.TrimSpace(repairGuidance) != "" {
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   "step-agent",
					Status:  "log",
					Message: repairGuidance,
				})
			}
			if qualityErr != nil {
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   qualityStage,
					Status:  "log",
					Message: fmt.Sprintf("Quality failed on task %d attempt %d: %v", taskIndex+1, attempt, qualityErr),
				})
			} else {
				_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
					RunID:   runID,
					Stage:   qualityStage,
					Status:  "log",
					Message: fmt.Sprintf("Quality not passed on task %d attempt %d", taskIndex+1, attempt),
				})
			}
		}

		if !taskPassed {
			_, _ = m.store.UpdateTask(ctx, task.ID, "todo", "", "")
			m.failRun(
				ctx,
				runID,
				req,
				"step-project-task",
				fmt.Errorf("task %s not completed after %d attempts", task.Title, iterationLimit),
				phasePacks,
			)
			return
		}
	}

	if strings.TrimSpace(qualitySummary) == "" {
		qualityLines, qualityErr := m.runCommandStageWithLines(
			ctx,
			runID,
			"step-quality-final",
			req.Options,
			[]string{"quality", "--type", "all"},
			78,
		)
		qualitySummary = summarizeQualityOutput(qualityLines)
		qualityPassed, qualityDecision := m.isQualityGatePassed(
			req.Options.ProjectDir,
			req.Prompt,
			req.Options.Backend,
			qualityErr,
			qualityLines,
		)
		if strings.TrimSpace(qualityDecision) != "" {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "step-quality-final",
				Status:  "log",
				Message: qualityDecision,
			})
		}
		if !qualityPassed {
			if qualityErr == nil {
				qualityErr = errors.New("quality gate not passed")
			}
			m.failRun(ctx, runID, req, "step-quality-final", qualityErr, phasePacks)
			return
		}
	}

	if err := m.buildNextIterationTaskBacklog(ctx, runID, req, changeID, docsBrief, qualitySummary); err != nil {
		m.failRun(ctx, runID, req, "step-task-backlog-next", err, phasePacks)
		return
	}

	acceptanceSummary := m.generateAcceptanceSummary(ctx, req, qualitySummary)
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", "step-acceptance", 80, nil, nil)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-acceptance",
		Status:  "completed",
		Message: acceptanceSummary,
	})

	if err := m.runCommandStage(
		ctx,
		runID,
		"step-preview",
		req.Options,
		[]string{"preview", "--output", "output/preview.html"},
		90,
	); err != nil {
		m.failRun(ctx, runID, req, "step-preview", err, phasePacks)
		return
	}

	if err := m.runCommandStage(
		ctx,
		runID,
		"step-release",
		req.Options,
		[]string{"deploy", "--docker", "--cicd", "all"},
		95,
	); err != nil {
		m.failRun(ctx, runID, req, "step-release", err, phasePacks)
		return
	}

	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "done",
		Status:  "completed",
		Message: "Step-by-step lifecycle finished",
	})
}

func (m *Manager) runCommandStage(
	ctx context.Context,
	runID string,
	stage string,
	options RunRequest,
	commandArgs []string,
	progress int,
) error {
	_, err := m.runCommandStageWithLines(ctx, runID, stage, options, commandArgs, progress)
	return err
}

func (m *Manager) runCommandStageWithLines(
	ctx context.Context,
	runID string,
	stage string,
	options RunRequest,
	commandArgs []string,
	progress int,
) ([]string, error) {
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", stage, progress, nil, nil)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   stage,
		Status:  "running",
		Message: "Executing super-dev " + strings.Join(commandArgs, " "),
	})

	lines, err := m.runner.RunCommand(ctx, options, commandArgs)
	for _, line := range lines {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   stage,
			Status:  "log",
			Message: line,
		})
	}
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   stage,
			Status:  "failed",
			Message: err.Error(),
		})
		return lines, err
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   stage,
		Status:  "completed",
		Message: "Stage command completed",
	})
	return lines, nil
}

func (m *Manager) failRun(
	ctx context.Context,
	runID string,
	req StartRequest,
	stage string,
	err error,
	phasePacks []PhaseContextPack,
) {
	m.writebackRunMemory(ctx, req, runID, "failed", stage, err.Error(), phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "failed", stage, 100, nil, &finished)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   stage,
		Status:  "failed",
		Message: err.Error(),
	})
}

func buildPipelineCommandArgs(options RunRequest, prompt string, skipQualityGate, skipRedteam, skipScaffold bool) []string {
	args := []string{"pipeline", strings.TrimSpace(prompt)}
	if strings.TrimSpace(options.Platform) != "" {
		args = append(args, "--platform", strings.TrimSpace(options.Platform))
	}
	if strings.TrimSpace(options.Frontend) != "" {
		args = append(args, "--frontend", strings.TrimSpace(options.Frontend))
	}
	if strings.TrimSpace(options.Backend) != "" {
		args = append(args, "--backend", strings.TrimSpace(options.Backend))
	}
	if strings.TrimSpace(options.Domain) != "" {
		args = append(args, "--domain", strings.TrimSpace(options.Domain))
	}
	if skipQualityGate {
		args = append(args, "--skip-quality-gate")
	}
	if skipRedteam {
		args = append(args, "--skip-redteam")
	}
	if skipScaffold {
		args = append(args, "--skip-scaffold")
	}
	return args
}

func buildCreateCommandArgs(options RunRequest, prompt string) []string {
	args := []string{"create", strings.TrimSpace(prompt)}
	if strings.TrimSpace(options.Platform) != "" {
		args = append(args, "--platform", strings.TrimSpace(options.Platform))
	}
	if strings.TrimSpace(options.Frontend) != "" {
		args = append(args, "--frontend", strings.TrimSpace(options.Frontend))
	}
	if strings.TrimSpace(options.Backend) != "" {
		args = append(args, "--backend", strings.TrimSpace(options.Backend))
	}
	if strings.TrimSpace(options.Domain) != "" {
		args = append(args, "--domain", strings.TrimSpace(options.Domain))
	}
	return args
}

func buildTaskRunCommandArgs(options RunRequest, changeID string) []string {
	return buildTaskRunCommandArgsWithRetries(options, changeID, 1)
}

func buildTaskRunCommandArgsWithRetries(options RunRequest, changeID string, maxRetries int) []string {
	if maxRetries <= 0 {
		maxRetries = 1
	}
	if maxRetries > 5 {
		maxRetries = 5
	}
	args := []string{"task", "run", strings.TrimSpace(changeID), "--max-retries", strconv.Itoa(maxRetries)}
	if strings.TrimSpace(options.Platform) != "" {
		args = append(args, "--platform", strings.TrimSpace(options.Platform))
	}
	if strings.TrimSpace(options.Frontend) != "" {
		args = append(args, "--frontend", strings.TrimSpace(options.Frontend))
	}
	if strings.TrimSpace(options.Backend) != "" {
		args = append(args, "--backend", strings.TrimSpace(options.Backend))
	}
	if strings.TrimSpace(options.Domain) != "" {
		args = append(args, "--domain", strings.TrimSpace(options.Domain))
	}
	return args
}

func buildTaskStatusCommandArgs(changeID string) []string {
	return []string{"task", "status", strings.TrimSpace(changeID)}
}

func buildSpecValidateCommandArgs(changeID string) []string {
	return []string{"spec", "validate", strings.TrimSpace(changeID)}
}

func buildCreateDocsBrief(projectDir string, createLines []string) string {
	docPaths := extractDocPathsFromCreateOutput(projectDir, createLines)
	if len(docPaths) == 0 {
		return ""
	}

	sections := make([]string, 0, len(docPaths))
	for _, docPath := range docPaths {
		content, err := os.ReadFile(docPath)
		if err != nil {
			continue
		}
		snippet := truncateForPrompt(string(content), 800)
		if strings.TrimSpace(snippet) == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("[%s]\n%s", filepath.Base(docPath), snippet))
		if len(strings.Join(sections, "\n\n")) >= 2600 {
			break
		}
	}
	return strings.Join(sections, "\n\n")
}

func extractDocPathsFromCreateOutput(projectDir string, createLines []string) []string {
	baseDir := resolveProjectDir(projectDir)
	docPattern := regexp.MustCompile(`(?:PRD|架构|UI/UX|执行路线图|前端蓝图)\s*:\s*(.+)$`)
	docPaths := []string{}
	seen := map[string]struct{}{}

	for _, rawLine := range createLines {
		line := stripANSIEscapeCodes(strings.TrimSpace(rawLine))
		match := docPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		pathValue := strings.Trim(strings.TrimSpace(match[1]), `"'`)
		if pathValue == "" {
			continue
		}
		if !filepath.IsAbs(pathValue) {
			pathValue = filepath.Join(baseDir, pathValue)
		}
		cleanPath := filepath.Clean(pathValue)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		if info, err := os.Stat(cleanPath); err != nil || info.IsDir() {
			continue
		}
		seen[cleanPath] = struct{}{}
		docPaths = append(docPaths, cleanPath)
	}

	if len(docPaths) > 0 {
		return docPaths
	}

	fallbackMatches, err := filepath.Glob(filepath.Join(baseDir, "output", "*-*.md"))
	if err != nil {
		return []string{}
	}
	for _, candidate := range fallbackMatches {
		baseName := strings.ToLower(filepath.Base(candidate))
		if !(strings.HasSuffix(baseName, "-prd.md") ||
			strings.HasSuffix(baseName, "-architecture.md") ||
			strings.HasSuffix(baseName, "-uiux.md") ||
			strings.HasSuffix(baseName, "-execution-plan.md") ||
			strings.HasSuffix(baseName, "-frontend-blueprint.md")) {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		docPaths = append(docPaths, candidate)
	}
	return docPaths
}

func truncateForPrompt(content string, maxRunes int) string {
	trimmed := strings.TrimSpace(content)
	if maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "\n...(truncated)"
}

type taskDraft struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

func (m *Manager) ensureProjectTasksFromDocs(
	ctx context.Context,
	runID string,
	req StartRequest,
	changeID string,
	docsBrief string,
	stage string,
) ([]store.Task, error) {
	stageName := strings.TrimSpace(stage)
	if stageName == "" {
		stageName = "step-task-backlog"
	}

	existingTasks, err := m.store.ListTasks(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if len(existingTasks) > 0 {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   stageName,
			Status:  "completed",
			Message: fmt.Sprintf("Use existing project backlog (%d tasks)", len(existingTasks)),
		})
		return sortTasksForExecution(existingTasks), nil
	}

	drafts := m.generateTaskBacklogDrafts(ctx, req, changeID, docsBrief)
	created := 0
	for _, draft := range drafts {
		if strings.TrimSpace(draft.Title) == "" {
			continue
		}
		_, createErr := m.store.CreateTask(ctx, store.Task{
			ProjectID:   req.ProjectID,
			Title:       strings.TrimSpace(draft.Title),
			Description: strings.TrimSpace(draft.Description),
			Status:      "todo",
			Priority:    normalizeTaskPriority(draft.Priority),
			Assignee:    "agent",
		})
		if createErr != nil {
			return nil, createErr
		}
		created++
	}

	tasks, err := m.store.ListTasks(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   stageName,
		Status:  "completed",
		Message: fmt.Sprintf("Created %d project tasks from initial docs", created),
	})
	return sortTasksForExecution(tasks), nil
}

func sortTasksForExecution(tasks []store.Task) []store.Task {
	sorted := append([]store.Task{}, tasks...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})
	return sorted
}

func normalizeTaskPriority(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func (m *Manager) generateTaskBacklogDrafts(
	ctx context.Context,
	req StartRequest,
	changeID string,
	docsBrief string,
) []taskDraft {
	if m.llmAdvisor == nil {
		return fallbackTaskBacklog(req.Prompt)
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是项目交付智能体。请基于需求和初始化文档，拆分 3-8 个可执行项目任务，用于驱动后续 super-dev 开发。"+
			"\n需求: %s\nchange_id: %s\n初始化文档摘要:\n%s\n"+
			"\n输出必须是 JSON 数组，不要任何额外说明。字段：title, description, priority(high|medium|low)。",
		req.Prompt,
		changeID,
		truncateForPrompt(docsBrief, 2800),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil {
		return fallbackTaskBacklog(req.Prompt)
	}
	drafts, parseErr := parseTaskDrafts(answer)
	if parseErr != nil || len(drafts) == 0 {
		return fallbackTaskBacklog(req.Prompt)
	}
	return drafts
}

func (m *Manager) verifyTaskCompletionAgainstRequirement(
	ctx context.Context,
	req StartRequest,
	task store.Task,
	qualitySummary string,
) (bool, string) {
	if m.llmAdvisor == nil {
		return true, "quality gate passed and verification fallback accepted"
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是交付验收评审。请判断当前任务是否已满足需求并可标记完成。"+
			"\n项目需求: %s\n任务标题: %s\n任务描述: %s\n质量摘要: %s"+
			"\n请仅输出一行：PASS:原因 或 FAIL:原因。",
		strings.TrimSpace(req.Prompt),
		strings.TrimSpace(task.Title),
		strings.TrimSpace(task.Description),
		strings.TrimSpace(qualitySummary),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return true, "quality gate passed and verification fallback accepted"
	}
	decision, reason, ok := parseTaskVerificationAnswer(answer)
	if !ok {
		return true, "quality gate passed and verification fallback accepted"
	}
	return decision, reason
}

func parseTaskVerificationAnswer(answer string) (bool, string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(answer, "\n", " "))
	if normalized == "" {
		return false, "", false
	}
	upper := strings.ToUpper(normalized)
	if strings.HasPrefix(upper, "PASS:") {
		reason := strings.TrimSpace(normalized[len("PASS:"):])
		if reason == "" {
			reason = "verified by acceptance checker"
		}
		return true, reason, true
	}
	if strings.HasPrefix(upper, "FAIL:") {
		reason := strings.TrimSpace(normalized[len("FAIL:"):])
		if reason == "" {
			reason = "requirement verification failed"
		}
		return false, reason, true
	}
	return false, "", false
}

func (m *Manager) buildNextIterationTaskBacklog(
	ctx context.Context,
	runID string,
	req StartRequest,
	changeID string,
	docsBrief string,
	qualitySummary string,
) error {
	tasks, err := m.store.ListTasks(ctx, req.ProjectID)
	if err != nil {
		return err
	}

	openCount := 0
	for _, item := range tasks {
		if strings.EqualFold(strings.TrimSpace(item.Status), "done") {
			continue
		}
		openCount++
	}
	if openCount > 0 {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-task-backlog-next",
			Status:  "completed",
			Message: fmt.Sprintf("Skip next-iteration task generation because %d open tasks remain", openCount),
		})
		return nil
	}

	drafts := m.generateFollowupTaskDrafts(ctx, req, changeID, docsBrief, tasks, qualitySummary)
	if len(drafts) == 0 {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-task-backlog-next",
			Status:  "completed",
			Message: "No next-iteration task drafts generated",
		})
		return nil
	}

	existingTitleMap := make(map[string]struct{}, len(tasks))
	for _, item := range tasks {
		key := strings.ToLower(strings.TrimSpace(item.Title))
		if key != "" {
			existingTitleMap[key] = struct{}{}
		}
	}

	created := 0
	for _, draft := range drafts {
		title := strings.TrimSpace(draft.Title)
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if _, exists := existingTitleMap[key]; exists {
			continue
		}
		if _, createErr := m.store.CreateTask(ctx, store.Task{
			ProjectID:   req.ProjectID,
			Title:       title,
			Description: strings.TrimSpace(draft.Description),
			Status:      "todo",
			Priority:    normalizeTaskPriority(draft.Priority),
			Assignee:    "agent",
		}); createErr != nil {
			return createErr
		}
		existingTitleMap[key] = struct{}{}
		created++
	}

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-task-backlog-next",
		Status:  "completed",
		Message: fmt.Sprintf("Generated %d next-iteration tasks", created),
	})
	return nil
}

func (m *Manager) generateFollowupTaskDrafts(
	ctx context.Context,
	req StartRequest,
	changeID string,
	docsBrief string,
	existingTasks []store.Task,
	qualitySummary string,
) []taskDraft {
	if m.llmAdvisor == nil {
		return fallbackFollowupTaskBacklog(req.Prompt)
	}

	completed := make([]string, 0, 6)
	for _, item := range existingTasks {
		if !strings.EqualFold(strings.TrimSpace(item.Status), "done") {
			continue
		}
		completed = append(completed, strings.TrimSpace(item.Title))
		if len(completed) >= 6 {
			break
		}
	}
	completedSummary := "无"
	if len(completed) > 0 {
		completedSummary = strings.Join(completed, " | ")
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是项目推进规划助手。请根据当前完成情况规划下一批任务（3-6条），用于持续迭代推进。"+
			"\n项目需求: %s\nchange_id: %s\n已完成任务: %s\n质量摘要: %s\n初始化文档摘要:\n%s"+
			"\n输出必须是 JSON 数组，不要任何额外说明。字段：title, description, priority(high|medium|low)。",
		strings.TrimSpace(req.Prompt),
		strings.TrimSpace(changeID),
		completedSummary,
		strings.TrimSpace(qualitySummary),
		truncateForPrompt(docsBrief, 2200),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil {
		return fallbackFollowupTaskBacklog(req.Prompt)
	}
	drafts, parseErr := parseTaskDrafts(answer)
	if parseErr != nil || len(drafts) == 0 {
		return fallbackFollowupTaskBacklog(req.Prompt)
	}
	return drafts
}

func fallbackFollowupTaskBacklog(prompt string) []taskDraft {
	base := strings.TrimSpace(prompt)
	if base == "" {
		base = "当前项目需求"
	}
	return []taskDraft{
		{
			Title:       "下一迭代：补齐自动化测试与覆盖率",
			Description: fmt.Sprintf("围绕需求「%s」补齐核心路径单测、集成测试与回归测试，并输出覆盖率报告。", base),
			Priority:    "high",
		},
		{
			Title:       "下一迭代：强化稳定性与异常处理",
			Description: "完善错误处理、重试与回滚策略，验证边界场景并补充运行监控。",
			Priority:    "medium",
		},
		{
			Title:       "下一迭代：优化性能与发布验收",
			Description: "执行性能基线对比、清理质量门禁告警项，并准备上线验收清单。",
			Priority:    "medium",
		},
	}
}

func parseTaskDrafts(raw string) ([]taskDraft, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty task draft response")
	}

	if strings.Contains(trimmed, "```") {
		re := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
		match := re.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			trimmed = strings.TrimSpace(match[1])
		}
	}

	start := strings.Index(trimmed, "[")
	end := strings.LastIndex(trimmed, "]")
	if start >= 0 && end > start {
		trimmed = strings.TrimSpace(trimmed[start : end+1])
	}

	var drafts []taskDraft
	if err := json.Unmarshal([]byte(trimmed), &drafts); err != nil {
		return nil, err
	}

	cleaned := make([]taskDraft, 0, len(drafts))
	seen := map[string]struct{}{}
	for _, draft := range drafts {
		title := strings.TrimSpace(draft.Title)
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, taskDraft{
			Title:       title,
			Description: strings.TrimSpace(draft.Description),
			Priority:    normalizeTaskPriority(draft.Priority),
		})
		if len(cleaned) >= 8 {
			break
		}
	}
	if len(cleaned) == 0 {
		return nil, errors.New("no valid task drafts")
	}
	return cleaned, nil
}

func fallbackTaskBacklog(prompt string) []taskDraft {
	base := strings.TrimSpace(prompt)
	if base == "" {
		base = "当前项目需求"
	}
	return []taskDraft{
		{
			Title:       "细化需求与验收标准",
			Description: "基于初始化文档拆解核心场景、边界条件与验收标准，确保后续开发闭环。",
			Priority:    "high",
		},
		{
			Title:       "实现核心功能并联调接口",
			Description: fmt.Sprintf("围绕需求「%s」完成核心业务实现、接口联调和错误处理。", base),
			Priority:    "high",
		},
		{
			Title:       "补齐测试并准备上线",
			Description: "完善测试、通过质量门禁，完成预览验证和部署准备。",
			Priority:    "medium",
		},
	}
}

func calcTaskProgress(taskIndex, totalTasks, attempt, attemptLimit int) int {
	if totalTasks <= 0 {
		return 35
	}
	if attemptLimit <= 0 {
		attemptLimit = 1
	}
	totalSlots := totalTasks * attemptLimit
	currentSlot := taskIndex*attemptLimit + attempt
	if currentSlot > totalSlots {
		currentSlot = totalSlots
	}
	return 30 + int(float64(currentSlot)/float64(totalSlots)*45)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Manager) generateTaskExecutionGuidance(
	ctx context.Context,
	req StartRequest,
	changeID string,
	task store.Task,
	docsBrief string,
	qualitySummary string,
) string {
	if m.llmAdvisor == nil {
		return fmt.Sprintf("执行任务：%s。先运行 task run，再根据 quality 结果修复问题。", task.Title)
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是软件交付智能体。请针对当前项目任务生成 3-5 条可执行动作，用于驱动 super-dev 自动化推进。"+
			"\n需求: %s\nchange_id: %s\n当前任务: %s\n任务描述: %s\n最近质量结果: %s\n初始化文档摘要:\n%s"+
			"\n输出要求：中文条目列表，每条一句。",
		req.Prompt,
		changeID,
		task.Title,
		task.Description,
		strings.TrimSpace(qualitySummary),
		truncateForPrompt(docsBrief, 2200),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return fmt.Sprintf("执行任务：%s。先运行 task run，再根据 quality 结果修复问题。", task.Title)
	}
	return strings.TrimSpace(answer)
}

func (m *Manager) generateStepByStepKickoffGuidance(
	ctx context.Context,
	req StartRequest,
	changeID string,
	docsBrief string,
) string {
	if m.llmAdvisor == nil {
		return "已加载初始文档，已生成项目任务，接下来按任务逐步执行：task run -> task status -> quality。"
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是资深技术负责人。请基于需求和初始文档，输出 4-6 条后续推进建议（每条一句，面向 super-dev 执行）。\n需求: %s\nchange_id: %s\n初始文档摘要:\n%s\n输出要求：只输出条目，不要代码块。",
		req.Prompt,
		changeID,
		truncateForPrompt(docsBrief, 2600),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return "已加载初始文档，已生成项目任务，接下来按任务逐步执行：task run -> task status -> quality。"
	}
	return strings.TrimSpace(answer)
}

func (m *Manager) generateStepByStepRepairGuidance(
	ctx context.Context,
	req StartRequest,
	changeID string,
	docsBrief string,
	iteration int,
	qualitySummary string,
) string {
	if m.llmAdvisor == nil {
		return fmt.Sprintf(
			"第 %d 轮修复建议：围绕 change_id=%s 优先修复质量门禁失败项，补齐测试并再次执行 task run 与 quality。",
			iteration,
			changeID,
		)
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是资深技术负责人。当前执行 super-dev 逐步开发流程。\n需求: %s\nchange_id: %s\n迭代轮次: %d\n最近质量输出: %s\n初始文档摘要:\n%s\n请给出不超过 5 条下一轮修复动作，要求可直接执行。",
		req.Prompt,
		changeID,
		iteration,
		strings.TrimSpace(qualitySummary),
		truncateForPrompt(docsBrief, 2200),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return fmt.Sprintf(
			"第 %d 轮修复建议：围绕 change_id=%s 优先修复质量门禁失败项，补齐测试并再次执行 task run 与 quality。",
			iteration,
			changeID,
		)
	}
	return strings.TrimSpace(answer)
}

func resolveLifecyclePrompt(req StartRequest) string {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt != "" {
		return prompt
	}
	prompt = strings.TrimSpace(req.Options.Prompt)
	if prompt != "" {
		return prompt
	}
	return "Please execute the planned software delivery pipeline."
}

func extractChangeID(lines []string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`变更\s*ID[:：]\s*([^\s]+)`),
		regexp.MustCompile(`change[\s_-]*id[:：]\s*([^\s]+)`),
		regexp.MustCompile(`\.super-dev[\\/]+changes[\\/]+([^\\/\s]+)`),
	}
	for _, line := range lines {
		normalized := stripANSIEscapeCodes(strings.TrimSpace(line))
		if normalized == "" {
			continue
		}
		for _, pattern := range patterns {
			match := pattern.FindStringSubmatch(normalized)
			if len(match) == 2 {
				return strings.TrimSpace(match[1])
			}
		}
	}
	return ""
}

func resolveChangeIDFromLinesOrLatest(projectDir string, lines []string) string {
	changeID := extractChangeID(lines)
	if strings.TrimSpace(changeID) != "" {
		return strings.TrimSpace(changeID)
	}
	latestChangeID, err := findLatestChangeID(projectDir)
	if err == nil {
		return strings.TrimSpace(latestChangeID)
	}
	return ""
}

func extractPipelineProjectName(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "项目:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "项目:"))
		}
	}
	return ""
}

func stripANSIEscapeCodes(text string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	return re.ReplaceAllString(text, "")
}

func findLatestChangeID(projectDir string) (string, error) {
	changesDir := filepath.Join(resolveProjectDir(projectDir), ".super-dev", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return "", err
	}

	latestName := ""
	var latestModTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if latestName == "" || info.ModTime().After(latestModTime) {
			latestName = entry.Name()
			latestModTime = info.ModTime()
		}
	}
	if latestName == "" {
		return "", os.ErrNotExist
	}
	return latestName, nil
}

func (m *Manager) syncSuperDevProjectName(
	ctx context.Context,
	runID string,
	options RunRequest,
	projectName string,
) {
	trimmed := strings.TrimSpace(projectName)
	if trimmed == "" {
		return
	}
	lines, err := m.runner.RunCommand(ctx, options, []string{"config", "set", "name", trimmed})
	for _, line := range lines {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "lifecycle-config",
			Status:  "log",
			Message: line,
		})
	}
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "lifecycle-config",
			Status:  "log",
			Message: fmt.Sprintf("Sync project name to super-dev config failed: %v", err),
		})
		return
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "lifecycle-config",
		Status:  "completed",
		Message: "Synced super-dev config project name: " + trimmed,
	})
}

func (m *Manager) isQualityGatePassed(
	projectDir string,
	prompt string,
	backend string,
	qualityErr error,
	qualityLines []string,
) (bool, string) {
	outputDir := filepath.Join(resolveProjectDir(projectDir), "output")
	reportPath := filepath.Join(outputDir, buildChangeID(prompt)+"-quality-gate.md")
	resolvedReportPath := reportPath
	content, err := os.ReadFile(reportPath)
	if err != nil {
		latestPath, latestErr := findLatestQualityGateReport(outputDir)
		if latestErr == nil {
			latestContent, latestReadErr := os.ReadFile(latestPath)
			if latestReadErr == nil {
				content = latestContent
				err = nil
				resolvedReportPath = latestPath
			}
		}
	}
	if err != nil {
		if qualityErr != nil {
			return false, ""
		}
		joined := strings.Join(qualityLines, " ")
		joinedLower := strings.ToLower(joined)
		if strings.Contains(joined, "未通过") {
			return false, ""
		}
		if strings.Contains(joinedLower, "failed") || strings.Contains(joinedLower, "error") {
			return false, ""
		}
		return true, ""
	}
	report := string(content)
	if !strings.Contains(report, "未通过") {
		return true, ""
	}
	if allowed, reason := allowQualitySoftPass(report, backend, projectDir, resolvedReportPath); allowed {
		return true, reason
	}
	return false, ""
}

func findLatestQualityGateReport(outputDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(outputDir, "*-quality-gate.md"))
	if err != nil {
		return "", err
	}
	latestPath := ""
	var latestModTime time.Time
	for _, candidate := range matches {
		info, statErr := os.Stat(candidate)
		if statErr != nil || info.IsDir() {
			continue
		}
		if latestPath == "" || info.ModTime().After(latestModTime) {
			latestPath = candidate
			latestModTime = info.ModTime()
		}
	}
	if latestPath == "" {
		return "", os.ErrNotExist
	}
	return latestPath, nil
}

func allowQualitySoftPass(report string, backend string, projectDir string, reportPath string) (bool, string) {
	failedChecks := extractFailedCheckNames(report)
	if len(failedChecks) == 0 {
		return false, ""
	}

	criticalFailures := extractCriticalFailures(report)
	nonPythonBackend := !strings.EqualFold(strings.TrimSpace(backend), "python")

	toleratedPythonFailure := false
	toleratedSpecFailure := false
	tasksClosedChecked := false

	for _, check := range failedChecks {
		switch {
		case strings.Contains(check, "Python 语法检查"):
			if !nonPythonBackend {
				return false, ""
			}
			toleratedPythonFailure = true
		case strings.Contains(check, "Spec任务完成度"):
			if !tasksClosedChecked {
				closed, err := isCurrentChangeTasksClosed(projectDir, reportPath)
				if err != nil || !closed {
					return false, ""
				}
				tasksClosedChecked = true
			}
			toleratedSpecFailure = true
		default:
			return false, ""
		}
	}

	for _, failure := range criticalFailures {
		switch {
		case strings.Contains(failure, "Spec 任务闭环状态"):
			if !toleratedSpecFailure {
				return false, ""
			}
		case strings.Contains(failure, "Python 语法检查"), strings.Contains(strings.ToLower(failure), "compileall"):
			if !toleratedPythonFailure {
				return false, ""
			}
		default:
			return false, ""
		}
	}

	score := extractQualityScore(report)
	if score < 60 {
		return false, ""
	}

	reasons := make([]string, 0, 2)
	if toleratedSpecFailure {
		reasons = append(reasons, "Spec task closure is complete for current change; likely impacted by historical changes")
	}
	if toleratedPythonFailure {
		reasons = append(reasons, fmt.Sprintf("Python syntax check failed for non-python backend (%s)", strings.TrimSpace(backend)))
	}
	if len(reasons) == 0 {
		return false, ""
	}
	return true, fmt.Sprintf("Quality gate soft-pass: %s, score=%d", strings.Join(reasons, "; "), score)
}

func extractFailedCheckNames(report string) []string {
	failed := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(report, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || !strings.Contains(trimmed, "| ✗ |") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		if name == "" || name == "检查项" || strings.HasPrefix(name, ":---") {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		failed = append(failed, name)
	}
	return failed
}

func extractCriticalFailures(report string) []string {
	section := extractMarkdownSection(report, "## 关键失败项")
	if strings.TrimSpace(section) == "" {
		return nil
	}
	failures := make([]string, 0, 4)
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if item != "" {
			failures = append(failures, item)
		}
	}
	return failures
}

func isCurrentChangeTasksClosed(projectDir string, reportPath string) (bool, error) {
	changeID := resolveChangeIDFromQualityReportPath(reportPath)
	if changeID == "" {
		return false, errors.New("change id not resolved from report path")
	}

	taskFile := filepath.Join(resolveProjectDir(projectDir), ".super-dev", "changes", changeID, "tasks.md")
	content, err := os.ReadFile(taskFile)
	if err != nil {
		return false, err
	}

	checkPattern := regexp.MustCompile(`^\s*-\s*\[([ xX~_])\]`)
	total := 0
	completed := 0
	inProgress := 0
	for _, line := range strings.Split(string(content), "\n") {
		match := checkPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		total++
		switch strings.ToLower(match[1]) {
		case "x":
			completed++
		case "~":
			inProgress++
		}
	}
	if total == 0 {
		return false, errors.New("no tasks parsed from tasks.md")
	}

	return completed == total && inProgress == 0, nil
}

func resolveChangeIDFromQualityReportPath(reportPath string) string {
	base := strings.TrimSpace(filepath.Base(reportPath))
	if base == "" {
		return ""
	}
	if !strings.HasSuffix(base, "-quality-gate.md") {
		return ""
	}
	return strings.TrimSpace(strings.TrimSuffix(base, "-quality-gate.md"))
}

func extractMarkdownSection(markdown string, heading string) string {
	lines := strings.Split(markdown, "\n")
	start := -1
	for idx, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = idx + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	var section []string
	for idx := start; idx < len(lines); idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		if strings.HasPrefix(trimmed, "## ") {
			break
		}
		section = append(section, lines[idx])
	}
	return strings.Join(section, "\n")
}

func extractQualityScore(report string) int {
	re := regexp.MustCompile(`([0-9]+)/100`)
	for _, line := range strings.Split(report, "\n") {
		if !strings.Contains(line, "总分") {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		score, err := strconv.Atoi(match[1])
		if err == nil {
			return score
		}
	}
	return 0
}

func extractMetricCount(report string, key string) int {
	re := regexp.MustCompile(`- ` + regexp.QuoteMeta(key) + `:\s*([0-9]+)`)
	match := re.FindStringSubmatch(report)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func generateFallbackIterationGuidance(prompt string, iteration int) string {
	return fmt.Sprintf(
		"围绕需求「%s」执行第 %d 轮修复：优先补齐单元测试、修复质量门禁失败项、完善边界场景并回归验证。",
		strings.TrimSpace(prompt),
		iteration,
	)
}

func (m *Manager) generateIterationGuidance(ctx context.Context, req StartRequest, iteration int, qualitySummary string) string {
	if m.llmAdvisor == nil {
		return generateFallbackIterationGuidance(req.Prompt, iteration)
	}
	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是资深技术负责人。当前项目需求：%s\n当前是第 %d 轮开发-单测-修复迭代。最近质量信息：%s\n请输出不超过5条、可直接执行的修复动作清单。",
		req.Prompt,
		iteration,
		strings.TrimSpace(qualitySummary),
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return generateFallbackIterationGuidance(req.Prompt, iteration)
	}
	return strings.TrimSpace(answer)
}

func (m *Manager) generateAcceptanceSummary(ctx context.Context, req StartRequest, qualitySummary string) string {
	if m.llmAdvisor == nil {
		return "验收总结：质量门禁通过，建议执行上线前检查（部署配置、回滚方案、监控告警）。"
	}
	prompt := strings.TrimSpace(fmt.Sprintf(
		"请基于以下信息生成上线前验收总结（3-5条）：\n需求：%s\n质量结果：%s\n要求：覆盖功能验收、测试结论、发布与回滚准备。",
		req.Prompt,
		qualitySummary,
	))
	answer, err := m.llmAdvisor.Advise(ctx, prompt)
	if err != nil || strings.TrimSpace(answer) == "" {
		return "验收总结：质量门禁通过，建议执行上线前检查（部署配置、回滚方案、监控告警）。"
	}
	return strings.TrimSpace(answer)
}

func summarizeQualityOutput(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	return strings.Join(lines, " | ")
}

func resolveProjectDir(projectDir string) string {
	trimmed := strings.TrimSpace(projectDir)
	if trimmed == "" {
		trimmed = "."
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return abs
}

func buildChangeID(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return "pipeline-run"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "pipeline-run"
	}
	return result
}

func (m *Manager) runSimulation(ctx context.Context, runID string, req StartRequest, phasePacks []PhaseContextPack) {
	phaseContextMap := map[string]PhaseContextPack{}
	for _, item := range phasePacks {
		phaseContextMap[item.Stage] = item
	}

	total := len(m.phases)
	for idx, stage := range m.phases {
		progress := int(float64(idx) / float64(total) * 100)
		if phaseCtx, ok := phaseContextMap[stage]; ok {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:  runID,
				Stage:  stage,
				Status: "log",
				Message: fmt.Sprintf(
					"Phase context loaded (memories=%d, knowledge=%d)",
					len(phaseCtx.Pack.Memories),
					len(phaseCtx.Pack.Knowledge),
				),
			})
		}
		_ = m.store.UpdatePipelineRun(ctx, runID, "running", stage, progress, nil, nil)
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   stage,
			Status:  "running",
			Message: fmt.Sprintf("%s started", stage),
		})

		time.Sleep(m.phaseDelay)

		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   stage,
			Status:  "completed",
			Message: fmt.Sprintf("%s completed", stage),
		})
	}

	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "done",
		Status:  "completed",
		Message: "Pipeline finished (simulated)",
	})
}

func (m *Manager) runWithSuperDev(ctx context.Context, runID string, req StartRequest, phasePacks []PhaseContextPack) {
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", "super-dev", 10, nil, nil)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "super-dev",
		Status:  "running",
		Message: "Executing super-dev pipeline command",
	})

	lines, err := m.runner.RunPipeline(ctx, req.Options)
	for _, line := range lines {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "super-dev",
			Status:  "log",
			Message: line,
		})
	}

	if err != nil {
		m.writebackRunMemory(ctx, req, runID, "failed", "super-dev", err.Error(), phasePacks)
		finished := time.Now().UTC()
		_ = m.store.UpdatePipelineRun(ctx, runID, "failed", "super-dev", 100, nil, &finished)
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "super-dev",
			Status:  "failed",
			Message: err.Error(),
		})
		return
	}

	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "done",
		Status:  "completed",
		Message: "Pipeline finished",
	})
}

func (m *Manager) writebackRunMemory(
	ctx context.Context,
	req StartRequest,
	runID string,
	status string,
	stage string,
	errorMessage string,
	phasePacks []PhaseContextPack,
) {
	if !req.Context.MemoryWriteback {
		return
	}

	tags := []string{"pipeline", "run", status}
	if req.Context.Mode != "" {
		tags = append(tags, "context-"+string(req.Context.Mode))
	}
	if req.Context.DynamicByPhase {
		tags = append(tags, "dynamic-phase-context")
	}

	content := []string{
		fmt.Sprintf("run_id: %s", runID),
		fmt.Sprintf("status: %s", status),
		fmt.Sprintf("stage: %s", stage),
		fmt.Sprintf("prompt: %s", strings.TrimSpace(req.Prompt)),
		fmt.Sprintf("phase_contexts: %d", len(phasePacks)),
	}
	if strings.TrimSpace(errorMessage) != "" {
		content = append(content, "error: "+strings.TrimSpace(errorMessage))
	}

	_, _ = m.store.CreateMemory(ctx, store.Memory{
		ProjectID:  req.ProjectID,
		Role:       "run-summary",
		Content:    strings.Join(content, "\n"),
		Tags:       tags,
		Importance: 0.85,
	})

	m.writebackRunKnowledge(ctx, req, runID, status, stage, errorMessage)
}

func (m *Manager) writebackRunKnowledge(
	ctx context.Context,
	req StartRequest,
	runID string,
	status string,
	stage string,
	errorMessage string,
) {
	existingDocs, err := m.store.ListKnowledgeDocuments(ctx, req.ProjectID)
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "knowledge-writeback",
			Status:  "log",
			Message: fmt.Sprintf("Knowledge writeback skipped: list docs failed: %v", err),
		})
		return
	}
	sourceSet := make(map[string]struct{}, len(existingDocs))
	for _, doc := range existingDocs {
		key := strings.TrimSpace(doc.Source)
		if key != "" {
			sourceSet[key] = struct{}{}
		}
	}

	added := 0
	addDoc := func(title, source, body string, chunkSize int) {
		title = strings.TrimSpace(title)
		source = strings.TrimSpace(source)
		body = strings.TrimSpace(body)
		if title == "" || source == "" || body == "" {
			return
		}
		if _, exists := sourceSet[source]; exists {
			return
		}
		if _, _, addErr := m.store.AddKnowledgeDocument(ctx, req.ProjectID, title, source, body, chunkSize); addErr != nil {
			_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
				RunID:   runID,
				Stage:   "knowledge-writeback",
				Status:  "log",
				Message: fmt.Sprintf("Knowledge writeback add doc failed (%s): %v", title, addErr),
			})
			return
		}
		sourceSet[source] = struct{}{}
		added++
	}

	events, eventErr := m.store.ListRunEvents(ctx, runID)
	if eventErr == nil {
		if planContent := buildRunPlanKnowledgeContent(runID, status, stage, errorMessage, events); strings.TrimSpace(planContent) != "" {
			addDoc(
				fmt.Sprintf("运行方案沉淀-%s", runID),
				fmt.Sprintf("volcengine-plan:%s", runID),
				planContent,
				600,
			)
		}
	}

	runInfo, runErr := m.store.GetPipelineRun(ctx, runID)
	if runErr == nil {
		projectDir := strings.TrimSpace(req.Options.ProjectDir)
		if projectDir == "" {
			projectDir = strings.TrimSpace(runInfo.ProjectDir)
		}
		markdownFiles := collectRunOutputMarkdownFiles(projectDir, runInfo)
		for _, path := range markdownFiles {
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				continue
			}
			rel := path
			if absBase, absErr := filepath.Abs(resolveProjectDir(projectDir)); absErr == nil {
				if relPath, relErr := filepath.Rel(absBase, path); relErr == nil {
					rel = filepath.ToSlash(relPath)
				}
			}
			addDoc(
				fmt.Sprintf("super-dev项目文档/%s", filepath.Base(path)),
				fmt.Sprintf("super-dev-output:%s:%s", runID, rel),
				string(raw),
				800,
			)
		}
	}

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "knowledge-writeback",
		Status:  "completed",
		Message: fmt.Sprintf("Knowledge writeback finished (added=%d)", added),
	})
}

func buildRunPlanKnowledgeContent(
	runID string,
	status string,
	stage string,
	errorMessage string,
	events []store.RunEvent,
) string {
	if len(events) == 0 && strings.TrimSpace(errorMessage) == "" {
		return ""
	}
	lines := []string{
		fmt.Sprintf("run_id: %s", runID),
		fmt.Sprintf("final_status: %s", strings.TrimSpace(status)),
		fmt.Sprintf("final_stage: %s", strings.TrimSpace(stage)),
		"",
		"## 方案与推进记录",
	}
	seen := map[string]struct{}{}
	addMessage := func(msg string) {
		normalized := strings.TrimSpace(msg)
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		lines = append(lines, "- "+normalized)
	}

	for _, event := range events {
		trimmed := strings.TrimSpace(event.Message)
		if trimmed == "" {
			continue
		}
		if event.Stage == "step-agent" ||
			event.Stage == "lifecycle-acceptance" ||
			event.Stage == "step-acceptance" ||
			strings.HasPrefix(trimmed, "LLM iteration guidance:") {
			addMessage(fmt.Sprintf("[%s] %s", event.Stage, trimmed))
		}
	}
	if strings.TrimSpace(errorMessage) != "" {
		lines = append(lines, "", "## 错误信息", strings.TrimSpace(errorMessage))
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if utf8.RuneCountInString(joined) > 12000 {
		runes := []rune(joined)
		joined = strings.TrimSpace(string(runes[:12000]))
	}
	return joined
}

func collectRunOutputMarkdownFiles(projectDir string, run store.PipelineRun) []string {
	baseDir := resolveProjectDir(projectDir)
	outputDir := filepath.Join(baseDir, "output")
	info, err := os.Stat(outputDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	start := run.CreatedAt
	if run.StartedAt != nil && !run.StartedAt.IsZero() {
		start = *run.StartedAt
	}
	end := run.UpdatedAt
	if run.FinishedAt != nil && !run.FinishedAt.IsZero() {
		end = *run.FinishedAt
	}
	lowerBound := start.Add(-2 * time.Minute)
	upperBound := end.Add(2 * time.Minute)

	type candidate struct {
		path    string
		modTime time.Time
	}
	candidates := make([]candidate, 0, 24)
	_ = filepath.Walk(outputDir, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil || fileInfo == nil || fileInfo.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		mod := fileInfo.ModTime().UTC()
		if mod.Before(lowerBound) || mod.After(upperBound) {
			return nil
		}
		candidates = append(candidates, candidate{path: path, modTime: mod})
		return nil
	})
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].modTime.Before(candidates[j].modTime)
	})
	if len(candidates) > 40 {
		candidates = candidates[:40]
	}
	paths := make([]string, 0, len(candidates))
	for _, item := range candidates {
		paths = append(paths, item.path)
	}
	return paths
}
