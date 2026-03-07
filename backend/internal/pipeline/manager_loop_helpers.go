package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"superdevstudio/internal/store"
)

type loopTemplateKind string

const (
	loopTemplateConcept    loopTemplateKind = "concept"
	loopTemplateDesign     loopTemplateKind = "design"
	loopTemplateReflection loopTemplateKind = "reflection"
)

type loopArtifactTemplate struct {
	stage    string
	title    string
	suffix   string
	prompt   string
	assets   []string
	sections []string
	kind     loopTemplateKind
	changeID string
}

type structuredLoopAnswer struct {
	Summary                            string
	UserValue                          []string
	Scenarios                          []string
	InformationArchitecture            []string
	PageFlows                          []string
	DesignConclusions                  []string
	InformationArchitectureAdjustments []string
	DataModelChanges                   []string
	PageRedesignPlan                   []string
	SuperdevActions                    []string
	DeliveredOutcomes                  []string
	QualityReview                      []string
	GapsAndRisks                       []string
	NextConcepts                       []string
	RetrospectiveNotes                 []string
	Risks                              []string
	AcceptanceCheckpoints              []string
	OpenQuestions                      []string
	NextActions                        []string
}

func (m *Manager) adviseWithOptionalAssets(ctx context.Context, prompt string, assetURLs []string) (string, error) {
	if m.llmAdvisor == nil {
		return "", errors.New("llm advisor is not configured")
	}
	assets := sanitizeAssetURLs(assetURLs)
	if len(assets) > 0 {
		if advisor, ok := m.llmAdvisor.(AssetAwareAdvisor); ok {
			return advisor.AdviseWithAssets(ctx, prompt, assets)
		}
	}
	return m.llmAdvisor.Advise(ctx, prompt)
}

func sanitizeAssetURLs(items []string) []string {
	cleaned := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

func (m *Manager) maybeGenerateConceptArtifact(ctx context.Context, runID string, req StartRequest) {
	if !req.LLM.EnhancedLoop {
		return
	}
	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是 SuperDev Studio 的构思引擎。请基于以下需求与参考素材，输出严格模板可用的构思稿 JSON。"+
			"\n需求：%s\n"+
			"\n要求：仅输出 JSON 对象，不要 Markdown、不要代码块。字段如下："+
			"\nsummary:string"+
			"\nuser_value:string[]"+
			"\nscenarios:string[]"+
			"\ninformation_architecture:string[]"+
			"\npage_flows:string[]"+
			"\nrisks:string[]"+
			"\nacceptance_checkpoints:string[]"+
			"\nnext_actions:string[]",
		resolveLifecyclePrompt(req),
	))
	m.maybeGenerateLoopArtifact(ctx, runID, req, loopArtifactTemplate{
		stage:    "llm-idea",
		title:    "构思增强稿",
		suffix:   "concept",
		prompt:   prompt,
		assets:   req.LLM.MultimodalAssets,
		kind:     loopTemplateConcept,
		changeID: effectiveLoopChangeID("", req),
		sections: []string{
			"## 输入需求\n" + strings.TrimSpace(resolveLifecyclePrompt(req)),
		},
	})
}

func (m *Manager) maybeGenerateDesignArtifact(ctx context.Context, runID string, req StartRequest, changeID string, docsBrief string) {
	if !req.LLM.EnhancedLoop {
		return
	}
	designSummary := strings.TrimSpace(docsBrief)
	if designSummary == "" {
		designSummary = m.buildLoopOutputSummary(ctx, runID, req, 2400)
	}
	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是 SuperDev Studio 的设计复核引擎。请基于需求、初始设计文档与视觉参考，输出严格模板可用的设计复核 JSON。"+
			"\n需求：%s\nchange_id：%s\n初始设计摘要：\n%s\n"+
			"\n要求：仅输出 JSON 对象，不要 Markdown、不要代码块。字段如下："+
			"\nsummary:string"+
			"\ndesign_conclusions:string[]"+
			"\ninformation_architecture_adjustments:string[]"+
			"\ndata_model_changes:string[]"+
			"\npage_redesign_plan:string[]"+
			"\nsuperdev_actions:string[]"+
			"\nrisks:string[]"+
			"\nacceptance_checkpoints:string[]"+
			"\nopen_questions:string[]"+
			"\nnext_actions:string[]",
		resolveLifecyclePrompt(req),
		effectiveLoopChangeID(changeID, req),
		truncateForPrompt(designSummary, 2800),
	))
	sections := []string{
		"## 输入需求\n" + strings.TrimSpace(resolveLifecyclePrompt(req)),
	}
	if strings.TrimSpace(designSummary) != "" {
		sections = append(sections, "## 初始设计摘要\n"+truncateForPrompt(designSummary, 2400))
	}
	m.maybeGenerateLoopArtifact(ctx, runID, req, loopArtifactTemplate{
		stage:    "llm-design",
		title:    "设计复核稿",
		suffix:   "design-loop",
		prompt:   prompt,
		assets:   req.LLM.MultimodalAssets,
		sections: sections,
		kind:     loopTemplateDesign,
		changeID: effectiveLoopChangeID(changeID, req),
	})
}

