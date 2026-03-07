package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type VolcengineAdvisor struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewVolcengineAdvisor(apiKey, model, baseURL string) *VolcengineAdvisor {
	trimmedBase := strings.TrimSpace(baseURL)
	if trimmedBase == "" {
		trimmedBase = "https://ark.cn-beijing.volces.com/api/v3"
	}
	return &VolcengineAdvisor{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: strings.TrimRight(trimmedBase, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (v *VolcengineAdvisor) Enabled() bool {
	return strings.TrimSpace(v.apiKey) != "" && strings.TrimSpace(v.model) != ""
}

func (v *VolcengineAdvisor) Advise(ctx context.Context, prompt string) (string, error) {
	return v.AdviseWithAssets(ctx, prompt, nil)
}

func (v *VolcengineAdvisor) AdviseWithAssets(ctx context.Context, prompt string, assetURLs []string) (string, error) {
	if !v.Enabled() {
		return "", errors.New("volcengine advisor is not configured")
	}
	userPrompt := strings.TrimSpace(prompt)
	if userPrompt == "" {
		return "", errors.New("prompt is required")
	}

	requestBody := map[string]any{
		"model":       v.model,
		"temperature": 0.2,
		"messages": []map[string]any{
			{
				"role":    "system",
				"content": "你是资深软件交付专家，请输出可执行、可验证的工程建议。",
			},
			{
				"role": "user",
				"content": func() any {
					if len(assetURLs) == 0 {
						return userPrompt
					}
					content := []map[string]any{{
						"type": "text",
						"text": userPrompt,
					}}
					for _, raw := range assetURLs {
						trimmed := strings.TrimSpace(raw)
						if trimmed == "" {
							continue
						}
						content = append(content, map[string]any{
							"type":      "image_url",
							"image_url": map[string]string{"url": trimmed},
						})
					}
					return content
				}(),
			},
		},
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&response); decodeErr != nil {
		return "", decodeErr
	}
	if resp.StatusCode >= http.StatusBadRequest {
		if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
			return "", fmt.Errorf("volcengine api error: %s", response.Error.Message)
		}
		return "", fmt.Errorf("volcengine api error: status %d", resp.StatusCode)
	}
	if len(response.Choices) == 0 {
		return "", errors.New("volcengine response has no choices")
	}

	content := extractContentText(response.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return "", errors.New("volcengine response content is empty")
	}
	return strings.TrimSpace(content), nil
}

func extractContentText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			switch typed := item.(type) {
			case map[string]any:
				if text, ok := typed["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}
