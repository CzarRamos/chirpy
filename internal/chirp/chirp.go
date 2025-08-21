package chirp

import (
	"time"

	"github.com/google/uuid"
)

const EXPIRES_IN_SECONDS_DEFAULT_LIMIT = 3600
const EXPIRES_IN_SECONDS_MAX_LIMIT = 3600

type ShortChirp struct {
	ID      uuid.UUID `json:"id"`
	Message string    `json:"body"`
	UserID  uuid.UUID `json:"user_id"`
}

type DetailedChirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type ChirpError struct {
	ErrorMessage string `json:"error"`
}

type ChirpValidated struct {
	IsValid      bool   `json:"valid"`
	CleanMessage string `json:"cleaned_body"`
}

type UserCredentials struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}
