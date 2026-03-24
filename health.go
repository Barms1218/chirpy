package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func (a *apiConfig) RespondWithError(w http.ResponseWriter, code int, message string) {
	if code >= 400 && code < 500 {
		a.RespondWithJson(w, code, message)
	} else if code >= 500 {
		a.RespondWithJson(w, code, message)
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
