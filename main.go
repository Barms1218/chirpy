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
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
}

func (a *apiConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

func (a *apiConfig) AddChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		a.RespondWithError(w, 404, fmt.Sprintf("Bad Request: %v", err))
	}
	messageArray := strings.Split(params.Body, " ")

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

	newChirp, err := a.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{Body: params.Body, UserID: params.UserID})
	chirp := Chirp{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      filteredString,
		UserID:    newChirp.UserID,
	}

	if err != nil {
		a.RespondWithError(w, 500, fmt.Sprintf("Internal Error: %v", err))
	}

	a.RespondWithJson(w, 201, chirp)
}

func (a *apiConfig) AddUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		a.RespondWithError(w, 404, fmt.Sprint("Must be an email address: %v", err))
	}

	newUser, err := a.dbQueries.CreateUser(r.Context(), params.Email)
	if err != nil {
		a.RespondWithError(w, 403, fmt.Sprintf("Error creating user: %v", err))
		return
	}

	user := User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	a.RespondWithJson(w, 201, user)
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
	if os.Getenv("PLATFORM") != "dev" {
		a.RespondWithError(w, 403, "Forbidden")
	}
	a.fileServerHits.Store(0)
	_, err := a.dbQueries.DeleteUsers(r.Context())
	if err != nil {
		a.RespondWithError(w, 500, "Could not delete users from database.")
	}
}

func (a *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
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
	mux.HandleFunc("POST /api/users", apiCfg.AddUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.AddChirpHandler)
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
