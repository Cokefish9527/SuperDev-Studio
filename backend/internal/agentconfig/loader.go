package agentconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Bundle struct {
	Agents   []AgentConfig   `json:"agents"`
	Modes    []ModeConfig    `json:"modes"`
	Skills   []SkillConfig   `json:"skills"`
	Hooks    []HookConfig    `json:"hooks"`
	Commands []CommandConfig `json:"commands"`
}

type AgentConfig struct {
	Name          string   `json:"name" yaml:"name"`
	Description   string   `json:"description" yaml:"description"`
	DefaultModel  string   `json:"default_model" yaml:"default_model"`
	AllowedTools  []string `json:"allowed_tools" yaml:"allowed_tools"`
	DefaultSkills []string `json:"default_skills" yaml:"default_skills"`
	MaxSteps      int      `json:"max_steps" yaml:"max_steps"`
}

type ModeConfig struct {
	Name            string `json:"name" yaml:"name"`
	Description     string `json:"description" yaml:"description"`
	AllowDeploy     bool   `json:"allow_deploy" yaml:"allow_deploy"`
	MaxRetries      int    `json:"max_retries" yaml:"max_retries"`
	RequireApproval bool   `json:"require_approval" yaml:"require_approval"`
}

type SkillConfig struct {
	Name             string   `json:"name" yaml:"name"`
	Description      string   `json:"description" yaml:"description"`
	PromptFragments  []string `json:"prompt_fragments" yaml:"prompt_fragments"`
	AllowedTools     []string `json:"allowed_tools" yaml:"allowed_tools"`
	PreferredSources []string `json:"preferred_sources" yaml:"preferred_sources"`
}

type HookConfig struct {
	Name        string   `json:"name" yaml:"name"`
	Stage       string   `json:"stage" yaml:"stage"`
	Description string   `json:"description" yaml:"description"`
	Actions     []string `json:"actions" yaml:"actions"`
}

type CommandConfig struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Steps       []string `json:"steps" yaml:"steps"`
}

func LoadProjectBundle(projectDir string) (Bundle, error) {
	root := strings.TrimSpace(projectDir)
	if root == "" {
		return defaultBundle(), nil
	}
	configRoot := filepath.Join(root, ".studio-agent")
	if _, err := os.Stat(configRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultBundle(), nil
		}
		return Bundle{}, err
	}
	bundle := defaultBundle()
	if err := loadConfigs(filepath.Join(configRoot, "agents"), &bundle.Agents); err != nil {
		return Bundle{}, err
	}
	if err := loadConfigs(filepath.Join(configRoot, "modes"), &bundle.Modes); err != nil {
		return Bundle{}, err
	}
	if err := loadConfigs(filepath.Join(configRoot, "skills"), &bundle.Skills); err != nil {
		return Bundle{}, err
	}
	if err := loadConfigs(filepath.Join(configRoot, "hooks"), &bundle.Hooks); err != nil {
		return Bundle{}, err
	}
	if err := loadConfigs(filepath.Join(configRoot, "commands"), &bundle.Commands); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func defaultBundle() Bundle {
	return Bundle{
		Agents: []AgentConfig{{
			Name:          "delivery-agent",
			Description:   "Default delivery agent for step-by-step software delivery.",
			AllowedTools:  []string{"search_context", "run_superdev_create", "run_superdev_task_status", "run_superdev_task_run", "run_superdev_quality", "read_artifact", "append_run_event"},
			DefaultSkills: []string{"super-dev-delivery"},
			MaxSteps:      24,
		}},
		Modes: []ModeConfig{{
			Name:        "step_by_step",
			Description: "Execute create -> spec validate -> task status -> task run -> quality -> preview -> deploy with repair loops.",
			MaxRetries:  3,
		}},
		Skills: []SkillConfig{{
			Name:             "super-dev-delivery",
			Description:      "Drive super-dev delivery workflow with evidence and quality repair.",
			AllowedTools:     []string{"search_context", "run_superdev_create", "run_superdev_task_status", "run_superdev_task_run", "run_superdev_quality"},
			PreferredSources: []string{"memory", "knowledge", "run", "task"},
		}},
	}
}

func loadConfigs[T any](dir string, target *[]T) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	items := make([]T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		var item T
		if err := yaml.Unmarshal(content, &item); err != nil {
			return err
		}
		items = append(items, item)
	}
	if len(items) > 0 {
		*target = items
	}
	return nil
}
