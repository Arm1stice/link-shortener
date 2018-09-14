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
	charToNum['0'] = 1
	charToNum['1'] = 2
	charToNum['2'] = 3
	charToNum['3'] = 4
	charToNum['4'] = 5
	charToNum['5'] = 6
	charToNum['6'] = 7
	charToNum['7'] = 8
	charToNum['8'] = 9
	charToNum['9'] = 10
	charToNum['a'] = 11
	charToNum['b'] = 12
	charToNum['c'] = 13
	charToNum['d'] = 14
	charToNum['e'] = 15
	charToNum['f'] = 16
	charToNum['g'] = 17
	charToNum['h'] = 18
	charToNum['i'] = 19
	charToNum['j'] = 20
	charToNum['k'] = 21
	charToNum['l'] = 22
	charToNum['m'] = 23
	charToNum['n'] = 24
	charToNum['o'] = 25
	charToNum['p'] = 26
	charToNum['q'] = 27
	charToNum['r'] = 28
	charToNum['s'] = 29
	charToNum['t'] = 30
	charToNum['u'] = 31
	charToNum['v'] = 32
	charToNum['w'] = 33
	charToNum['x'] = 34
	charToNum['y'] = 35
	charToNum['z'] = 36
	charToNum['A'] = 37
	charToNum['B'] = 38
	charToNum['C'] = 39
	charToNum['D'] = 40
	charToNum['E'] = 41
	charToNum['F'] = 42
	charToNum['G'] = 43
	charToNum['H'] = 44
	charToNum['I'] = 45
	charToNum['J'] = 46
	charToNum['K'] = 47
	charToNum['L'] = 48
	charToNum['M'] = 49
	charToNum['N'] = 50
	charToNum['O'] = 51
	charToNum['P'] = 52
	charToNum['Q'] = 53
	charToNum['R'] = 54
	charToNum['S'] = 55
	charToNum['T'] = 56
	charToNum['U'] = 57
	charToNum['V'] = 58
	charToNum['W'] = 59
	charToNum['X'] = 60
	charToNum['Y'] = 61
	charToNum['Z'] = 62
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