func (m *Manager) maybeGenerateReflectionArtifact(
	ctx context.Context,
	runID string,
	req StartRequest,
	changeID string,
	qualitySummary string,
	finalStatus string,
	errorMessage string,
) {
	if !req.LLM.EnhancedLoop {
		return
	}
	eventSummary := m.buildLoopEventSummary(ctx, runID, 8)
	outputSummary := m.buildLoopOutputSummary(ctx, runID, req, 2600)
	prompt := strings.TrimSpace(fmt.Sprintf(
		"你是 SuperDev Studio 的复盘引擎。请基于本轮执行结果生成严格模板可用的复盘 JSON，并给出下一轮构思建议。"+
			"\n需求：%s\nchange_id：%s\n最终状态：%s\n质量摘要：%s\n错误信息：%s\n最近推进记录：\n%s\n当前产物摘要：\n%s\n"+
			"\n要求：仅输出 JSON 对象，不要 Markdown、不要代码块。字段如下："+
			"\nsummary:string"+
			"\ndelivered_outcomes:string[]"+
			"\nquality_review:string[]"+
			"\ngaps_and_risks:string[]"+
			"\nnext_concepts:string[]"+
			"\nretrospective_notes:string[]"+
			"\nacceptance_checkpoints:string[]"+
			"\nnext_actions:string[]",
		resolveLifecyclePrompt(req),
		effectiveLoopChangeID(changeID, req),
		strings.TrimSpace(finalStatus),
		truncateForPrompt(strings.TrimSpace(qualitySummary), 900),
		truncateForPrompt(strings.TrimSpace(errorMessage), 600),
		truncateForPrompt(eventSummary, 1800),
		truncateForPrompt(outputSummary, 2400),
	))
	sections := []string{
		"## 执行状态\n" + strings.TrimSpace(finalStatus),
	}
	if strings.TrimSpace(qualitySummary) != "" {
		sections = append(sections, "## 质量摘要\n"+truncateForPrompt(qualitySummary, 900))
	}
	if strings.TrimSpace(eventSummary) != "" {
		sections = append(sections, "## 最近推进记录\n"+truncateForPrompt(eventSummary, 1800))
	}
	if strings.TrimSpace(outputSummary) != "" {
		sections = append(sections, "## 当前产物摘要\n"+truncateForPrompt(outputSummary, 2200))
	}
	if strings.TrimSpace(errorMessage) != "" {
		sections = append(sections, "## 错误信息\n"+truncateForPrompt(errorMessage, 600))
	}
	m.maybeGenerateLoopArtifact(ctx, runID, req, loopArtifactTemplate{
		stage:    "llm-rethink",
		title:    "复盘再构思稿",
		suffix:   "reflection",
		prompt:   prompt,
		assets:   req.LLM.MultimodalAssets,
		sections: sections,
		kind:     loopTemplateReflection,
		changeID: effectiveLoopChangeID(changeID, req),
	})
}

