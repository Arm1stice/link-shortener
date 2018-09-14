package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	redistore "gopkg.in/boj/redistore.v1"
)

var alphabet = [62]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
var charToNum = make(map[rune]int)

func init() {
	charToNum['0'] = 0
	charToNum['1'] = 1
	charToNum['2'] = 2
	charToNum['3'] = 3
	charToNum['4'] = 4
	charToNum['5'] = 5
	charToNum['6'] = 6
	charToNum['7'] = 7
	charToNum['8'] = 8
	charToNum['9'] = 9
	charToNum['a'] = 10
	charToNum['b'] = 11
	charToNum['c'] = 12
	charToNum['d'] = 13
	charToNum['e'] = 14
	charToNum['f'] = 15
	charToNum['g'] = 16
	charToNum['h'] = 17
	charToNum['i'] = 18
	charToNum['j'] = 19
	charToNum['k'] = 20
	charToNum['l'] = 21
	charToNum['m'] = 22
	charToNum['n'] = 23
	charToNum['o'] = 24
	charToNum['p'] = 25
	charToNum['q'] = 26
	charToNum['r'] = 27
	charToNum['s'] = 28
	charToNum['t'] = 29
	charToNum['u'] = 30
	charToNum['v'] = 31
	charToNum['w'] = 32
	charToNum['x'] = 33
	charToNum['y'] = 34
	charToNum['z'] = 35
	charToNum['A'] = 36
	charToNum['B'] = 37
	charToNum['C'] = 38
	charToNum['D'] = 39
	charToNum['E'] = 40
	charToNum['F'] = 41
	charToNum['G'] = 42
	charToNum['H'] = 43
	charToNum['I'] = 44
	charToNum['J'] = 45
	charToNum['K'] = 46
	charToNum['L'] = 47
	charToNum['M'] = 48
	charToNum['N'] = 49
	charToNum['O'] = 50
	charToNum['P'] = 51
	charToNum['Q'] = 52
	charToNum['R'] = 53
	charToNum['S'] = 54
	charToNum['T'] = 55
	charToNum['U'] = 56
	charToNum['V'] = 57
	charToNum['W'] = 58
	charToNum['X'] = 59
	charToNum['Y'] = 60
	charToNum['Z'] = 61
}
func shortenerRouter(store *redistore.RediStore) chi.Router {
	// MySQL database
	db := DB

	r := chi.NewRouter()

	// Redirect to site on root
	websiteURL := os.Getenv("WEBSITE_URL")
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+websiteURL, 302)
	})
	// Link redirect
	r.Get("/{linkID}", func(w http.ResponseWriter, r *http.Request) {
		// Create prepared statements
		selectStatement, err := db.Prepare("SELECT * from links WHERE id = ?")
		if err != nil {
			log.Println("Failed to prepare selectStatement")
			panic(err)
		}
		defer selectStatement.Close()
		updateStatement, err := db.Prepare("UPDATE links SET views=views+1 WHERE id = ?")
		if err != nil {
			log.Println("Failed to prepare updateStatement")
			panic(err)
		}
		defer updateStatement.Close()
		// Get linkID out of URL
		linkID := chi.URLParam(r, "linkID")

		// Convert back to a number
		parsedID, ok := idToInt(linkID)

		// See if there was an error while converting
		if !ok {
			fmt.Fprintf(w, "Invalid link ID format")
		}

		// Now get the URL that this links to
		var rowID int64
		var link string
		var views int64
		err = selectStatement.QueryRow(parsedID).Scan(&rowID, &link, &views)
		if err != nil {
			log.Println("Failed to select link")
			fmt.Fprintf(w, "That link doesn't exist")
			return
		}

		result, err3 := updateStatement.Exec(parsedID)
		if err3 != nil {
			log.Println("Failed to update views")
			http.Redirect(w, r, link, 302)
			return
		}
		rowsAffected, err4 := result.RowsAffected()
		if err4 != nil {
			log.Println("Failed to get rows affected")
			http.Redirect(w, r, link, 302)
			return
		}
		if rowsAffected != 1 {
			log.Println("Rows affected wasn't 1, it was ", rowsAffected)
		}
		http.Redirect(w, r, link, 302)
	})

	return r
}

func idToInt(id string) (int64, bool) {
	id = reverse(id)
	total := int64(0)
	for i, c := range id {
		if x, ok := charToNum[c]; ok {
			total += int64(x) * int64(math.Pow(61.0, float64(i)))
		} else {
			return 0, false
		}
	}
	return total, true
}

func intToID(i int64) string {
	// Find the most significant bit but taking the log of the ID
	if i == 0 {
		return "0"
	}
	str := ""
	mostSignificant := int64(math.Floor(math.Log(float64(i)) / math.Log(float64(61))))
	for x := mostSignificant; x >= 0; x-- {
		bit := int64(math.Pow(61, float64(x)))
		quotient := i / bit
		i = i % bit
		str += alphabet[quotient]
	}
	return str
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
