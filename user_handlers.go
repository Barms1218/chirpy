package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	Email     string    `json:"email"`
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
