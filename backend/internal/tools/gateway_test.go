package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGatewayRegisterListExecute(t *testing.T) {
	gateway := NewGateway()
	gateway.Register(Definition{Name: "  ", Description: "ignored"})
	gateway.Register(Definition{Name: "beta", Handler: func(_ context.Context, args map[string]any) (map[string]any, error) {
		return map[string]any{"echo": args["input"]}, nil
	}})
	gateway.Register(Definition{Name: "alpha", Handler: nil})

	items := gateway.List()
	if len(items) != 2 || items[0].Name != "alpha" || items[1].Name != "beta" {
		t.Fatalf("expected sorted definitions, got %#v", items)
	}
	if _, ok := gateway.Get("beta"); !ok {
		t.Fatalf("expected beta to be registered")
	}
	if _, err := gateway.Execute(context.Background(), "missing", nil); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected missing tool error, got %v", err)
	}
	if _, err := gateway.Execute(context.Background(), "alpha", nil); err == nil || !strings.Contains(err.Error(), "no handler") {
		t.Fatalf("expected missing handler error, got %v", err)
	}
	result, err := gateway.Execute(context.Background(), "beta", map[string]any{"input": "ok"})
	if err != nil {
		t.Fatalf("execute beta: %v", err)
	}
	if result["echo"] != "ok" {
		t.Fatalf("expected echo result, got %#v", result)
	}
}

func TestRegisterBuiltinDefinitions(t *testing.T) {
	gateway := NewGateway()
	RegisterBuiltinDefinitions(gateway)
	items := gateway.List()
	if len(items) == 0 {
		t.Fatalf("expected builtin definitions")
	}
	preview, ok := gateway.Get("run_superdev_preview")
	if !ok {
		t.Fatalf("expected preview definition to exist")
	}
	if preview.DefaultTimeout != 5*time.Minute {
		t.Fatalf("expected preview timeout 5m, got %s", preview.DefaultTimeout)
	}
	RegisterBuiltinDefinitions(nil)
}