func (m *Manager) maybeGenerateLoopArtifact(ctx context.Context, runID string, req StartRequest, spec loopArtifactTemplate) {
	if !req.LLM.EnhancedLoop {
		return
	}
	if strings.TrimSpace(spec.prompt) == "" {
		return
	}
	if m.llmAdvisor == nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   spec.stage,
			Status:  "log",
			Message: spec.title + " skipped: volcengine advisor is not configured",
		})
		return
	}

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   spec.stage,
		Status:  "running",
		Message: "Generating " + spec.title,
	})

	answer, err := m.adviseWithOptionalAssets(ctx, spec.prompt, spec.assets)
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   spec.stage,
			Status:  "failed",
			Message: fmt.Sprintf("Generate %s failed: %v", spec.title, err),
		})
		return
	}

	outputDir := filepath.Join(resolveProjectDir(req.Options.ProjectDir), "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   spec.stage,
			Status:  "failed",
			Message: fmt.Sprintf("Prepare output dir for %s failed: %v", spec.title, err),
		})
		return
	}

	baseName := effectiveLoopChangeID(spec.changeID, req)
	if runInfo, getErr := m.store.GetPipelineRun(ctx, runID); getErr == nil {
		baseName = effectiveLoopChangeID(firstNonEmpty(spec.changeID, runInfo.ExternalChangeID), req)
	}
	filePath := filepath.Join(outputDir, fmt.Sprintf("%s-%s.md", baseName, spec.suffix))
	spec.changeID = baseName
	content := buildLoopArtifactMarkdown(runID, spec, strings.TrimSpace(answer))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   spec.stage,
			Status:  "failed",
			Message: fmt.Sprintf("Write %s failed: %v", spec.title, err),
		})
		return
	}

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   spec.stage,
		Status:  "completed",
		Message: fmt.Sprintf("Generated %s: %s", spec.title, filepath.Base(filePath)),
	})
}

func buildLoopArtifactMarkdown(runID string, spec loopArtifactTemplate, answer string) string {
	structured := parseStructuredLoopAnswer(answer)
	assets := sanitizeAssetURLs(spec.assets)
	lines := []string{
		fmt.Sprintf("# %s", spec.title),
		"",
		"## 文档元数据",
		"| 字段 | 值 |",
		"| --- | --- |",
		fmt.Sprintf("| run_id | %s |", strings.TrimSpace(runID)),
		fmt.Sprintf("| stage | %s |", strings.TrimSpace(spec.stage)),
		fmt.Sprintf("| template_kind | %s |", strings.TrimSpace(string(spec.kind))),
		fmt.Sprintf("| change_id | %s |", strings.TrimSpace(spec.changeID)),
		fmt.Sprintf("| generated_at | %s |", time.Now().UTC().Format(time.RFC3339)),
		fmt.Sprintf("| multimodal_assets | %d |", len(assets)),
		"",
		"## 输入快照",
	}
	inputSections := normalizeTemplateInputSections(spec.sections)
	if len(inputSections) == 0 {
		lines = append(lines, "### 输入摘要", "待补充。")
	} else {
		lines = append(lines, inputSections...)
	}
	if len(assets) > 0 {
		lines = append(lines, "", "### 参考素材")
		for _, asset := range assets {
			lines = append(lines, "- "+asset)
		}
	}
	lines = append(lines, "", "## 执行摘要", nonEmptyOrDefault(structured.Summary, summarizeLoopRawAnswer(answer)))
	lines = append(lines, renderLoopArtifactBody(spec.kind, structured)...)
	lines = append(lines, renderBulletSection("## 风险与依赖", collectLoopRisks(spec.kind, structured))...)
	lines = append(lines, renderBulletSection("## 验收检查点", collectLoopAcceptance(spec.kind, structured))...)
	lines = append(lines, renderBulletSection("## 下一步动作", collectLoopNextActions(spec.kind, structured))...)
	if openQuestions := collectLoopOpenQuestions(spec.kind, structured); len(openQuestions) > 0 {
		lines = append(lines, renderBulletSection("## 待确认问题", openQuestions)...)
	}
	lines = append(lines, "", "## LLM 原始输出", "```text", rawLoopAnswer(answer), "```")
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func parseStructuredLoopAnswer(answer string) structuredLoopAnswer {
	payload := extractLoopJSONObject(answer)
	if payload == "" {
		return structuredLoopAnswer{}
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return structuredLoopAnswer{}
	}
	return structuredLoopAnswer{
		Summary:                            extractLoopString(raw, "summary"),
		UserValue:                          extractLoopList(raw, "user_value"),
		Scenarios:                          extractLoopList(raw, "scenarios"),
		InformationArchitecture:            extractLoopList(raw, "information_architecture"),
		PageFlows:                          extractLoopList(raw, "page_flows"),
		DesignConclusions:                  extractLoopList(raw, "design_conclusions"),
		InformationArchitectureAdjustments: extractLoopList(raw, "information_architecture_adjustments"),
		DataModelChanges:                   extractLoopList(raw, "data_model_changes"),
		PageRedesignPlan:                   extractLoopList(raw, "page_redesign_plan"),
		SuperdevActions:                    extractLoopList(raw, "superdev_actions"),
		DeliveredOutcomes:                  extractLoopList(raw, "delivered_outcomes"),
		QualityReview:                      extractLoopList(raw, "quality_review"),
		GapsAndRisks:                       extractLoopList(raw, "gaps_and_risks"),
		NextConcepts:                       extractLoopList(raw, "next_concepts"),
		RetrospectiveNotes:                 extractLoopList(raw, "retrospective_notes"),
		Risks:                              extractLoopList(raw, "risks"),
		AcceptanceCheckpoints:              extractLoopList(raw, "acceptance_checkpoints"),
		OpenQuestions:                      extractLoopList(raw, "open_questions"),
		NextActions:                        extractLoopList(raw, "next_actions"),
	}
}

func extractLoopJSONObject(answer string) string {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(trimmed[start : end+1])
}

func extractLoopString(raw map[string]any, key string) string {
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.Join(parts, "；")
	default:
		return ""
	}
}

