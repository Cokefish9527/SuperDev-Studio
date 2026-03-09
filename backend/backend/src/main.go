package main

import (
    "log"
    "net/http"
)

func coreHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"core","status":"todo"}`))
}

func profileHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"module":"profile","status":"todo"}`))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/api/core", coreHandler)
    http.HandleFunc("/api/profile", profileHandler)
    log.Println("Backend scaffold running on :3001")
    log.Fatal(http.ListenAndServe(":3001", nil))
}
