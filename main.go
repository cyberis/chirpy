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

	"github.com/google/uuid"

	"github.com/cyberis/chirpy/internal/auth"
	"github.com/cyberis/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type chirpRequest struct {
	Body   string        `json:"body"`
	UserID uuid.NullUUID `json:"user_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type userRequest struct {
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	const filepathRoot = "."
	const port = "8080"
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Access database URL from environment variables
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Println("DB_URL not set in environment variables")
	} else {
		log.Printf("Database URL: %s", dbURL)
	}

	// Access platform from environment variables
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Println("PLATFORM not set in environment variables")
	} else {
		log.Printf("Running on platform: %s", platform)
	}

	// Open database connection here if needed
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Set up HTTP server and routes

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      database.New(db),
		platform:       platform,
	}

	// Handle root path with a FileServer to provide /index.html
	fileServer := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	//Handle a health check endpoint
	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	// Handle a metrics endpoint
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	// Handle database user creation endpoint
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	// Handle Login endpoint
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	// Handle database chirp creation endpoint
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)

	// Handle Get All Chirps endpoint
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)

	// Handle Get Chirp by ID endpoint
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}

// Middleware to increment file server hits
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// API Handlers

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>
		`, cfg.fileserverHits.Load())))
}

// Handle database user creation endpoint
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user email")
		return
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(req.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	req.HashedPassword = hashedPassword

	// Create user in the database
	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: req.HashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Respond with created user details
	respondWithJSON(w, http.StatusCreated, user)
}

// Handle Login endpoint
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid login data")
		return
	}

	// Retrieve user by email
	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve user")
		}
		return
	}

	// Compare provided password with stored hashed password
	match, err := auth.CheckPasswordHash(req.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to verify password")
		return
	}
	if !match {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Respond with user details on successful login
	respondWithJSON(w, http.StatusOK, database.CreateUserRow{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	var req chirpRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp data")
		return
	}

	// Validate and clean the chirp body
	cleanedBody, err := validateChirp(req.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Body = cleanedBody

	// Create chirp in the database
	chirpParams := database.CreateChirpParams{
		Body:   req.Body,
		UserID: req.UserID,
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	// Respond with created chirp details
	respondWithJSON(w, http.StatusCreated, chirp)
}

// Handle Get All Chirps endpoint

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirps")
		return
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

// Handle Get Chirp by ID endpoint
func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
		return
	}

	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirp")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Reset endpoint is only available in dev environment")
		return
	}

	err := cfg.dbQueries.ResetUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset users")
		return
	}
	cfg.fileserverHits.Store(0)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Hits reset to 0, users table cleared"})
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

// Helper functions
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func validateChirp(body string) (string, error) {
	if body == "" {
		return "", fmt.Errorf("Blank chirps are not allowed")
	}
	if len(body) > 140 {
		return "", fmt.Errorf("Chirp is too long")
	}
	return cleanBody(body), nil
}

func cleanBody(body string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(body, " ")
	for i, word := range words {
		for _, badWord := range badWords {
			if strings.ToLower(word) == badWord {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}
