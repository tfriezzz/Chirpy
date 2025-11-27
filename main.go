package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/tfriezzz/Chirpy/internal/auth"
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
	Password  string    `json:"password"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
	mux.Handle("GET /api/chirps", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerGetAllChirps)))
	mux.Handle("GET /api/chirps/{chirpID}", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerGetChirp)))
	mux.Handle("POST /api/users", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerAddUser)))
	mux.Handle("POST /api/login", http.StripPrefix("/api/", http.HandlerFunc(apiCfg.handlerLogin)))
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

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := User{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Print(err)
	}

	user, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		log.Print(err)
		err := respondWithError(w, 401, "Incorrect email or password")
		if err != nil {
			log.Print(err)
		}
	}

	if err != nil {
		log.Print(err)
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		log.Print(err)
	}
	if match {
		userResponse := dbUserToUserResponse(user)
		err := respondWithJSON(w, 200, userResponse)
		if err != nil {
			log.Print(err)
		}
	} else {

		err := respondWithError(w, 401, "Incorrect email or password")
		if err != nil {
			log.Print(err)
		}
	}
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	chirp := Chirp{}
	err := decoder.Decode(&chirp)
	fmt.Printf("params: %v\n", chirp.Body)
	fmt.Printf("error: %v\n", err)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
	}
	if len(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else {
		if cleanedBody, profane := profanityChecker(chirp.Body); profane {
			fmt.Printf("ok: %v\n", profane)
			respondWithJSON(w, 200, map[string]string{"cleaned_body": cleanedBody})
		} else {
			chirpParams := database.CreateChirpParams{
				chirp.Body, chirp.UserID,
			}
			chirp, err := cfg.DB.CreateChirp(r.Context(), chirpParams)
			if err != nil {
				fmt.Print(err)
			}

			respondWithJSON(w, 201, Chirp(chirp))
		}
	}
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	strID := r.PathValue("chirpID")
	fmt.Printf("IDtest: %v\n", strID)
	chirpID, err := uuid.Parse(strID)
	if err != nil {
		log.Print(err)
	}

	chirp, err := cfg.DB.GetChirpsByID(r.Context(), chirpID)
	if err != nil {
		log.Print(err)
		respondWithError(w, 404, "chirp not found")
	}
	// fmt.Printf("test: %v\n", chirp)
	respondWithJSON(w, 200, Chirp(chirp))
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.DB.GetAllChirps(r.Context())
	var chirpsSlice []Chirp
	// var chirpsArray [chirpLen]Chirp
	if err != nil {
		log.Print(err)
	}
	for _, c := range chirps {
		chirp := Chirp(c)
		chirpsSlice = append(chirpsSlice, chirp)
	}
	if err := respondWithJSON(w, 200, chirpsSlice); err != nil {
		log.Print(err)
	}
}

func (cfg *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := User{}
	if err := decoder.Decode(&params); err != nil {
		fmt.Print(err)
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Print(err)
	}
	userParams := database.CreateUserParams{
		params.Email, hashedPassword,
	}
	user, err := cfg.DB.CreateUser(r.Context(), userParams)
	if err != nil {
		fmt.Printf("CreateUser call: %v\n", err)
	}

	userResponse := dbUserToUserResponse(user)

	if err := respondWithJSON(w, 201, userResponse); err != nil {
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

func dbUserToUserResponse(u database.User) userResponse {
	return userResponse{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.CreatedAt,
		Email:     u.Email,
	}
}
