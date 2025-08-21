package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/CzarRamos/chirpy/internal/config"
	"github.com/CzarRamos/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const OK_STATUS_CODE = 200
const ERROR_STATUS_CODE = 400

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("error unable to open %s: %s", dbURL, err)
		return
	}

	officialSecretToken := os.Getenv("secret")
	OfficialPolkaKey := os.Getenv("POLKA_KEY")

	dbQueries := database.New(db)

	userConfig := config.ApiConfig{
		FileserverHits: atomic.Int32{},
		DbQueries:      dbQueries,
		SecretToken:    officialSecretToken,
		PolkaKey:       OfficialPolkaKey,
	}

	serverMux := http.NewServeMux()

	homepageHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serverMux.Handle("/app/", userConfig.MiddlewareMetricsInc(homepageHandler))          // shows the home page
	serverMux.HandleFunc("GET /admin/metrics", userConfig.HandlerMetrics)                // shows number of visitors to home page
	serverMux.HandleFunc("POST /admin/reset", userConfig.HandlerResetMetrics)            // reset all metrics to zero
	serverMux.HandleFunc("GET /api/healthz", userConfig.HandlerHealthz)                  // helps check if website is running
	serverMux.HandleFunc("POST /api/users", userConfig.CreateNewUserHandler)             // registers a new user
	serverMux.HandleFunc("PUT /api/users", userConfig.UpdateCredentialsHandler)          // lets user update their email and password
	serverMux.HandleFunc("GET /api/chirps", userConfig.GetAllChirpsHandler)              // shows all chirps
	serverMux.HandleFunc("GET /api/chirps/{chirp_id}", userConfig.GetChirpViaIdHandler)  // lets user find chirps
	serverMux.HandleFunc("DELETE /api/chirps/{chirp_id}", userConfig.DeleteChirpHandler) // lets user delete chirps
	serverMux.HandleFunc("POST /api/chirps", userConfig.NewChirpHandler)                 // lets user creates new chirps
	serverMux.HandleFunc("POST /api/login", userConfig.LoginHandler)                     // lets the user log in
	serverMux.HandleFunc("POST /api/refresh", userConfig.RefreshHandler)                 // gives user access token with valid refresh token
	serverMux.HandleFunc("POST /api/revoke", userConfig.RevokeRefreshTokenHandler)       // remove access to refresh token
	serverMux.HandleFunc("POST /api/polka/webhooks", userConfig.UpgradeUserHandler)      // upgrades user to chirpy red

	server := http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	server.ListenAndServe()
}
