package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func newRootHandler(db *sql.DB, shortHost string) http.Handler {
	r := websiteRouter(db, shortHost)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	r.Mount("/", shortenerRouter(db))
	return r
}

func main() {
	// Read the .env file and parse it into the local environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file to load")
	} else {
		log.Println("Successfully loaded .env file")
	}

	// Initialize the database
	initDatabase()
	defer DB.Close()

	// Initialize the main router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	shortURL := os.Getenv("SHORT_URL")
	r.Mount("/", newRootHandler(DB, shortURL))

	// Handle all 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "This page was unable to be found")
	})

	// Listen and serve the web server
	port := ":5000"
	if value, ok := os.LookupEnv("PORT"); ok {
		port = ":" + value
	}
	log.Println("Running on port " + port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
