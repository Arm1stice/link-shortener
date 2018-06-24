package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
)

//IndexMessage Basic get on the index
type IndexMessage struct {
	ErrorMessage   string
	SuccessMessage string
}

func websiteRouter() chi.Router {
	// Create a packr box
	box := packr.NewBox("./views")
	indexHTMLString := box.String("index.html")
	// Implement the templates
	indexTemplate := template.Must(template.New("index.html").Parse(indexHTMLString))

	// MySQL database
	db := DB

	r := chi.NewRouter()

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
				log.Println("Failed to prepare linkInsertionStatement")
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

	return r
}
