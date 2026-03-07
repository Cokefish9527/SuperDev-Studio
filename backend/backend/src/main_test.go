package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlersReturnJSON(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{name: "core", handler: coreHandler, want: "\"module\":\"core\""},
		{name: "search", handler: searchHandler, want: "\"module\":\"search\""},
		{name: "analytics", handler: analyticsHandler, want: "\"module\":\"analytics\""},
		{name: "notification", handler: notificationHandler, want: "\"module\":\"notification\""},
		{name: "health", handler: healthHandler, want: "\"status\":\"ok\""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			res := httptest.NewRecorder()
			tc.handler(res, req)
			if got := res.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected json content type, got %q", got)
			}
			if !strings.Contains(res.Body.String(), tc.want) {
				t.Fatalf("expected body to contain %s, got %s", tc.want, res.Body.String())
			}
		})
	}
}
