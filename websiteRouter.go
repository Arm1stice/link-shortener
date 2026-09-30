package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"github.com/wcalandro/base62"
)

//go:embed views/index.html
var indexHTML string

func loadIndexTemplate() (*template.Template, error) {
	return template.New("index.html").Parse(indexHTML)
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL field cannot be empty")
	}
	if !govalidator.IsURL(rawURL) {
		return "", fmt.Errorf("Invalid URL")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("Invalid URL")
	}
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}
	return parsedURL.String(), nil
}

func buildShortURL(shortHost, code string) string {
	shortURL := &url.URL{Scheme: "https", Host: shortHost}
	shortURL.Path = path.Join(shortURL.Path, code)
	return shortURL.String()
}

func websiteRouter(db *sql.DB, shortHost string) chi.Router {
	indexTemplate := template.Must(loadIndexTemplate())

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := indexTemplate.Execute(w, nil); err != nil {
			log.Printf("failed to render index: %v", err)
		}
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
		fmt.Fprintf(w, "Link: %s | Views: %s", link, strconv.FormatInt(views, 10))
	})

	r.Post("/api/links", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var request createLinkRequest
		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Request body must be valid JSON"})
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Request body must be valid JSON"})
			return
		}

		userURL, err := normalizeURL(request.URL)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		result, err := db.ExecContext(r.Context(), "INSERT INTO links (url, views) VALUES (?, 0)", userURL)
		if err != nil {
			log.Printf("failed to insert URL: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Unable to shorten URL"})
			return
		}

		insertedID, err := result.LastInsertId()
		if err != nil {
			log.Printf("failed to get inserted link ID: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "Unable to shorten URL"})
			return
		}

		code := base62.ToB62(uint64(insertedID))
		writeJSON(w, http.StatusCreated, createLinkResponse{
			Code:     code,
			ShortURL: buildShortURL(shortHost, code),
		})
	})

	return r
}
