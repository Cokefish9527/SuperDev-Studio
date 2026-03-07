package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVolcengineAdvisorEnabled(t *testing.T) {
	disabled := NewVolcengineAdvisor("", "", "")
	if disabled.Enabled() {
		t.Fatalf("expected advisor disabled without api key and model")
	}
	enabled := NewVolcengineAdvisor("test-key", "ep-123", "")
	if !enabled.Enabled() {
		t.Fatalf("expected advisor enabled with api key and model")
	}
}

func TestVolcengineAdvisorAdviseSuccessStringContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer token, got %s", got)
		}

		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "ep-123" {
			t.Fatalf("expected model ep-123, got %s", payload.Model)
		}
		if len(payload.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(payload.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"第一条建议"}}]}`))
	}))
	defer server.Close()

	advisor := NewVolcengineAdvisor("test-key", "ep-123", server.URL)
	answer, err := advisor.Advise(context.Background(), "请给出修复建议")
	if err != nil {
		t.Fatalf("advise failed: %v", err)
	}
	if answer != "第一条建议" {
		t.Fatalf("unexpected answer: %s", answer)
	}
}

func TestVolcengineAdvisorAdviseSuccessArrayContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"建议A"},{"type":"text","text":"建议B"}]}}]}`))
	}))
	defer server.Close()

	advisor := NewVolcengineAdvisor("test-key", "ep-123", server.URL)
	answer, err := advisor.Advise(context.Background(), "请给出修复建议")
	if err != nil {
		t.Fatalf("advise failed: %v", err)
	}
	if answer != "建议A\n建议B" {
		t.Fatalf("unexpected answer: %s", answer)
	}
}

func TestVolcengineAdvisorAdviseWithAssetsBuildsMultimodalPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(payload.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(payload.Messages))
		}
		content, ok := payload.Messages[1].Content.([]any)
		if !ok {
			t.Fatalf("expected multimodal content array, got %#v", payload.Messages[1].Content)
		}
		if len(content) != 2 {
			t.Fatalf("expected text + image content, got %d items", len(content))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"多模态建议"}}]}`))
	}))
	defer server.Close()

	advisor := NewVolcengineAdvisor("test-key", "ep-vision", server.URL)
	answer, err := advisor.AdviseWithAssets(context.Background(), "请基于视觉参考给出建议", []string{"https://example.com/demo.png"})
	if err != nil {
		t.Fatalf("advise with assets failed: %v", err)
	}
	if answer != "多模态建议" {
		t.Fatalf("unexpected answer: %s", answer)
	}
}

func TestVolcengineAdvisorAdviseAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":{"message":"invalid request"}}`))
	}))
	defer server.Close()

	advisor := NewVolcengineAdvisor("test-key", "ep-123", server.URL)
	_, err := advisor.Advise(context.Background(), "请给出修复建议")
	if err == nil {
		t.Fatalf("expected api error")
	}
	if !strings.Contains(err.Error(), "invalid request") {
		t.Fatalf("expected invalid request message, got %v", err)
	}
}

func TestVolcengineAdvisorAdviseRequiresPrompt(t *testing.T) {
	advisor := NewVolcengineAdvisor("test-key", "ep-123", "https://ark.cn-beijing.volces.com/api/v3")
	_, err := advisor.Advise(context.Background(), "")
	if err == nil {
		t.Fatalf("expected prompt validation error")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
