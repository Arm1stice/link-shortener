package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gobuffalo/packr"
	"github.com/joho/godotenv"
)

var (
	mysql *sql.DB
)

func main() {
	// Read the .env file and parse it into the local environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file to load")
	} else {
		log.Println("Successfully loaded .env file")
	}

	// Set up MySQL
	log.Println("Connecting to MySQL Database")
	address := os.Getenv("MYSQL_URI")
	db, err := sql.Open("mysql", address)
	if err != nil {
		log.Fatal("Couldn't connect to the MySQL database")
		panic(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging MySQL database")
		panic(err)
	}
	log.Println("Successfully connected to MySQL database")
	defer db.Close()

	// Initialize the main router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Create a packr box
	box := packr.NewBox("./views")
	indexHTMLString := box.String("index.html")
	// Implement the templates
	indexTemplate := template.Must(template.New("index.html").Parse(indexHTMLString))

	// Basic get on the index
	type IndexMessage struct {
		ErrorMessage   string
		SuccessMessage string
	}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := r.URL.Query().Get("error")
		if err != "" {
			indexMessage := &IndexMessage{}
			switch err {
			case "noURL":
				indexMessage.ErrorMessage = "You must enter a URL"
			case "invalidURL":
				indexMessage.ErrorMessage = "Invalid URL"
			case "insError":
				indexMessage.ErrorMessage = "Error occurring inserting the link into the database"
			case "getIdError":
				indexMessage.ErrorMessage = "An error occurred getting the id of the link"
			default:
				indexMessage.ErrorMessage = "An error occurred"
			}
			indexTemplate.Execute(w, indexMessage)
			return
		}

		link := r.URL.Query().Get("link")
		if link != "" {
			indexMessage := &IndexMessage{SuccessMessage: link}
			indexTemplate.Execute(w, indexMessage)
			return
		}
		indexTemplate.Execute(w, nil)
	})

	// Link stats
	r.Get("/stats/{linkID}", func(w http.ResponseWriter, r *http.Request) {
		// Create prepared statements
		selectStatement, err := db.Prepare("SELECT * from links WHERE id = ?")
		if err != nil {
			log.Fatal("Failed to prepare selectStatement")
			panic(err)
		}
		defer selectStatement.Close()
		// Get linkID out of URL
		linkID := chi.URLParam(r, "linkID")

		// Convert back to a number
		parsedID, err := strconv.ParseInt(linkID, 36, 64)

		// See if there was an error while converting
		if err != nil {
			fmt.Fprintf(w, "Invalid link ID format")
		}

		// Now get the URL that this links to
		var rowID int64
		var link string
		var views int64
		err = selectStatement.QueryRow(parsedID).Scan(&rowID, &link, &views)
		if err != nil {
			log.Println("Failed to select link")
			log.Fatal(err.Error())
			return
		}
		fmt.Fprintf(w, "Link: "+link+" | Views: "+strconv.FormatInt(views, 10))
	})

	r.Post("/createShortURL", func(w http.ResponseWriter, r *http.Request) {
		// First, we parse the form
		r.ParseForm()

		// Get the value of URL from the form
		url := r.FormValue("url")

		// If URL value wasn't passed or is blank, redirect to noURL error message
		if len(url) == 0 {
			http.Redirect(w, r, "/?error=noURL", 301)
		} else {
			// Create link insertion statement
			linkInsertionStatement, err := db.Prepare("INSERT INTO links (url, views) VALUES (?, 0)")
			if err != nil {
				log.Fatal("Failed to prepare linkInsertionStatement")
				panic(err)
			}
			defer linkInsertionStatement.Close()
			// Check if the URL is valid, if not, then redirect to invalidURL message
			if isValidURL := govalidator.IsURL(url); isValidURL {
				// Execute prepared statement
				result, err := linkInsertionStatement.Exec(url)
				if err != nil {
					http.Redirect(w, r, "/?error=insError", 301)
					return
				}

				// Get id of the inserted link
				insertedID, err := result.LastInsertId()
				if err != nil {
					http.Redirect(w, r, "/?error=getIdError", 301)
					return
				}

				// Convert the id to base36 and redirect successfully
				base36Id := strconv.FormatInt(insertedID, 36)
				http.Redirect(w, r, "/?link="+base36Id, 301)
			} else {
				http.Redirect(w, r, "/?error=invalidURL", 301)
			}
		}
	})

	// Link redirect
	r.Get("/{linkID}", func(w http.ResponseWriter, r *http.Request) {
		// Create prepared statements
		selectStatement, err := db.Prepare("SELECT * from links WHERE id = ?")
		if err != nil {
			log.Fatal("Failed to prepare selectStatement")
			panic(err)
		}
		defer selectStatement.Close()
		updateStatement, err := db.Prepare("UPDATE links SET views=views+1 WHERE id = ?")
		if err != nil {
			log.Fatal("Failed to prepare updateStatement")
			panic(err)
		}
		defer updateStatement.Close()
		// Get linkID out of URL
		linkID := chi.URLParam(r, "linkID")

		// Convert back to a number
		parsedID, err := strconv.ParseInt(linkID, 36, 64)

		// See if there was an error while converting
		if err != nil {
			fmt.Fprintf(w, "Invalid link ID format")
		}

		// Now get the URL that this links to
		var rowID int64
		var link string
		var views int64
		err = selectStatement.QueryRow(parsedID).Scan(&rowID, &link, &views)
		if err != nil {
			log.Println("Failed to select link")
			log.Fatal(err.Error())
			return
		}

		result, err3 := updateStatement.Exec(parsedID)
		if err3 != nil {
			log.Println("Failed to update views")
			log.Fatal(err.Error())
			return
		}
		rowsAffected, err4 := result.RowsAffected()
		if err4 != nil {
			log.Println("Failed to get rows affected")
			log.Fatal(err.Error())
			return
		}
		if rowsAffected != 1 {
			log.Println("Rows affected wasn't 1, it was ", rowsAffected)
			fmt.Fprintf(w, "Error")
			return
		}
		http.Redirect(w, r, link, 302)
	})

	// Handle all 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "This page was unable to be found")
	})

	// Listen and serve the web server
	port := ":8000"
	if value, ok := os.LookupEnv("PORT"); ok {
		port = ":" + value
	}
	log.Println("Running on port " + port)
	http.ListenAndServe(port, r)
}
