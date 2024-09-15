
package db

import (
    "database/sql"
    "log"
    _ "github.com/lib/pq" 
)

var DB *sql.DB

func InitDB() error {
    return initPostgresDB("user=postgres password=Trademark dbname=trademark sslmode=disable")
}

func InitTestDB(connStr string) error {
    return initPostgresDB(connStr)
}

func initPostgresDB(connStr string) error {
    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return err
    }

    err = DB.Ping()
    if err != nil {
        return err
    }

    log.Println("Database connection established")
    return nil
}
