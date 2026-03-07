package api

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"superdevstudio/internal/store"
)

type scannedOutputFile struct {
	fullPath       string
	baseRelative   string
	outputRelative string
	sizeBytes      int64
	updatedAt      time.Time
}

var pipelineStageCatalog = []struct {
	Key   string
	Title string
}{
	{Key: "idea", Title: "构思"},
	{Key: "design", Title: "设计"},
	{Key: "superdev", Title: "super-dev"},
	{Key: "output", Title: "产出"},
	{Key: "rethink", Title: "再构思"},
}

func collectPipelineOutputFiles(run store.PipelineRun) []scannedOutputFile {
	baseDir := resolveRunBaseDir(run)
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
	lowerBound := start.Add(-5 * time.Minute)
	upperBound := end.Add(10 * time.Minute)
	externalChangeID := strings.ToLower(strings.TrimSpace(run.ExternalChangeID))
	promptChangeID := strings.ToLower(changeIDFromPrompt(run.Prompt))

	matches := make([]scannedOutputFile, 0, 24)
	fallback := make([]scannedOutputFile, 0, 24)
	_ = filepath.Walk(outputDir, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil || fileInfo == nil || fileInfo.IsDir() {
			return nil
		}
		outputRel, relErr := filepath.Rel(outputDir, path)
		if relErr != nil {
			return nil
		}
		baseRel, baseErr := filepath.Rel(baseDir, path)
		if baseErr != nil {
			baseRel = path
		}
		item := scannedOutputFile{
			fullPath:       path,
			baseRelative:   filepath.ToSlash(baseRel),
			outputRelative: filepath.ToSlash(outputRel),
			sizeBytes:      fileInfo.Size(),
			updatedAt:      fileInfo.ModTime().UTC(),
		}
		fallback = append(fallback, item)
		withinWindow := !item.updatedAt.Before(lowerBound) && !item.updatedAt.After(upperBound)
		if withinWindow || outputFileMatchesRun(item.outputRelative, externalChangeID, promptChangeID) {
			matches = append(matches, item)
		}
		return nil
	})
	if len(matches) == 0 {
		matches = fallback
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].updatedAt.Equal(matches[j].updatedAt) {
			return matches[i].outputRelative < matches[j].outputRelative
		}
		return matches[i].updatedAt.Before(matches[j].updatedAt)
	})
	if len(matches) > 80 {
		matches = matches[len(matches)-80:]
	}
	return matches
}

func outputFileMatchesRun(outputRelative, externalChangeID, promptChangeID string) bool {
	lower := strings.ToLower(filepath.ToSlash(outputRelative))
	if lower == "preview.html" || strings.HasPrefix(lower, "frontend/") {
		return true
	}
	if externalChangeID != "" && strings.Contains(lower, externalChangeID) {
		return true
	}
	if promptChangeID != "" && strings.Contains(lower, promptChangeID) {
		return true
	}
	importantTokens := []string{
		"concept",
		"design-loop",
		"reflection",
		"prd",
		"architecture",
		"uiux",
		"frontend-blueprint",
		"execution-plan",
		"quality-gate",
		"redteam",
		"task-execution",
	}
	for _, token := range importantTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func buildScannedArtifact(run store.PipelineRun, item scannedOutputFile) pipelineArtifact {
	previewType := inferArtifactPreviewType(item.outputRelative)
	return pipelineArtifact{
		Name:        prettyArtifactName(item.outputRelative),
		Path:        item.baseRelative,
		Kind:        inferArtifactKind(item.outputRelative, previewType),
		SizeBytes:   item.sizeBytes,
		UpdatedAt:   item.updatedAt.Format(time.RFC3339),
		PreviewURL:  fmt.Sprintf("/api/pipeline/runs/%s/preview/%s", run.ID, item.outputRelative),
		PreviewType: previewType,
		Stage:       inferArtifactStage(item.outputRelative),
	}
}

func inferArtifactKind(outputRelative, previewType string) string {
	lower := strings.ToLower(filepath.ToSlash(outputRelative))
	if strings.HasPrefix(lower, "frontend/") {
		return "frontend"
	}
	if previewType != "" {
		return previewType
	}
	return "file"
}

func inferArtifactPreviewType(outputRelative string) string {
	lower := strings.ToLower(filepath.ToSlash(outputRelative))
	ext := strings.ToLower(filepath.Ext(lower))
	switch ext {
	case ".md":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".txt", ".log", ".json", ".yaml", ".yml", ".css", ".js", ".ts", ".tsx", ".xml":
		return "text"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "image"
	default:
		if strings.HasPrefix(lower, "frontend/") {
			return "text"
		}
		return "binary"
	}
}

func inferArtifactStage(outputRelative string) string {
	lower := strings.ToLower(filepath.ToSlash(outputRelative))
	switch {
	case strings.Contains(lower, "concept") || strings.Contains(lower, "idea") || strings.Contains(lower, "brainstorm"):
		return "idea"
	case strings.Contains(lower, "reflection") || strings.Contains(lower, "rethink") || strings.Contains(lower, "retro"):
		return "rethink"
	case strings.Contains(lower, "prd") ||
		strings.Contains(lower, "architecture") ||
		strings.Contains(lower, "uiux") ||
		strings.Contains(lower, "design") ||
		strings.Contains(lower, "research") ||
		strings.Contains(lower, "execution-plan") ||
		strings.Contains(lower, "frontend-blueprint"):
		return "design"
	case lower == "preview.html" || strings.HasPrefix(lower, "frontend/") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".css") || strings.HasSuffix(lower, ".js"):
		return "output"
	case strings.Contains(lower, "quality") ||
		strings.Contains(lower, "redteam") ||
		strings.Contains(lower, "task") ||
		strings.Contains(lower, "spec") ||
		strings.Contains(lower, "review") ||
		strings.Contains(lower, "cicd"):
		return "superdev"
	default:
		if strings.HasSuffix(lower, ".md") {
			return "design"
		}
		return "output"
	}
}

