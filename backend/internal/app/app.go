package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"superdevstudio/internal/agentruntime/eino"
	"superdevstudio/internal/api"
	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/llm"
	"superdevstudio/internal/pipeline"
	"superdevstudio/internal/retrieval"
	"superdevstudio/internal/store"
)

type Config struct {
	Addr                string
	DBPath              string
	SuperDevCmd         string
	SuperDevWorkdir     string
	VolcengineAPIKey    string
	VolcengineModel     string
	VolcengineBaseURL   string
	APIRateLimitEnabled bool
	APIRateLimitWindow  time.Duration
	APIMutationLimit    int
	APIExpensiveLimit   int
	APIPipelineLimit    int
}

func LoadConfig() Config {
	loadDotEnv()

	return Config{
		Addr:                envOrDefault("SUPERDEV_STUDIO_ADDR", ":8080"),
		DBPath:              envOrDefault("SUPERDEV_STUDIO_DB", "./data/superdev_studio.db"),
		SuperDevCmd:         envOrDefault("SUPER_DEV_CMD", "python -m super_dev.cli"),
		SuperDevWorkdir:     envOrDefault("SUPER_DEV_WORKDIR", ""),
		VolcengineAPIKey:    envOrDefault("VOLCENGINE_ARK_API_KEY", ""),
		VolcengineModel:     envOrDefault("VOLCENGINE_ARK_MODEL", ""),
		VolcengineBaseURL:   envOrDefault("VOLCENGINE_ARK_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3"),
		APIRateLimitEnabled: envBoolOrDefault("SUPERDEV_STUDIO_API_RATE_LIMIT_ENABLED", true),
		APIRateLimitWindow:  envDurationOrDefault("SUPERDEV_STUDIO_API_RATE_LIMIT_WINDOW", time.Minute),
		APIMutationLimit:    envIntOrDefault("SUPERDEV_STUDIO_API_RATE_LIMIT_MUTATION", 24),
		APIExpensiveLimit:   envIntOrDefault("SUPERDEV_STUDIO_API_RATE_LIMIT_EXPENSIVE", 10),
		APIPipelineLimit:    envIntOrDefault("SUPERDEV_STUDIO_API_RATE_LIMIT_PIPELINE", 6),
	}
}

func loadDotEnv() {
	_ = loadDotEnvFile(".env")
	_ = loadDotEnvFile(filepath.Join("backend", ".env"))
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		if strings.HasPrefix(raw, "export ") {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "export "))
		}
		key, value, parseErr := parseDotEnvLine(raw)
		if parseErr != nil {
			return fmt.Errorf("parse %s:%d failed: %w", path, lineNum, parseErr)
		}
		if key == "" {
			continue
		}
		if existing, exists := os.LookupEnv(key); exists && strings.TrimSpace(existing) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

func parseDotEnvLine(raw string) (string, string, error) {
	idx := strings.Index(raw, "=")
	if idx <= 0 {
		return "", "", errors.New("invalid line, expected KEY=VALUE")
	}
	key := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+1:])
	if key == "" {
		return "", "", errors.New("empty key")
	}
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
	}
	return key, value, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

type App struct {
	cfg        Config
	store      *store.Store
	httpServer *http.Server
}

func New(cfg Config) (*App, error) {
	dbDir := filepath.Dir(cfg.DBPath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return nil, err
		}
	}

	s, err := store.New(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	contextOpt := contextopt.NewService(s)
	pipelineManager := pipeline.NewManager(s, pipeline.NewCommandAdapter(cfg.SuperDevCmd), contextOpt)
	volcAdvisor := llm.NewVolcengineAdvisor(cfg.VolcengineAPIKey, cfg.VolcengineModel, cfg.VolcengineBaseURL)
	if volcAdvisor.Enabled() {
		pipelineManager.SetLLMAdvisor(volcAdvisor)
	}
	retrievalService := retrieval.NewService(s)
	einoRuntime, einoErr := eino.New(context.Background(), s, retrievalService, eino.Config{
		APIKey:  cfg.VolcengineAPIKey,
		Model:   cfg.VolcengineModel,
		BaseURL: cfg.VolcengineBaseURL,
	})
	if einoErr == nil && einoRuntime != nil {
		pipelineManager.SetAgentRuntime(einoRuntime)
	}
	apiServer := api.NewServerWithConfig(s, pipelineManager, contextOpt, api.ServerConfig{
		RateLimit: api.RateLimitConfig{
			Enabled:        cfg.APIRateLimitEnabled,
			Window:         cfg.APIRateLimitWindow,
			MutationLimit:  cfg.APIMutationLimit,
			ExpensiveLimit: cfg.APIExpensiveLimit,
			PipelineLimit:  cfg.APIPipelineLimit,
		},
	})

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &App{cfg: cfg, store: s, httpServer: httpServer}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		err := a.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = a.httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Close() error {
	return a.store.Close()
}
