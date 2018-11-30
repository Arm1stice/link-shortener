package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"gopkg.in/boj/redistore.v1"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
	"github.com/wcalandro/base62"
)

// Messages when shortening
type shortenMessage struct {
	ErrorMessages   []interface{}
	SuccessMessages []interface{}
}

func websiteRouter(store *redistore.RediStore) chi.Router {
	// Create a packr box
	box := packr.NewBox("./views")
	indexHTMLString := box.String("index.html")
	// Implement the templates
	indexTemplate := template.Must(template.New("index.html").Parse(indexHTMLString))

	// MySQL database
	db := DB

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// Get a session.
		session, err := store.Get(r, "session")
		if err != nil {
			log.Println("ERROR GETTING SESSION: ", err.Error())
		}

		indexMessage := &shortenMessage{}
		indexMessage.ErrorMessages = session.Flashes("shorten_error")
		indexMessage.SuccessMessages = session.Flashes("shorten_success")

		session.Save(r, w)

		indexTemplate.Execute(w, indexMessage)
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
		parsedID, err := base62.FromB62(linkID)

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
			log.Println("Failed to select link, it probably doesn't exist")
			fmt.Fprintf(w, "That link doesn't exist")
			return
		}
		fmt.Fprintf(w, "Link: "+link+" | Views: "+strconv.FormatInt(views, 10))
	})

	r.Post("/createShortURL", func(w http.ResponseWriter, r *http.Request) {
		// Get a session.
		session, err := store.Get(r, "session")
		if err != nil {
			log.Println("ERROR GETTING SESSION: ", err.Error())
		}

		// First, we parse the form
		r.ParseForm()

		// Get the value of URL from the form
		userURL := r.FormValue("url")

		// If URL value wasn't passed or is blank, redirect to noURL error message
		if len(userURL) == 0 {
			session.AddFlash("URL field cannot be empty", "shorten_error")
			session.Save(r, w)
			http.Redirect(w, r, "/", 302)
		} else {
			// Create link insertion statement
			linkInsertionStatement, err := db.Prepare("INSERT INTO links (url, views) VALUES (?, 0)")
			if err != nil {
				log.Println("Failed to prepare linkInsertionStatement")
				panic(err)
			}
			defer linkInsertionStatement.Close()
			// Check if the URL is valid, if not, then redirect to invalidURL message
			if isValidURL := govalidator.IsURL(userURL); isValidURL {
				// If it's valid, we want to make sure it has a url scheme attached to it
				var parsedURL *url.URL
				parsedURL, err = url.Parse(userURL)
				if err != nil {
					session.AddFlash("An error occurred while parsing the URL", "shorten_error")
					session.Save(r, w)
					http.Redirect(w, r, "/", 302)
					return
				}

				// Check if the parsed URL has a scheme and if not, add one
				if parsedURL.Scheme == "" {
					parsedURL.Scheme = "http"
				}
				userURL = parsedURL.String()

				// Execute prepared statement
				result, err := linkInsertionStatement.Exec(userURL)
				if err != nil {
					session.AddFlash("An error occurred while trying to insert URL into the database", "shorten_error")
					session.Save(r, w)
					http.Redirect(w, r, "/", 302)
					return
				}

				// Get id of the inserted link
				insertedID, err := result.LastInsertId()
				if err != nil {
					session.AddFlash("An error occurred while tryig to parse URL", "shorten_error")
					session.Save(r, w)
					http.Redirect(w, r, "/", 302)
					return
				}

				// Convert the id to base36 and redirect successfully
				base62Id := base62.ToB62(insertedID)
				session.AddFlash(base62Id, "shorten_success")
				session.Save(r, w)
				http.Redirect(w, r, "/", 302)
			} else {
				session.AddFlash("Invalid URL", "shorten_error")
				session.Save(r, w)
				http.Redirect(w, r, "/", 302)
			}
		}
	})

	return r
}