func prettyArtifactName(outputRelative string) string {
	lower := strings.ToLower(filepath.ToSlash(outputRelative))
	switch {
	case strings.HasSuffix(lower, "-concept.md"):
		return "构思增强稿"
	case strings.HasSuffix(lower, "-design-loop.md"):
		return "设计复核稿"
	case strings.HasSuffix(lower, "-reflection.md"):
		return "复盘再构思稿"
	case strings.HasSuffix(lower, "-prd.md"):
		return "PRD 文档"
	case strings.HasSuffix(lower, "-architecture.md"):
		return "架构文档"
	case strings.HasSuffix(lower, "-uiux.md"):
		return "UI/UX 文档"
	case strings.HasSuffix(lower, "-execution-plan.md"):
		return "执行计划"
	case strings.HasSuffix(lower, "-frontend-blueprint.md"):
		return "前端蓝图"
	case strings.HasSuffix(lower, "-quality-gate.md"):
		return "质量门禁报告"
	case strings.HasSuffix(lower, "-redteam.md"):
		return "红队报告"
	case lower == "preview.html":
		return "统一预览页面"
	case lower == "frontend/index.html":
		return "前端首页预览"
	case lower == "frontend/styles.css":
		return "前端样式"
	case lower == "frontend/app.js":
		return "前端脚本"
	default:
		return filepath.Base(outputRelative)
	}
}

func choosePrimaryPreviewURL(artifacts []pipelineArtifact) string {
	priorities := []string{"output/frontend/index.html", "output/preview.html", "frontend/index.html", "preview.html"}
	for _, candidate := range priorities {
		for _, artifact := range artifacts {
			if artifact.PreviewType == "html" && strings.EqualFold(artifact.Path, candidate) {
				return artifact.PreviewURL
			}
		}
	}
	for _, artifact := range artifacts {
		if artifact.PreviewType == "html" {
			return artifact.PreviewURL
		}
	}
	return ""
}

func buildPipelineStages(run store.PipelineRun, artifacts []pipelineArtifact) []pipelineStage {
	grouped := make(map[string][]pipelineArtifact, len(pipelineStageCatalog))
	for _, artifact := range artifacts {
		stageKey := strings.TrimSpace(artifact.Stage)
		if stageKey == "" {
			stageKey = inferArtifactStage(artifact.Path)
		}
		grouped[stageKey] = append(grouped[stageKey], artifact)
	}

	stages := make([]pipelineStage, 0, len(pipelineStageCatalog))
	for _, entry := range pipelineStageCatalog {
		stageArtifacts := append([]pipelineArtifact{}, grouped[entry.Key]...)
		sort.SliceStable(stageArtifacts, func(i, j int) bool {
			return stageArtifacts[i].Path < stageArtifacts[j].Path
		})
		stages = append(stages, pipelineStage{
			Key:       entry.Key,
			Title:     entry.Title,
			Status:    determinePipelineStageStatus(run, entry.Key, len(stageArtifacts) > 0),
			Artifacts: stageArtifacts,
		})
	}
	return stages
}

func determinePipelineStageStatus(run store.PipelineRun, stageKey string, hasArtifacts bool) string {
	if hasArtifacts {
		return "completed"
	}
	current := inferCurrentPipelineStage(run.Stage)
	runStatus := normalizeRunStatus(run.Status)
	if runStatus == "failed" {
		if stageKey == current {
			return "failed"
		}
		if pipelineStageOrder(stageKey) < pipelineStageOrder(current) {
			return "missing"
		}
		return "pending"
	}
	if runStatus == "completed" {
		return "missing"
	}
	if stageKey == current {
		return "in_progress"
	}
	if pipelineStageOrder(stageKey) < pipelineStageOrder(current) {
		return "missing"
	}
	if run.Status == "queued" && stageKey == "idea" {
		return "in_progress"
	}
	return "pending"
}

func inferCurrentPipelineStage(stage string) string {
	lower := strings.ToLower(strings.TrimSpace(stage))
	switch {
	case lower == "" || lower == "queued" || lower == "starting" || strings.Contains(lower, "context") || strings.Contains(lower, "idea"):
		return "idea"
	case strings.Contains(lower, "create") || strings.Contains(lower, "design") || strings.Contains(lower, "spec") || strings.Contains(lower, "docs"):
		return "design"
	case strings.Contains(lower, "task") || strings.Contains(lower, "quality") || strings.Contains(lower, "redteam") || strings.Contains(lower, "super-dev") || strings.Contains(lower, "iteration") || strings.Contains(lower, "acceptance"):
		return "superdev"
	case strings.Contains(lower, "preview") || strings.Contains(lower, "release") || strings.Contains(lower, "deploy"):
		return "output"
	case strings.Contains(lower, "rethink") || strings.Contains(lower, "reflection"):
		return "rethink"
	default:
		if lower == "done" {
			return "rethink"
		}
		return "superdev"
	}
}

func pipelineStageOrder(stageKey string) int {
	for idx, entry := range pipelineStageCatalog {
		if entry.Key == stageKey {
			return idx
		}
	}
	return len(pipelineStageCatalog)
}
