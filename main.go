package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	port := "8080"
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", handleReadiness)

	fmt.Printf("server listening on port %s", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	j, _ := json.Marshal("OK")
	w.Write(j)
}
