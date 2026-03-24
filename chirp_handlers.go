package main

import (
	"encoding/json"
	"fmt"
	"github.com/chirpy/internal/database"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
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

	filteredString := FilterChirp(params.Body)

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

func (a *apiConfig) GetChirpHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("chirpID")

	id, err := uuid.Parse(idString)
	if err != nil {
		a.RespondWithError(w, 400, fmt.Sprintf("Invalid ID: %v", err))
		return
	}

	chirp, err := a.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		a.RespondWithError(w, 404, fmt.Sprintf("Chirp not found: %v", err))
		return
	}

	a.RespondWithJson(w, 200, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func (a *apiConfig) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	var chirps []Chirp

	dbChirps, err := a.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		a.RespondWithError(w, 500, fmt.Sprintf("Failed to get chirps from database: %v", err))
	}

	for _, chirp := range dbChirps {
		chirp.Body = FilterChirp(chirp.Body)
		chirps = append(chirps, Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	a.RespondWithJson(w, 200, chirps)
}

func FilterChirp(chirp string) string {
	messageArray := strings.Split(chirp, " ")

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

	return strings.Join(messageArray, " ")
}
