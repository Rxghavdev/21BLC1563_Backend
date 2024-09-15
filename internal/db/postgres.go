package db

import (
    "database/sql"
    "log"
    _ "github.com/lib/pq"
)

var DB *sql.DB

// Initialize the PostgreSQL connection
func InitDB() error {
    var err error
    connStr := "user=postgres password=Trademark dbname=trademark sslmode=disable"
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return err
    }

    // Ping the database to ensure the connection is valid
    err = DB.Ping()
    if err != nil {
        return err
    }

    log.Println("Database connection established")
    return nil
}
