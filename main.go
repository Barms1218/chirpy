package main

import (
	"database/sql"
	"github.com/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Email     string    `json:"email"`
}

type apiConfig struct {
	fileServerHits atomic.Int32
	dbQueries      *database.Queries
}

func (a *apiConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {

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
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpHandler)
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
