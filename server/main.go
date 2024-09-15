package main

import (
    "log"
    "net/http"
    "trademarkia/internal/db"
    "trademarkia/internal/handlers"
    "trademarkia/internal/middlewares"
)

func main() {
    // Initialize the database connection
    err := db.InitDB()
    if err != nil {
        log.Fatal("Error connecting to the database: ", err)
    }

    // Define routes
    http.HandleFunc("/register", handlers.RegisterUser)
    http.HandleFunc("/login", handlers.Login)

    // Protected routes with JWT middleware
    http.Handle("/upload", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileUpload)))
    http.Handle("/search", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileSearch)))

    // Start the server
    log.Println("Server is running on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
