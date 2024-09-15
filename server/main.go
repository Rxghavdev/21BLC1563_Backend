package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux" // Use Gorilla Mux for routing with parameters
    "trademarkia/internal/db"
    "trademarkia/internal/handlers"
    "trademarkia/internal/middlewares"
    "trademarkia/middleware"
    "trademarkia/internal/background" // Import the background package
)

func main() {
    // Initialize the database connection
    err := db.InitDB()
    if err != nil {
        log.Fatal("Error connecting to the database: ", err)
    }

    // Start the background worker for deleting expired files
    background.StartFileDeletionWorker()

    // Create a new router using Gorilla Mux
    router := mux.NewRouter()

    // Define public routes
    router.HandleFunc("/register", handlers.RegisterUser).Methods("POST")
    router.HandleFunc("/login", handlers.Login).Methods("POST")

    // Protected routes with JWT middleware
    router.Handle("/upload", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileUpload))).Methods("POST")
    router.Handle("/search", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileSearch))).Methods("GET")
    router.Handle("/files", middlewares.JWTMiddleware(http.HandlerFunc(handlers.GetFiles))).Methods("GET")
    router.Handle("/share/{file_id}", middlewares.JWTMiddleware(http.HandlerFunc(handlers.ShareFile))).Methods("GET")
    router.Handle("/file/update/{file_id}", middlewares.JWTMiddleware(http.HandlerFunc(handlers.UpdateFileMetadata))).Methods("POST")

    // Start the server
    log.Println("Server is running on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", router))
}
