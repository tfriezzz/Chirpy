package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tfriezzz/Chirpy/internal/auth"
	"github.com/tfriezzz/Chirpy/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
}

type userResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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
		return
	}

	if err != nil {
		log.Print(err)
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		log.Print(err)
	}

	JWTtoken, refreshToken, err := RJWTokenMaker(cfg, user)
	if err != nil {
		log.Printf("RJWTokenMaker returned err: %v", err)
	}
	if err := refreshTokenToDatabase(cfg, r, refreshToken, user.ID); err != nil {
		log.Printf("refreshTokenToDatabase returned err: %v", err)
	}

	if match {
		userResponse := dbUserToUserResponse(user, cfg, JWTtoken, refreshToken)
		// database.
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

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication failed")
		log.Printf("GetBearerToken returned err: %v", err)
		return
	}

	dbUser, err := cfg.DB.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication failed")
		log.Printf("GetUserFromRefreshToken returned err: %v", err)
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.JWTString, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication failed")
		log.Printf("MakeJWT returned err: %v", err)
	}
	respondWithJSON(w, 200, map[string]string{"token": token})
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	chirp := Chirp{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("decoder.Decode returned err: %v", err)
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication failed")
		return
	}
	userID, err := auth.ValidateJWT(tokenString, cfg.JWTString)
	if err != nil {
		respondWithError(w, 401, "authentication failed")
		return
	}

	// if err != nil {
	// 	err := respondWithError(w, 500, "Something went wrong")
	// 	if err != nil {
	// 		log.Printf("respondWithError returned error: %v", err)
	// 	}
	// }

	if len(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	if cleanedBody, profane := profanityChecker(chirp.Body); profane {
		fmt.Printf("ok: %v\n", profane)
		respondWithJSON(w, 200, map[string]string{"cleaned_body": cleanedBody})
		return
	}

	chirpParams := database.CreateChirpParams{
		chirp.Body, userID,
	}
	dbChirp, err := cfg.DB.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		fmt.Print(err)
	}

	if err := respondWithJSON(w, 201, Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't create chirp")
		return
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
		return
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
		Email: params.Email, HashedPassword: hashedPassword,
	}
	user, err := cfg.DB.CreateUser(r.Context(), userParams)
	if err != nil {
		fmt.Printf("CreateUser call: %v\n", err)
	}

	JWTtoken, refreshToken, err := RJWTokenMaker(cfg, user)
	if err != nil {
		log.Printf("RJWTokenMaker returned err: %v", err)
	}

	if err := refreshTokenToDatabase(cfg, r, refreshToken, user.ID); err != nil {
		log.Printf("refreshTokenToDatabase returned err: %v", err)
	}

	userResponse := dbUserToUserResponse(user, cfg, JWTtoken, refreshToken)

	if err := respondWithJSON(w, 201, userResponse); err != nil {
		fmt.Print(err)
	}
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
	_, err := w.Write([]byte(strHits))
	if err != nil {
		log.Printf("w.Write returned err: %v", err)
	}
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if err := cfg.DB.DeleteAllUsers(r.Context()); err != nil {
		fmt.Print(err)
	}
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "authentication failed")
		log.Printf("GetBearerToken returned err: %v", err)
		return
	}

	if err := cfg.DB.RevokeRefreshToken(r.Context(), refreshToken); err != nil {
		respondWithError(w, http.StatusUnauthorized, "can't revoke refreshToken")
		log.Printf("RevokeRefreshToken returned err: %v", err)
		return
	}

	respondWithJSON(w, 204, "")
}

func (cfg *apiConfig) handlerUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	JWTtoken, err := auth.GetBearerToken(r.Header)
	if err != nil || JWTtoken == "" {
		respondWithError(w, 401, "authentication failed")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := User{}
	if err := decoder.Decode(&params); err != nil {
		log.Printf("Decode returned err: %v", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("HashPassword returned err: %v", err)
		return
	}

	validatedUser, err := auth.ValidateJWT(JWTtoken, cfg.JWTString)
	if err != nil {
		respondWithError(w, 401, "authentication failed")
		return
	}

	userParams := database.UpdateUserCredentialsParams{
		ID:             validatedUser,
		Email:          params.Email,
		HashedPassword: hashedPassword,
	}
	dbUser, err := cfg.DB.UpdateUserCredentials(r.Context(), userParams)
	if err != nil {
		log.Printf("UpdateUserCredentials returned err: %v", err)
		return
	}

	// dbUser, err := cfg.DB.GetUserByEmail(r.Context(), params.Email)
	// fmt.Printf("USERbyEMAIL: %v, Email: %v", dbUser, params.Email)
	// if err != nil {
	// 	log.Printf("GetUserByEmail returnerd err %v", err)
	// 	return
	// }
	userResponse := userResponse{
		ID:    dbUser.ID,
		Email: dbUser.Email,
	}
	respondWithJSON(w, 200, userResponse)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	if _, err := w.Write(response); err != nil {
		return err
	}
	return nil
}

func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

func profanityChecker(str string) (string, bool) {
	// newStr := sta
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

func RJWTokenMaker(cfg *apiConfig, user database.User) (string, string, error) {
	JWTtoken, err := auth.MakeJWT(user.ID, cfg.JWTString, time.Hour)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		return "", "", err
	}
	return JWTtoken, refreshToken, nil
}

func refreshTokenToDatabase(cfg *apiConfig, r *http.Request, refreshToken string, user uuid.UUID) error {
	returnTokenParams := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_, err := cfg.DB.CreateRefreshToken(r.Context(), returnTokenParams)
	if err != nil {
		return err
	}
	return nil
}

func dbUserToUserResponse(u database.User, cfg *apiConfig, JWTtoken string, refreshToken string) userResponse {
	// var expiration time.Duration
	// if seconds != 0 && seconds >= 3600 {
	// 	expiration = time.Duration(seconds) * time.Second
	// } else {
	// expiration = time.Hour
	// }

	return userResponse{
		ID:           u.ID,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.CreatedAt,
		Email:        u.Email,
		Token:        JWTtoken,
		RefreshToken: refreshToken,
	}
}
