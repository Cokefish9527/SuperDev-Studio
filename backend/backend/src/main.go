package main

import (
    "log"
    "net/http"
)

func coreHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"core","status":"todo"}`))
}

func searchHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"search","status":"todo"}`))
}

func analyticsHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"analytics","status":"todo"}`))
}

func notificationHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"notification","status":"todo"}`))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/core", coreHandler)
    http.HandleFunc("/api/search", searchHandler)
    http.HandleFunc("/api/analytics", analyticsHandler)
    http.HandleFunc("/api/notification", notificationHandler)
    log.Println("Backend scaffold running on :3001")
    log.Fatal(http.ListenAndServe(":3001", nil))
}
