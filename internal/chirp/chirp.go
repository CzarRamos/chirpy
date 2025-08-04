package chirp

type Chirp struct {
	Message string `json:"body"`
}

type ChirpError struct {
	ErrorMessage string `json:"error"`
}

type ChirpValidated struct {
	IsValid      bool   `json:"valid"`
	CleanMessage string `json:"cleaned_body"`
}
