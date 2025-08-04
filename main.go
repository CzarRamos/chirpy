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

	"github.com/CzarRamos/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const OK_STATUS_CODE = 200
const ERROR_STATUS_CODE = 400

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

type chirp struct {
	Message string `json:"body"`
}

type chirpError struct {
	ErrorMessage string `json:"error"`
}

type chirpValidated struct {
	IsValid      bool   `json:"valid"`
	CleanMessage string `json:"cleaned_body"`
}

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("error unable to open %s: %s", dbURL, err)
		return
	}

	dbQueries := database.New(db)

	config := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
	}

	serverMux := http.NewServeMux()

	homepageHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serverMux.Handle("/app/", config.middlewareMetricsInc(homepageHandler))
	serverMux.HandleFunc("GET /admin/metrics", config.handlerMetrics)
	serverMux.HandleFunc("POST /admin/reset", config.handlerResetMetrics)
	serverMux.HandleFunc("GET /api/healthz", config.handleHealthz)
	serverMux.HandleFunc("POST /api/validate_chirp", ChirpValidatorHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	server.ListenAndServe()
}

func (config *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (config *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	hits := config.fileserverHits.Load()
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

func (config *apiConfig) handlerResetMetrics(w http.ResponseWriter, r *http.Request) {
	config.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func (config *apiConfig) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func NewChirpError(errorMessage string) []byte {

	newChirpError := chirpError{
		ErrorMessage: errorMessage,
	}

	data, err := json.Marshal(newChirpError)
	if err != nil {
		log.Printf("error marshalling error message: %s", err)
		return nil
	}

	return data
}

func NewValidatedChirp(userChirp chirpValidated) []byte {

	data, err := json.Marshal(userChirp)
	if err != nil {
		log.Printf("error marshalling chirp validity: %s", err)
		return nil
	}

	return data
}

func ChirpValidatorHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := chirp{}
	// correct info will be stored in params
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	isValid := IsChirpValid(params)

	if isValid {
		filteredChirp, err := FilterChirp(params)
		if err != nil {
			log.Printf("error filtering chirp: %s", err)
			return
		}

		newChirp := chirpValidated{
			IsValid:      true,
			CleanMessage: filteredChirp.Message,
		}
		w.WriteHeader(200)
		w.Write(NewValidatedChirp(newChirp))
		return
	}

	w.WriteHeader(400)
	w.Write(NewChirpError("Chirp is too long"))
}

func IsChirpValid(chirp chirp) bool {
	// valid if message is 140 characters or less
	return len(chirp.Message) <= 140
}

func FilterChirp(userChirp chirp) (chirp, error) {

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

	modifiedChirp := chirp{
		Message: strings.Join(splitMessage, " "),
	}

	return modifiedChirp, nil
}
