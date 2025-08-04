package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	config "github.com/CzarRamos/chirpy/internal/config"
	"github.com/CzarRamos/chirpy/internal/database"
	handlers "github.com/CzarRamos/chirpy/internal/handlers"
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
	serverMux.HandleFunc("POST /api/validate_chirp", handlers.ChirpValidatorHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	server.ListenAndServe()
}
