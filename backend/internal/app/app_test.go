package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDotEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "plain",
			line:      "VOLCENGINE_ARK_MODEL=ep-123",
			wantKey:   "VOLCENGINE_ARK_MODEL",
			wantValue: "ep-123",
		},
		{
			name:      "double quoted",
			line:      "VOLCENGINE_ARK_API_KEY=\"sk-demo\"",
			wantKey:   "VOLCENGINE_ARK_API_KEY",
			wantValue: "sk-demo",
		},
		{
			name:      "single quoted",
			line:      "VOLCENGINE_ARK_BASE_URL='https://example.com/api/v3'",
			wantKey:   "VOLCENGINE_ARK_BASE_URL",
			wantValue: "https://example.com/api/v3",
		},
		{
			name:    "invalid",
			line:    "INVALID_LINE",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			key, value, err := parseDotEnvLine(tc.line)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse line failed: %v", err)
			}
			if key != tc.wantKey {
				t.Fatalf("expected key %s, got %s", tc.wantKey, key)
			}
			if value != tc.wantValue {
				t.Fatalf("expected value %s, got %s", tc.wantValue, value)
			}
		})
	}
}

func TestLoadDotEnvFileSetsVariablesWithoutOverridingExisting(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")
	content := []byte(
		"# comment\n" +
			"VOLCENGINE_ARK_API_KEY=sk-from-dotenv\n" +
			"VOLCENGINE_ARK_MODEL=ep-from-dotenv\n" +
			"export SUPERDEV_STUDIO_ADDR=:9090\n",
	)
	if err := os.WriteFile(envFile, content, 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("VOLCENGINE_ARK_MODEL", "ep-from-env")
	if err := loadDotEnvFile(envFile); err != nil {
		t.Fatalf("load env file: %v", err)
	}

	if got := os.Getenv("VOLCENGINE_ARK_API_KEY"); got != "sk-from-dotenv" {
		t.Fatalf("expected api key from dotenv, got %s", got)
	}
	if got := os.Getenv("VOLCENGINE_ARK_MODEL"); got != "ep-from-env" {
		t.Fatalf("expected existing env to win, got %s", got)
	}
	if got := os.Getenv("SUPERDEV_STUDIO_ADDR"); got != ":9090" {
		t.Fatalf("expected addr from dotenv, got %s", got)
	}
}

func TestLoadConfigReadsDotEnv(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")
	content := []byte(
		"VOLCENGINE_ARK_API_KEY=sk-from-dotenv\n" +
			"VOLCENGINE_ARK_MODEL=ep-from-dotenv\n",
	)
	if err := os.WriteFile(envFile, content, 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv("VOLCENGINE_ARK_API_KEY", "")
	t.Setenv("VOLCENGINE_ARK_MODEL", "")

	cfg := LoadConfig()
	if cfg.VolcengineAPIKey != "sk-from-dotenv" {
		t.Fatalf("expected api key from dotenv, got %s", cfg.VolcengineAPIKey)
	}
	if cfg.VolcengineModel != "ep-from-dotenv" {
		t.Fatalf("expected model from dotenv, got %s", cfg.VolcengineModel)
	}
}

func TestLoadConfigReadsRateLimitEnv(t *testing.T) {
	t.Setenv("SUPERDEV_STUDIO_API_RATE_LIMIT_ENABLED", "false")
	t.Setenv("SUPERDEV_STUDIO_API_RATE_LIMIT_WINDOW", "90s")
	t.Setenv("SUPERDEV_STUDIO_API_RATE_LIMIT_MUTATION", "12")
	t.Setenv("SUPERDEV_STUDIO_API_RATE_LIMIT_EXPENSIVE", "5")
	t.Setenv("SUPERDEV_STUDIO_API_RATE_LIMIT_PIPELINE", "3")

	cfg := LoadConfig()
	if cfg.APIRateLimitEnabled {
		t.Fatalf("expected api rate limit to be disabled via env")
	}
	if cfg.APIRateLimitWindow != 90*time.Second {
		t.Fatalf("expected api rate limit window 90s, got %s", cfg.APIRateLimitWindow)
	}
	if cfg.APIMutationLimit != 12 {
		t.Fatalf("expected mutation limit 12, got %d", cfg.APIMutationLimit)
	}
	if cfg.APIExpensiveLimit != 5 {
		t.Fatalf("expected expensive limit 5, got %d", cfg.APIExpensiveLimit)
	}
	if cfg.APIPipelineLimit != 3 {
		t.Fatalf("expected pipeline limit 3, got %d", cfg.APIPipelineLimit)
	}
}

func TestLoadConfigReadsAutoAdvanceWorkerEnv(t *testing.T) {
	t.Setenv("SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_ENABLED", "false")
	t.Setenv("SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_INTERVAL", "15s")
	t.Setenv("SUPERDEV_STUDIO_AUTO_ADVANCE_WORKER_BATCH_SIZE", "3")

	cfg := LoadConfig()
	if cfg.AutoAdvanceWorkerEnabled {
		t.Fatalf("expected auto advance worker to be disabled via env")
	}
	if cfg.AutoAdvanceWorkerInterval != 15*time.Second {
		t.Fatalf("expected auto advance worker interval 15s, got %s", cfg.AutoAdvanceWorkerInterval)
	}
	if cfg.AutoAdvanceWorkerBatchSize != 3 {
		t.Fatalf("expected auto advance worker batch size 3, got %d", cfg.AutoAdvanceWorkerBatchSize)
	}
}
