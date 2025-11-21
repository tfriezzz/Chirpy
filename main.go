package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	port := "8080"
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	var apiCfg apiConfig
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("GET /api/healthz", http.StripPrefix("/api/", http.HandlerFunc(handlerReadiness)))
	mux.Handle("GET /api/metrics", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerMetrics)))
	mux.Handle("POST /api/reset", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerReset)))

	fmt.Printf("server listening on port %s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	hits := cfg.fileserverHits.Load()
	strHits := fmt.Sprintf("Hits: %v", hits)
	w.WriteHeader(200)
	w.Write([]byte(strHits))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
