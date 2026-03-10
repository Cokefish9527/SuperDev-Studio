package pipeline

import (
	"context"
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

	"superdevstudio/internal/store"
)

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

	m.maybeGenerateReflectionArtifact(ctx, runID, req, "", "", "completed", "")
	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	m.touchChangeBatch(ctx, req.ChangeBatchID, "completed", runID, "")
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
		m.maybeGenerateReflectionArtifact(ctx, runID, req, "", "", "failed", err.Error())
		m.writebackRunMemory(ctx, req, runID, "failed", "super-dev", err.Error(), phasePacks)
		finished := time.Now().UTC()
		_ = m.store.UpdatePipelineRun(ctx, runID, "failed", "super-dev", 100, nil, &finished)
		m.touchChangeBatch(ctx, req.ChangeBatchID, "failed", runID, "")
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "super-dev",
			Status:  "failed",
			Message: err.Error(),
		})
		return
	}

	changeID := resolveChangeIDFromLinesOrLatest(req.Options.ProjectDir, lines)
	docsBrief := buildCreateDocsBrief(req.Options.ProjectDir, lines)
	m.maybeGenerateDesignArtifact(ctx, runID, req, changeID, docsBrief)
	m.maybeGenerateReflectionArtifact(ctx, runID, req, changeID, "", "completed", "")
	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", phasePacks)
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	m.touchChangeBatch(ctx, req.ChangeBatchID, "completed", runID, "")
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

func (m *Manager) touchChangeBatch(ctx context.Context, changeBatchID, status, latestRunID, externalChangeID string) {
	trimmedChangeBatchID := strings.TrimSpace(changeBatchID)
	if trimmedChangeBatchID == "" {
		return
	}
	_, _ = m.store.UpdateChangeBatch(ctx, trimmedChangeBatchID, status, latestRunID, externalChangeID)
	_ = m.store.SyncRequirementSessionsLatestRunByChangeBatch(ctx, trimmedChangeBatchID, latestRunID)
}

func (m *Manager) bindExternalChangeID(ctx context.Context, runID, changeBatchID, externalChangeID string) {
	trimmed := strings.TrimSpace(externalChangeID)
	if trimmed == "" {
		return
	}
	_ = m.store.SetPipelineRunExternalChangeID(ctx, runID, trimmed)
	m.touchChangeBatch(ctx, changeBatchID, "running", runID, trimmed)
}
