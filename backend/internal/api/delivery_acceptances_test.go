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

	"superdevstudio/internal/store"
)

func TestUpdateRunDeliveryAcceptanceEndpointRecordsAcceptedState(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "DeliveryAcceptanceReady")
	projectDir := filepath.Join(t.TempDir(), "delivery-acceptance-ready")
	frontendDir := filepath.Join(projectDir, "output", "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("create frontend output dir: %v", err)
	}
	writeTestArtifact(t, filepath.Join(frontendDir, "index.html"), "<!doctype html><html><body>ready</body></html>")
	writeTestArtifact(t, filepath.Join(projectDir, "output", "superdev-studio-quality-gate.md"), "quality passed")
	writeTestArtifact(t, filepath.Join(projectDir, "output", "superdev-studio-task-execution.md"), "task execution")

	run, err := env.store.CreatePipelineRun(context.Background(), store.PipelineRun{
		ProjectID:       project.ID,
		Prompt:          "Ship the release candidate",
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
	if _, err := env.store.AppendRunEvent(context.Background(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "lifecycle-quality",
		Status:  "completed",
		Message: "Quality gate passed on iteration 1",
	}); err != nil {
		t.Fatalf("append quality event: %v", err)
	}
	if _, err := env.store.UpsertPreviewSession(context.Background(), store.PreviewSession{
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		PreviewURL:    "/api/pipeline/runs/" + run.ID + "/preview/frontend/index.html",
		PreviewType:   "html",
		Title:         "Final preview",
		SourceKey:     "preview:" + run.ID,
		Status:        "accepted",
		ReviewerNote:  "Looks good",
	}); err != nil {
		t.Fatalf("seed accepted preview session: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{"status": "accepted"})
	req := httptest.NewRequest(http.MethodPut, "/api/pipeline/runs/"+run.ID+"/delivery-acceptance", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var updated store.DeliveryAcceptance
	if err := json.Unmarshal(res.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode accepted delivery acceptance: %v", err)
	}
	if updated.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", updated.Status)
	}
	if updated.ReviewedAt == nil {
		t.Fatal("expected reviewed_at to be populated")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/pipeline/runs/"+run.ID+"/delivery-acceptance", nil)
	getRes := httptest.NewRecorder()
	env.handler.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200 from get, got %d: %s", getRes.Code, getRes.Body.String())
	}
	var got store.DeliveryAcceptance
	if err := json.Unmarshal(getRes.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode stored delivery acceptance: %v", err)
	}
	if got.Status != "accepted" {
		t.Fatalf("expected stored accepted status, got %q", got.Status)
	}
	events, err := env.store.ListRunEvents(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	matched := false
	for _, item := range events {
		if item.Stage == "lifecycle-acceptance" && strings.Contains(item.Message, "recorded") {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("expected lifecycle-acceptance event after recording final acceptance, got %#v", events)
	}
}

func TestUpdateRunDeliveryAcceptanceEndpointRejectsRunWithoutAcceptedPreview(t *testing.T) {
	env := newAPITestEnv(t)
	project := createProjectViaAPI(t, env.handler, "DeliveryAcceptanceBlocked")
	projectDir := filepath.Join(t.TempDir(), "delivery-acceptance-blocked")
	frontendDir := filepath.Join(projectDir, "output", "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatalf("create frontend output dir: %v", err)
	}
	writeTestArtifact(t, filepath.Join(frontendDir, "index.html"), "<!doctype html><html><body>blocked</body></html>")
	writeTestArtifact(t, filepath.Join(projectDir, "output", "superdev-studio-quality-gate.md"), "quality passed")
	writeTestArtifact(t, filepath.Join(projectDir, "output", "superdev-studio-task-execution.md"), "task execution")

	run, err := env.store.CreatePipelineRun(context.Background(), store.PipelineRun{
		ProjectID:       project.ID,
		Prompt:          "Attempt sign-off too early",
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
	if _, err := env.store.AppendRunEvent(context.Background(), store.RunEvent{
		RunID:   run.ID,
		Stage:   "lifecycle-quality",
		Status:  "completed",
		Message: "Quality gate passed on iteration 1",
	}); err != nil {
		t.Fatalf("append quality event: %v", err)
	}
	if _, err := env.store.UpsertPreviewSession(context.Background(), store.PreviewSession{
		ProjectID:     project.ID,
		PipelineRunID: run.ID,
		PreviewURL:    "/api/pipeline/runs/" + run.ID + "/preview/frontend/index.html",
		PreviewType:   "html",
		Title:         "Final preview",
		SourceKey:     "preview:" + run.ID,
		Status:        "generated",
	}); err != nil {
		t.Fatalf("seed generated preview session: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{"status": "accepted"})
	req := httptest.NewRequest(http.MethodPut, "/api/pipeline/runs/"+run.ID+"/delivery-acceptance", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	env.handler.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "preview must be accepted") {
		t.Fatalf("expected preview acceptance blocker, got %s", res.Body.String())
	}
}

func writeTestArtifact(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write artifact %s: %v", path, err)
	}
}
