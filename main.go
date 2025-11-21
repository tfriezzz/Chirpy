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
	mux.HandleFunc("/healthz", handlerReadiness)
	mux.HandleFunc("/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("/reset", apiCfg.handlerReset)

	fmt.Printf("server listening on port %s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	// fmt.Print("hi from handleReadiness")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	// j, _ := json.Marshal("OK")
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("hi form numberOfRequests")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	hits := cfg.fileserverHits.Load()
	strHits := fmt.Sprintf("Hits: %v", hits)
	// fmt.Printf("test %v\n", hits)
	w.WriteHeader(200)
	// j, _ := json.Marshal(strHits)
	w.Write([]byte(strHits))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	// return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	cfg.fileserverHits.Store(0)
	// 	// next.ServeHTTP(w, r)
	// })

	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// fmt.Println("hi from middlewareMetricsInc")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// func middleWareCfgWrap(handler func(cfg *apiConfig, w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
// 	newHandler := func(w http.ResponseWriter, r *http.Request) {
// 		handler(cfg, w, r)
// 	}
// 	fmt.Printf("hi from middleWareCfgWrap")
//
// 	return newHandler
// }
