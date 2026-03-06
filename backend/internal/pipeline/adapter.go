package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/shlex"
)

type RunRequest struct {
	Prompt     string
	ProjectDir string
	Platform   string
	Frontend   string
	Backend    string
	Domain     string
}

type Runner interface {
	RunPipeline(ctx context.Context, req RunRequest) ([]string, error)
	RunCommand(ctx context.Context, req RunRequest, commandArgs []string) ([]string, error)
}

type CommandAdapter struct {
	Command string
}

func NewCommandAdapter(command string) *CommandAdapter {
	if strings.TrimSpace(command) == "" {
		command = "python -m super_dev.cli"
	}
	return &CommandAdapter{Command: command}
}

func (a *CommandAdapter) RunPipeline(ctx context.Context, req RunRequest) ([]string, error) {
	args := []string{"pipeline", req.Prompt}
	if req.Platform != "" {
		args = append(args, "--platform", req.Platform)
	}
	if req.Frontend != "" {
		args = append(args, "--frontend", req.Frontend)
	}
	if req.Backend != "" {
		args = append(args, "--backend", req.Backend)
	}
	if req.Domain != "" {
		args = append(args, "--domain", req.Domain)
	}

	return a.RunCommand(ctx, req, args)
}

func (a *CommandAdapter) RunCommand(ctx context.Context, req RunRequest, commandArgs []string) ([]string, error) {
	parts, err := shlex.Split(a.Command)
	if err != nil {
		return nil, fmt.Errorf("invalid SUPER_DEV_CMD: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty SUPER_DEV_CMD")
	}
	if len(commandArgs) == 0 {
		return nil, fmt.Errorf("empty command args")
	}

	args := append([]string{}, parts[1:]...)
	args = append(args, commandArgs...)

	cmd := exec.CommandContext(ctx, parts[0], args...)
	if req.ProjectDir != "" {
		cmd.Dir = req.ProjectDir
	}
	output, err := cmd.CombinedOutput()
	lines := splitLines(string(output))
	if err != nil {
		return lines, fmt.Errorf("super-dev command failed: %w", err)
	}
	return lines, nil
}

func splitLines(raw string) []string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return []string{}
	}
	parts := strings.Split(normalized, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
