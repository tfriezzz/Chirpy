# Chirpy

Chirpy is a lightweight Go-based web API for creating and managing short messages ("chirps") like tweets. It demonstrates backend concepts including user auth with JWT/refresh tokens, database interactions (PostgreSQL), profanity filtering, and webhooks for user upgrades. Built as a practice project to explore Go server development.

## Why Use It?

- **Learning Tool**: Great for beginners to study real-world Go patterns: HTTP routing, auth, DB queries, and middleware.
- **Proof of Concept**: Quick setup for testing API ideas; extensible for features like social feeds or microblogging.
- **Simple & Secure**: Handles auth, validation, and filtering out-of-the-box, saving time on boilerplate.

## Installation & Running

1. **Clone the Repo**:
   ```
   git clone https://github.com/tfriezzz/Chirpy.git
   cd Chirpy
   ```

2. **Install Dependencies**:
   ```
   go mod tidy
   ```

3. **Set Up Environment**:
   Create `.env` with:
   ```
   DB_URL=postgres://user:password@localhost:5432/chirpy?sslmode=disable
   JWTSTRING=your-secret-key
   POLKA_KEY=your-polka-api-key
   ```
   Ensure PostgreSQL is running and schema is set (users, chirps, refresh_tokens tables via `internal/database`).

4. **Build & Run**:
   ```
   go build -o chirpy
   ./chirpy
   ```
   Server runs on `http://localhost:8080`.

## API Usage

All API endpoints under `/api/`. Auth-required endpoints use `Authorization: Bearer <JWT>` (or refresh token for specific ones).

### Auth & Users
- **POST /users**: Register. Body: `{"email": "string", "password": "string"}`. Returns user with JWT & refresh token.
- **POST /login**: Login. Body: same as above. Returns user with tokens.
- **PUT /users**: Update credentials. Auth required. Body: `{"email": "string", "password": "string"}`.
- **POST /refresh**: Get new JWT. Use `Bearer <refresh_token>`.
- **POST /revoke**: Revoke refresh token. Use `Bearer <refresh_token>`.

### Chirps
- **POST /chirps**: Create. Auth required. Body: `{"body": "string"}` (≤140 chars). Filters profanity (e.g., "kerfuffle" → "****").
- **GET /chirps**: List all. Query: `?author_id=<UUID>` to filter, `?sort=asc|desc` for time sort.
- **GET /chirps/{id}**: Get by ID (UUID).
- **DELETE /chirps/{id}**: Delete own chirp. Auth required.

### Webhooks & Upgrades
- **POST /polka/webhooks**: Upgrade user to premium ("Chirpy Red"). Requires `ApiKey` header. Body: `{"event": "user.upgraded", "data": {"user_id": "UUID"}}`.

### Admin & Health
- **GET /healthz**: Server status ("OK").
- **GET /admin/metrics**: File server hit count (HTML).
- **POST /admin/reset**: Clear users & reset metrics.

Static files served at `/app/`. Use tools like Postman for testing.
