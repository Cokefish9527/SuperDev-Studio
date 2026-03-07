package agentconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectBundleFallsBackToDefault(t *testing.T) {
	bundle, err := LoadProjectBundle("")
	if err != nil {
		t.Fatalf("load default bundle: %v", err)
	}
	if len(bundle.Agents) == 0 || bundle.Agents[0].Name == "" {
		t.Fatalf("expected default agents in bundle")
	}
}

func TestLoadProjectBundleReadsProjectConfigs(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, ".studio-agent")
	for _, dir := range []string{"agents", "modes", "skills", "hooks", "commands"} {
		if err := os.MkdirAll(filepath.Join(configRoot, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(configRoot, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("agents/custom.yaml", "name: reviewer\ndescription: Custom reviewer\nallowed_tools:\n  - inspect\nmax_steps: 3\n")
	write("modes/review.yml", "name: review\ndescription: Review mode\nmax_retries: 2\nrequire_approval: true\n")
	write("skills/review.yaml", "name: quality\ndescription: Quality helper\nallowed_tools:\n  - inspect\n")
	write("hooks/post.yaml", "name: post-check\nstage: after_task\nactions:\n  - notify\n")
	write("commands/retry.yaml", "name: retry\ndescription: Retry command\nsteps:\n  - task status\n")

	bundle, err := LoadProjectBundle(root)
	if err != nil {
		t.Fatalf("load custom bundle: %v", err)
	}
	if len(bundle.Agents) != 1 || bundle.Agents[0].Name != "reviewer" {
		t.Fatalf("expected custom agent override, got %#v", bundle.Agents)
	}
	if len(bundle.Modes) != 1 || bundle.Modes[0].Name != "review" {
		t.Fatalf("expected custom mode override, got %#v", bundle.Modes)
	}
	if len(bundle.Commands) != 1 || bundle.Commands[0].Name != "retry" {
		t.Fatalf("expected custom command override, got %#v", bundle.Commands)
	}
}

func TestBundleResolveSelectionFallsBackToDefaults(t *testing.T) {
	bundle := defaultBundle()
	agent := bundle.ResolveAgent("missing")
	mode := bundle.ResolveMode("missing")
	if agent.Name != "delivery-agent" {
		t.Fatalf("expected fallback agent delivery-agent, got %q", agent.Name)
	}
	if mode.Name != "step_by_step" {
		t.Fatalf("expected fallback mode step_by_step, got %q", mode.Name)
	}
}

func TestBundleFindAgentAndMode(t *testing.T) {
	bundle := defaultBundle()
	if _, ok := bundle.FindAgent("delivery-agent"); !ok {
		t.Fatalf("expected delivery-agent to be found")
	}
	if _, ok := bundle.FindMode("step_by_step"); !ok {
		t.Fatalf("expected step_by_step mode to be found")
	}
}
