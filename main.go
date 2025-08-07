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

	dbQueries := database.New(db)

	userConfig := config.ApiConfig{
		FileserverHits: atomic.Int32{},
		DbQueries:      dbQueries,
	}

	serverMux := http.NewServeMux()

	homepageHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	serverMux.Handle("/app/", userConfig.MiddlewareMetricsInc(homepageHandler))

	serverMux.HandleFunc("GET /admin/metrics", userConfig.HandlerMetrics)
	serverMux.HandleFunc("POST /admin/reset", userConfig.HandlerResetMetrics)

	serverMux.HandleFunc("GET /api/healthz", userConfig.HandlerHealthz)

	serverMux.HandleFunc("POST /api/users", userConfig.CreateNewUserHandler)

	serverMux.HandleFunc("GET /api/chirps", userConfig.GetAllChirpsHandler)
	serverMux.HandleFunc("GET /api/chirps/{chirp_id}", userConfig.GetChirpViaIdHandler)
	serverMux.HandleFunc("POST /api/chirps", userConfig.NewChirpHandler)
	serverMux.HandleFunc("POST /api/login", userConfig.LoginHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	server.ListenAndServe()
}
