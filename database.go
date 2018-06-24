package main

import (
	"database/sql"
	"log"
	"os"
)

//DB MySQL database
var (
	DB *sql.DB
)

// Initialize the MySQL Database
func initDatabase() {
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
	DB = db
}
