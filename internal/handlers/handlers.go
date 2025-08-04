package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	chirp "github.com/CzarRamos/chirpy/internal/chirp"
)

func NewChirpError(errorMessage string) []byte {

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

func NewValidatedChirp(userChirp chirp.ChirpValidated) []byte {

	data, err := json.Marshal(userChirp)
	if err != nil {
		log.Printf("error marshalling chirp validity: %s", err)
		return nil
	}

	return data
}

func ChirpValidatorHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := chirp.Chirp{}
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

		newChirp := chirp.ChirpValidated{
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

func IsChirpValid(chirp chirp.Chirp) bool {
	// valid if message is 140 characters or less
	return len(chirp.Message) <= 140
}

func FilterChirp(userChirp chirp.Chirp) (chirp.Chirp, error) {

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

	modifiedChirp := chirp.Chirp{
		Message: strings.Join(splitMessage, " "),
	}

	return modifiedChirp, nil
}
