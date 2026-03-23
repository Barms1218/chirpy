package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"last_updated"`
	Email     string    `json:"email"`
}

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
}

func (a *apiConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

func (a *apiConfig) AddUserHandler(w http.ResponseWriter, r *http.Request) {
	var email string

	if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
		a.RespondWithError(w, 404, "Must be an email address")
	}

	newUser, err := a.dbQueries.CreateUser(r.Context(), email)
	if err != nil {
		a.RespondWithError(w, 500, "Error creating user")
	}

	user := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	a.RespondWithJson(w, 200, user)
}

func (a *apiConfig) RespondWithError(w http.ResponseWriter, code int, message string) {
	if code >= 400 && code < 500 {
		http.Error(w, message, code)
	} else if code >= 500 {
		http.Error(w, "Internal Server Error", code)
	}

	type respStruct struct {
		Message string `json:"error"`
	}

	a.RespondWithJson(w, code, respStruct{Message: message})
}

func (a *apiConfig) RespondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling json respond: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func (a *apiConfig) HitsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
	<body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileServerHits.Load())))
}

func (a *apiConfig) ResetHandler(w http.ResponseWriter, r *http.Request) {
	a.fileServerHits.Store(0)

}

func (a *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})

}

func (a *apiConfig) ValidateChirp(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	val := returnVals{}
	err := decoder.Decode(&val)
	if err != nil || len(val.Body) > 140 {
		a.RespondWithError(w, http.StatusBadRequest, "Error decoding json request")
		return
	}

	messageArray := strings.Split(val.Body, " ")

	replacer := strings.NewReplacer(
		"kerfuffle", "****",
		"sharbert", "****",
		"fornax", "****",
		"Kerfuffle", "****",
		"Sharbert", "****",
		"Fornax", "****",
	)

	for i := range messageArray {
		messageArray[i] = replacer.Replace(messageArray[i])
	}
	filteredString := strings.Join(messageArray, " ")

	type filteredStruct struct {
		Response string `json:"cleaned_body"`
	}
	type validStruct struct {
		Response bool `json:"valid"`
	}

	a.RespondWithJson(w, 200, filteredStruct{
		Response: filteredString,
	})

}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	mux := http.NewServeMux()

	dbQueries := database.New(db)
	var n atomic.Int32
	apiCfg := apiConfig{
		fileServerHits: n,
		dbQueries:      dbQueries,
	}
	fileHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileHandler))
	mux.HandleFunc("GET /admin/metrics", apiCfg.HitsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetHandler)
	mux.HandleFunc("POST /api/users")
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.ValidateChirp)
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	http.ListenAndServe(server.Addr, server.Handler)
}
