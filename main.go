package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/tfriezzz/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	JWTString      string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	JWTString := os.Getenv("JWTSTRING")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Print(err)
	}
	dbQueries := database.New(db)
	port := "8080"
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	var apiCfg apiConfig
	apiCfg.DB = dbQueries
	apiCfg.JWTString = JWTString
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("GET /api/healthz", http.StripPrefix("/api/", http.HandlerFunc(handlerReadiness)))
	mux.Handle("POST /api/chirps", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerPostChirp)))
	mux.Handle("GET /api/chirps", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerGetAllChirps)))
	mux.Handle("GET /api/chirps/{chirpID}", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerGetChirp)))
	mux.Handle("DELETE /api/chirps/{chirpID}", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerDeleteChirp)))
	mux.Handle("POST /api/users", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerAddUser)))
	mux.Handle("PUT /api/users", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerUpdateCredentials)))
	mux.Handle("POST /api/login", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerLogin)))
	mux.Handle("POST /api/refresh", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerRefresh)))
	mux.Handle("POST /api/revoke", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerRevoke)))
	mux.Handle("POST /api/polka/webhooks", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerUpgradeToRed)))
	mux.Handle("GET /admin/metrics", http.StripPrefix("/admin/", http.HandlerFunc(apiCfg.handlerMetrics)))
	mux.Handle("POST /admin/reset", http.StripPrefix("/admin/", http.HandlerFunc(apiCfg.handlerReset)))

	fmt.Printf("server listening on port %s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}