func extractLoopList(raw map[string]any, key string) []string {
	value, ok := raw[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string:
		return splitLoopStringList(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			switch element := item.(type) {
			case string:
				items = append(items, strings.TrimSpace(element))
			default:
				items = append(items, strings.TrimSpace(fmt.Sprint(element)))
			}
		}
		return compactLoopItems(items)
	default:
		return nil
	}
}

func splitLoopStringList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	replaced := strings.NewReplacer("\r\n", "\n", "；", "\n", ";", "\n", "•", "\n", "- ", "\n").Replace(trimmed)
	parts := strings.Split(replaced, "\n")
	return compactLoopItems(parts)
}

func compactLoopItems(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		trimmed := strings.TrimSpace(strings.TrimPrefix(raw, "-"))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeTemplateInputSections(sections []string) []string {
	result := make([]string, 0, len(sections)*2)
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")
		if strings.HasPrefix(normalized, "## ") {
			normalized = "### " + strings.TrimPrefix(normalized, "## ")
		}
		result = append(result, normalized)
	}
	return result
}

func renderLoopArtifactBody(kind loopTemplateKind, answer structuredLoopAnswer) []string {
	switch kind {
	case loopTemplateConcept:
		return concatLoopSections(
			renderBulletSection("## 用户价值", answer.UserValue),
			renderBulletSection("## 核心场景", answer.Scenarios),
			renderBulletSection("## 信息架构草案", answer.InformationArchitecture),
			renderBulletSection("## 关键页面与流程", answer.PageFlows),
		)
	case loopTemplateDesign:
		return concatLoopSections(
			renderBulletSection("## 设计结论", answer.DesignConclusions),
			renderBulletSection("## 信息架构调整", answer.InformationArchitectureAdjustments),
			renderBulletSection("## 数据模型调整", answer.DataModelChanges),
			renderBulletSection("## 页面改版草图", answer.PageRedesignPlan),
			renderBulletSection("## super-dev 执行动作", answer.SuperdevActions),
		)
	case loopTemplateReflection:
		return concatLoopSections(
			renderBulletSection("## 本轮产出", answer.DeliveredOutcomes),
			renderBulletSection("## 质量复盘", answer.QualityReview),
			renderBulletSection("## 缺口与债务", answer.GapsAndRisks),
			renderBulletSection("## 下一轮构思", answer.NextConcepts),
			renderBulletSection("## 复盘备注", answer.RetrospectiveNotes),
		)
	default:
		return renderBulletSection("## 模板正文", nil)
	}
}

func concatLoopSections(parts ...[]string) []string {
	result := make([]string, 0, 32)
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		if len(result) > 0 {
			result = append(result, "")
		}
		result = append(result, part...)
	}
	return result
}

func renderBulletSection(title string, items []string) []string {
	lines := []string{title}
	items = compactLoopItems(items)
	if len(items) == 0 {
		return append(lines, "- 待 LLM/人工补充")
	}
	for _, item := range items {
		lines = append(lines, "- "+item)
	}
	return lines
}

