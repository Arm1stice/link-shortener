package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	redistore "gopkg.in/boj/redistore.v1"
)

func newRootHandler(shortHost string, shortHandler, websiteHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
			return
		}

		host := r.Host
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
		if strings.EqualFold(host, shortHost) {
			shortHandler.ServeHTTP(w, r)
			return
		}

		websiteHandler.ServeHTTP(w, r)
	})
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

	shortURL := os.Getenv("SHORT_URL")
	r.Mount("/", newRootHandler(shortURL, shortenerRouter(store), websiteRouter(store)))

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
