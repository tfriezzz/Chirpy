package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/tfriezzz/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
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
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.Handle("GET /api/healthz", http.StripPrefix("/api/", http.HandlerFunc(handlerReadiness)))
	mux.Handle("POST /api/chirps", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerChirps)))
	mux.Handle("POST /api/users", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerAddUser)))
	mux.Handle("GET /admin/metrics", http.StripPrefix("/admin/", http.HandlerFunc(apiCfg.handlerMetrics)))
	mux.Handle("POST /admin/reset", http.StripPrefix("/admin/", http.HandlerFunc(apiCfg.handlerReset)))

	fmt.Printf("server listening on port %s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	// fmt.Print("hi from handlerValidation")
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	fmt.Printf("params: %v\n", params.Body)
	fmt.Printf("error: %v\n", err)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else {
		if cleanedBody, profane := profanityChecker(params.Body); profane {
			fmt.Printf("ok: %v\n", profane)
			respondWithJSON(w, 200, map[string]string{"cleaned_body": cleanedBody})
		} else {
			respondWithJSON(w, 200, map[string]string{"cleaned_body": params.Body})
		}
	}
}

func (cfg *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := User{}
	if err := decoder.Decode(&params); err != nil {
		fmt.Print(err)
	}
	// fmt.Printf("email: %v\n", params.Email)
	user, err := cfg.DB.CreateUser(r.Context(), params.Email)
	if err != nil {
		fmt.Printf("CreateUser call: %v\n", err)
	}
	if err := respondWithJSON(w, 201, User(user)); err != nil {
		fmt.Print(err)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}

func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	hits := cfg.fileserverHits.Load()
	strHits := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
	w.WriteHeader(200)
	w.Write([]byte(strHits))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if err := cfg.DB.DeleteAllUsers(r.Context()); err != nil {
		fmt.Print(err)
	}
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func profanityChecker(str string) (string, bool) {
	// newStr := str
	isProfane := false

	profanities := []string{"kerfuffle", "sharbert", "fornax"}

	strSplit := strings.Split(str, " ")
	strLower := strings.ToLower(str)
	lowerStrSplit := strings.Split(strLower, " ")

	for i, word := range lowerStrSplit {
		for _, profanity := range profanities {
			if word == profanity {
				strSplit[i] = "****"
				isProfane = true
			}
		}
	}
	return strings.Join(strSplit, " "), isProfane
}