func collectLoopRisks(kind loopTemplateKind, answer structuredLoopAnswer) []string {
	switch kind {
	case loopTemplateReflection:
		return compactLoopItems(append(answer.Risks, answer.GapsAndRisks...))
	default:
		return compactLoopItems(answer.Risks)
	}
}

func collectLoopAcceptance(kind loopTemplateKind, answer structuredLoopAnswer) []string {
	items := compactLoopItems(answer.AcceptanceCheckpoints)
	if len(items) > 0 {
		return items
	}
	switch kind {
	case loopTemplateConcept:
		return []string{"构思阶段的用户价值、场景和页面流程均已明确。", "已输出可直接进入设计阶段的下一步动作。"}
	case loopTemplateDesign:
		return []string{"设计复核已覆盖信息架构、数据模型和页面改版。", "已给出可直接交给 super-dev 执行的动作列表。"}
	case loopTemplateReflection:
		return []string{"复盘已说明本轮产出、缺口与下一轮方向。", "已形成可进入下一次构思的行动项。"}
	default:
		return []string{"待补充验收检查点。"}
	}
}

func collectLoopNextActions(kind loopTemplateKind, answer structuredLoopAnswer) []string {
	items := compactLoopItems(answer.NextActions)
	if len(items) > 0 {
		return items
	}
	switch kind {
	case loopTemplateConcept:
		return []string{"基于构思稿进入设计复核阶段。", "将关键场景与页面流程转化为 UI/UX 与架构约束。"}
	case loopTemplateDesign:
		return []string{"将设计复核结论转化为 super-dev 任务。", "按验收点检查数据模型和页面改版是否落地。"}
	case loopTemplateReflection:
		return []string{"基于复盘结果整理下一轮需求输入。", "把缺口与风险写回后续迭代 backlog。"}
	default:
		return []string{"待补充后续动作。"}
	}
}

func collectLoopOpenQuestions(kind loopTemplateKind, answer structuredLoopAnswer) []string {
	if kind != loopTemplateDesign {
		return nil
	}
	return compactLoopItems(answer.OpenQuestions)
}

func summarizeLoopRawAnswer(answer string) string {
	trimmed := strings.TrimSpace(rawLoopAnswer(answer))
	if trimmed == "" {
		return "LLM 未返回结构化摘要，待人工补充。"
	}
	return truncateForPrompt(trimmed, 260)
}

func rawLoopAnswer(answer string) string {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return "LLM 未返回内容。"
	}
	return trimmed
}

func nonEmptyOrDefault(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	trimmedFallback := strings.TrimSpace(fallback)
	if trimmedFallback != "" {
		return trimmedFallback
	}
	return "待补充。"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func effectiveLoopChangeID(changeID string, req StartRequest) string {
	trimmed := strings.TrimSpace(changeID)
	if trimmed != "" {
		return trimmed
	}
	return buildChangeID(resolveLifecyclePrompt(req))
}

func (m *Manager) buildLoopEventSummary(ctx context.Context, runID string, limit int) string {
	events, err := m.store.ListRunEvents(ctx, runID)
	if err != nil || len(events) == 0 {
		return ""
	}
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}
	selected := events[len(events)-limit:]
	lines := make([]string, 0, len(selected))
	for _, event := range selected {
		message := strings.TrimSpace(event.Message)
		if message == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- [%s/%s] %s", event.Stage, event.Status, message))
	}
	return strings.Join(lines, "\n")
}

func (m *Manager) buildLoopOutputSummary(ctx context.Context, runID string, req StartRequest, maxRunes int) string {
	runInfo, err := m.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return ""
	}
	projectDir := strings.TrimSpace(req.Options.ProjectDir)
	if projectDir == "" {
		projectDir = strings.TrimSpace(runInfo.ProjectDir)
	}
	markdownFiles := collectRunOutputMarkdownFiles(projectDir, runInfo)
	if len(markdownFiles) == 0 {
		return ""
	}
	sections := make([]string, 0, minInt(len(markdownFiles), 6))
	for _, path := range markdownFiles {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		snippet := truncateForPrompt(string(raw), 420)
		if strings.TrimSpace(snippet) == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("[%s]\n%s", filepath.Base(path), snippet))
		if len([]rune(strings.Join(sections, "\n\n"))) >= maxRunes {
			break
		}
	}
	return truncateForPrompt(strings.Join(sections, "\n\n"), maxRunes)
}
