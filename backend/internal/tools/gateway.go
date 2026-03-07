package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Definition struct {
	Name           string
	Description    string
	InputSchema    map[string]any
	OutputSchema   map[string]any
	RiskLevel      RiskLevel
	Retryable      bool
	DefaultTimeout time.Duration
	Handler        Handler
}

type Handler func(ctx context.Context, arguments map[string]any) (map[string]any, error)

type Gateway struct {
	definitions map[string]Definition
}

func NewGateway() *Gateway {
	return &Gateway{definitions: map[string]Definition{}}
}

func (g *Gateway) Register(def Definition) {
	if strings.TrimSpace(def.Name) == "" {
		return
	}
	g.definitions[strings.TrimSpace(def.Name)] = def
}

func (g *Gateway) Get(name string) (Definition, bool) {
	def, ok := g.definitions[strings.TrimSpace(name)]
	return def, ok
}

func (g *Gateway) List() []Definition {
	items := make([]Definition, 0, len(g.definitions))
	for _, def := range g.definitions {
		items = append(items, def)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func (g *Gateway) Execute(ctx context.Context, name string, arguments map[string]any) (map[string]any, error) {
	def, ok := g.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool %s is not registered", name)
	}
	if def.Handler == nil {
		return nil, fmt.Errorf("tool %s has no handler", name)
	}
	return def.Handler(ctx, arguments)
}

func RegisterBuiltinDefinitions(g *Gateway) {
	if g == nil {
		return
	}
	definitions := []Definition{
		{Name: "search_context", Description: "Search evidence from memory, knowledge, tasks, and prior runs.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 10 * time.Second},
		{Name: "run_superdev_create", Description: "Execute super-dev create for the current delivery goal.", RiskLevel: RiskMedium, Retryable: false, DefaultTimeout: 10 * time.Minute},
		{Name: "run_superdev_spec_validate", Description: "Execute super-dev spec validate for the current change.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 2 * time.Minute},
		{Name: "run_superdev_task_status", Description: "Execute super-dev task status for the current change.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 2 * time.Minute},
		{Name: "run_superdev_task_run", Description: "Execute super-dev task run for the current change.", RiskLevel: RiskMedium, Retryable: true, DefaultTimeout: 20 * time.Minute},
		{Name: "run_superdev_quality", Description: "Execute super-dev quality checks.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 10 * time.Minute},
		{Name: "run_superdev_preview", Description: "Execute super-dev preview generation.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 5 * time.Minute},
		{Name: "run_superdev_deploy", Description: "Execute super-dev deploy generation.", RiskLevel: RiskHigh, Retryable: false, DefaultTimeout: 10 * time.Minute},
		{Name: "read_artifact", Description: "Read generated artifact content.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 5 * time.Second},
		{Name: "append_run_event", Description: "Append an event to the pipeline run timeline.", RiskLevel: RiskLow, Retryable: true, DefaultTimeout: 5 * time.Second},
	}
	for _, def := range definitions {
		g.Register(def)
	}
}
