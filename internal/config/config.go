package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/CzarRamos/chirpy/internal/auth"
	"github.com/CzarRamos/chirpy/internal/chirp"
	"github.com/CzarRamos/chirpy/internal/database"
	"github.com/CzarRamos/chirpy/internal/events"
	"github.com/google/uuid"
)

var SORT_ASC_KEYWORD = "asc"
var SORT_DESC_KEYWORD = "desc"

type ApiConfig struct {
	FileserverHits atomic.Int32
	DbQueries      *database.Queries
	SecretToken    string
	PolkaKey       string
}

func (config *ApiConfig) HandlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	// reset user list
	config.DbQueries.RemoveAllUsers(r.Context())

	// reset hit metrics
	config.FileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func (config *ApiConfig) HandlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (config *ApiConfig) HandlerMetrics(w http.ResponseWriter, r *http.Request) {
	hits := config.FileserverHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	output := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, hits)

	w.Write([]byte(output))
}

func (config *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (config *ApiConfig) CreateNewUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := chirp.UserCredentials{}
	// correct info will be stored in params
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	newUser, err := config.DbQueries.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		HashedPassword: hashedPassword,
		UpdatedAt:      time.Now(),
		Email:          params.Email,
	})

	userInfo := chirp.User{
		ID:        newUser.ID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	}

	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	data, err := json.Marshal(userInfo)
	if err != nil {
		log.Printf("error marshalling newly created user: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	w.Write(data)
}

func (config *ApiConfig) NewChirpHandler(w http.ResponseWriter, r *http.Request) {

	output, err := auth.GetTokenBearer(r.Header)
	if err != nil {
		log.Printf("error getting token bearer: %s", err)
		w.WriteHeader(500)
		return
	}

	log.Printf("TOKEN BEARER: %s", output)

	userID, err := auth.ValidateJWT(output, config.SecretToken)
	if err != nil {
		log.Printf("error validating new chirp token: %s", err)
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := chirp.ShortChirp{}
	// correct info will be stored in params
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	isValid := isChirpValid(params)

	if !isValid {
		w.WriteHeader(400)
		w.Write(newChirpError("Chirp is too long"))
		return
	}

	filteredChirp, err := filterChirp(params)
	if err != nil {
		log.Printf("error filtering chirp: %s", err)
		return
	}

	newChirp, err := config.DbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		ID:        uuid.New(),
		UpdatedAt: time.Now(),
		Body:      filteredChirp.Message,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("error adding chirp: %s", err)
		w.WriteHeader(500)
		return
	}

	chirpRes := chirp.ShortChirp{
		ID:      newChirp.ID,
		Message: newChirp.Body,
		UserID:  newChirp.UserID,
	}
	w.WriteHeader(201)
	w.Write(newShortChirpData(chirpRes))

}

func isChirpValid(chirp chirp.ShortChirp) bool {
	// valid if message is 140 characters or less
	return len(chirp.Message) <= 140
}

func filterChirp(userChirp chirp.ShortChirp) (chirp.ShortChirp, error) {

	var profaneWords [3]string
	profaneWords[0] = "kerfuffle"
	profaneWords[1] = "sharbert"
	profaneWords[2] = "fornax"

	hasProfaneWord := false
	for _, profaneWord := range profaneWords {
		if !hasProfaneWord && (strings.Contains(strings.ToLower(userChirp.Message), strings.ToLower(profaneWord)) ||
			strings.Contains(strings.ToUpper(userChirp.Message), strings.ToUpper(profaneWord))) {
			hasProfaneWord = true
			break
		}
	}

	if !hasProfaneWord {
		return userChirp, nil
	}

	splitMessage := strings.Split(userChirp.Message, " ")

	for idx, word := range splitMessage {
		for _, profaneWord := range profaneWords {
			if strings.EqualFold(word, profaneWord) {
				splitMessage[idx] = "****"
			}
		}
	}

	modifiedChirp := chirp.ShortChirp{
		Message: strings.Join(splitMessage, " "),
	}

	return modifiedChirp, nil
}

func newChirpError(errorMessage string) []byte {

	newChirpError := chirp.ChirpError{
		ErrorMessage: errorMessage,
	}

	data, err := json.Marshal(newChirpError)
	if err != nil {
		log.Printf("error marshalling error message: %s", err)
		return nil
	}

	return data
}

func newShortChirpData(userChirp chirp.ShortChirp) []byte {

	data, err := json.Marshal(userChirp)
	if err != nil {
		log.Printf("error marshalling chirp validity: %s", err)
		return nil
	}

	return data
}

func (config *ApiConfig) GetAllChirpsHandler(w http.ResponseWriter, r *http.Request) {

	authorID := r.URL.Query().Get("author_id")
	customSort := r.URL.Query().Get("sort")

	foundChirps := make([]chirp.DetailedChirp, 0)

	if len(authorID) > 0 {
		// authorId exists
		var err error
		authorUUID := uuid.Must(uuid.MustParse(authorID), err)
		if err != nil {
			log.Printf("error parsing authorID to UUID: %s", err)
			w.WriteHeader(500)
			return
		}

		allUserChirps, err := config.DbQueries.GetAllChirpsOfUserID(r.Context(), authorUUID)
		if err != nil {
			log.Printf("error getting user's chirps: %s", err)
			w.WriteHeader(500)
			return
		}

		for _, chirpRow := range allUserChirps {
			foundChirp := chirp.DetailedChirp{
				ID:        chirpRow.ID,
				CreatedAt: chirpRow.CreatedAt,
				UpdatedAt: chirpRow.UpdatedAt,
				Body:      chirpRow.Body,
				UserID:    chirpRow.UserID,
			}

			foundChirps = append(foundChirps, foundChirp)
		}
	} else {
		// authorId does not exist, show all chirps
		allChirps, err := config.DbQueries.GetAllChirpsSinceCreation(r.Context())
		if err != nil {
			log.Printf("error getting all chirps: %s", err)
			w.WriteHeader(500)
			return
		}
		for _, chirpRow := range allChirps {
			foundChirp := chirp.DetailedChirp{
				ID:        chirpRow.ID,
				CreatedAt: chirpRow.CreatedAt,
				UpdatedAt: chirpRow.UpdatedAt,
				Body:      chirpRow.Body,
				UserID:    chirpRow.UserID,
			}

			foundChirps = append(foundChirps, foundChirp)
		}
	}

	// FoundChirps starts off sorted from oldest to latest

	// this sorts from latest to oldest
	if customSort == SORT_DESC_KEYWORD {
		sort.Slice(foundChirps, func(i, j int) bool { return foundChirps[i].CreatedAt.After(foundChirps[j].CreatedAt) })
	}

	data, err := json.Marshal(foundChirps)
	if err != nil {
		log.Printf("error marshalling chirp validity: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Write(data)
	w.WriteHeader(200)
}

func (config *ApiConfig) GetChirpViaIdHandler(w http.ResponseWriter, r *http.Request) {

	chirpId := r.PathValue("chirp_id")

	var err error
	chirpUUID := uuid.Must(uuid.MustParse(chirpId), err)

	foundChirp, err := config.DbQueries.GetChirpViaID(r.Context(), chirpUUID)
	if err != nil {
		log.Printf("error chirp does not exist: %s", err)
		w.WriteHeader(404)
		return
	}

	output := chirp.DetailedChirp{
		ID:        foundChirp.ID,
		CreatedAt: foundChirp.CreatedAt,
		UpdatedAt: foundChirp.UpdatedAt,
		Body:      foundChirp.Body,
		UserID:    foundChirp.UserID,
	}

	data, err := json.Marshal(output)
	if err != nil {
		log.Printf("error marshalling found chirp: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Write(data)
	w.WriteHeader(200)
}

func (config *ApiConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := chirp.UserCredentials{}
	// correct info will be stored in params
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding chirp login data: %s", err)
		w.WriteHeader(500)
		return
	}

	foundUser, err := config.DbQueries.GetUserViaEmail(r.Context(), params.Email)
	if err != nil {
		log.Printf("Incorrect email or password: %s", err)
		w.WriteHeader(401)
		return
	}

	err = auth.CheckPasswordHash(params.Password, foundUser.HashedPassword)
	if err != nil {
		log.Printf("Incorrect email or password: %s", err)
		w.WriteHeader(401)
		return
	}

	newAccessToken, err := auth.MakeJWT(foundUser.ID, config.SecretToken)
	if err != nil {
		log.Printf("error creating token: %s", err)
		w.WriteHeader(500)
		return
	}

	newRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("error creating refresh token: %s", err)
		w.WriteHeader(500)
		return
	}

	// add new refresh token to Db
	_, err = config.DbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     newRefreshToken.Token,
		UpdatedAt: time.Now(),
		ExpiresAt: newRefreshToken.ExpiresAt,
		UserID:    foundUser.ID,
	})
	if err != nil {
		log.Printf("error adding refresh tokento db: %s", err)
		w.WriteHeader(500)
		return
	}

	output := chirp.User{
		ID:           foundUser.ID,
		CreatedAt:    foundUser.CreatedAt,
		UpdatedAt:    foundUser.UpdatedAt,
		Email:        foundUser.Email,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken.Token,
		IsChirpyRed:  foundUser.IsChirpyRed.Bool,
	}

	data, err := json.Marshal(output)
	if err != nil {
		log.Printf("error marshalling returning user info: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Write([]byte(data))
	w.WriteHeader(200)
}

func (config *ApiConfig) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetTokenBearer(r.Header)
	if err != nil {
		log.Printf("error getting token bearer info: %s", err)
		w.WriteHeader(500)
		return
	}

	// check if token exists
	foundRefreshToken, err := config.DbQueries.GetUserViaRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("error token is not valid: %s", err)
		w.WriteHeader(401)
		w.Write([]byte("error: token is not valid"))
		return
	}

	// if token has already expired or has ever been revoked
	if foundRefreshToken.ExpiresAt.Compare(time.Now()) <= 0 || foundRefreshToken.RevokedAt.Valid {
		log.Printf("error token has expired")
		w.WriteHeader(401)
		w.Write([]byte("error: token is not valid"))
		return
	}

	newJWTToken, err := auth.MakeJWT(foundRefreshToken.UserID, config.SecretToken)
	if err != nil {
		log.Printf("error creating JWT token: %s", err)
		w.WriteHeader(401)
		return
	}

	output := auth.NewAccessToken{
		Token: newJWTToken,
	}

	data, err := json.Marshal(output)
	if err != nil {
		log.Printf("error marshalling newly created access token: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Write(data)
	w.WriteHeader(200)

}

func (config *ApiConfig) RevokeRefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// get the token provided
	refreshToken, err := auth.GetTokenBearer(r.Header)
	if err != nil {
		log.Printf("error getting token bearer info: %s", err)
		w.WriteHeader(500)
		return
	}

	// check if token exists
	foundRefreshToken, err := config.DbQueries.GetUserViaRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("error token is not valid: %s", err)
		w.WriteHeader(401)
		w.Write([]byte("error: token is not valid"))
		return
	}

	// proceed with revoking access
	err = config.DbQueries.SetRefreshTokenRevoked(r.Context(), database.SetRefreshTokenRevokedParams{
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		UpdatedAt: time.Now(),
		Token:     foundRefreshToken.Token,
	})
	if err != nil {
		log.Printf("error revoking refresh token provided: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
}

func (config *ApiConfig) UpdateCredentialsHandler(w http.ResponseWriter, r *http.Request) {

	// grab user access token
	accessToken, err := auth.GetTokenBearer(r.Header)
	if err != nil {
		log.Printf("error getting token bearer info: %s", err)
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(accessToken, config.SecretToken)
	if err != nil {
		log.Printf("error token not valid: %s", err)
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := chirp.UserCredentials{}
	// correct info will be stored in params
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	newPasswordHash, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	err = config.DbQueries.UpdateUserCredentials(r.Context(), database.UpdateUserCredentialsParams{
		Email:          params.Email,
		HashedPassword: newPasswordHash,
		ID:             userID,
	})
	if err != nil {
		log.Printf("error updating user email and password: %s", err)
		w.WriteHeader(500)
		return
	}

	newUserCredentials := chirp.UserCredentials{
		Email: params.Email,
	}

	data, err := json.Marshal(newUserCredentials)
	if err != nil {
		log.Printf("error marshalling found chirp: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Write(data)
	w.WriteHeader(200)
}

func (config *ApiConfig) DeleteChirpHandler(w http.ResponseWriter, r *http.Request) {
	// grab user access token
	var err error

	accessToken, err := auth.GetTokenBearer(r.Header)
	if err != nil {
		log.Printf("error getting token bearer info: %s", err)
		w.WriteHeader(401)
		return
	}

	chirpId := r.PathValue("chirp_id")

	chirpUUID := uuid.Must(uuid.MustParse(chirpId), err)
	foundChirp, err := config.DbQueries.GetChirpViaID(r.Context(), chirpUUID)
	if err != nil {
		log.Printf("error chirp does not exist: %s", err)
		w.WriteHeader(404)
		return
	}

	userID, err := auth.ValidateJWT(accessToken, config.SecretToken)
	if err != nil {
		log.Printf("error token not valid: %s", err)
		w.WriteHeader(401)
		return
	}

	// if the user is not the author of the chirp
	if foundChirp.UserID != userID {
		log.Printf("error forbidden access")
		w.WriteHeader(403)
		return
	}

	err = config.DbQueries.DeleteChirpPerm(r.Context(), foundChirp.ID)
	if foundChirp.UserID != userID {
		log.Printf("error deleting chirp from db: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)

}

func (config *ApiConfig) UpgradeUserHandler(w http.ResponseWriter, r *http.Request) {

	providedApiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		log.Printf("error getting api key info: %s", err)
		w.WriteHeader(401)
		return
	}

	if providedApiKey != config.PolkaKey {
		log.Printf("error invalid key: %s", err)
		w.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := auth.ChirpyEvent{}
	// correct info will be stored in params
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	// we only want the upgrade events. Everything else is ignored
	if params.Event != events.UpgradeUserEvent {
		log.Printf("error unrecognized event: %s", err)
		w.WriteHeader(204)
		return
	}

	err = config.DbQueries.UpgradeToChirpyRedViaID(r.Context(), params.Data.ID)
	if err != nil {
		log.Printf("error upgrading user to chirpy red: %s", err)
		w.WriteHeader(404)
		return
	}

	userInfo := chirp.User{
		IsChirpyRed: true,
	}

	data, err := json.Marshal(userInfo)
	if err != nil {
		log.Printf("error marshalling newly created access token: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(204)
	w.Write(data)
}
