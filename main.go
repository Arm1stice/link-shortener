package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/hostrouter"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/joho/godotenv"
	redistore "gopkg.in/boj/redistore.v1"
)

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

	// Session store
	secretKey := os.Getenv("SESSION_SECRET")
	redisHost := os.Getenv("REDIS_HOST")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	store, err := redistore.NewRediStore(10, "tcp", redisHost, redisPassword, []byte(secretKey))
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Initialize the main router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	hr := hostrouter.New()

	shortURL := os.Getenv("SHORT_URL")
	websiteURL := os.Getenv("WEBSITE_URL")
	hr.Map(shortURL, shortenerRouter(store))

	hr.Map("*", websiteRouter(store))

	r.Mount("/", hr)

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
	http.ListenAndServe(port, context.ClearHandler(r))
}
