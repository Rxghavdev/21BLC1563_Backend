package main

import (
    "log"
    "net/http"
    "github.com/gorilla/mux" 
    "trademarkia/internal/db"
    "trademarkia/internal/handlers"
    "trademarkia/internal/middlewares"
    //"trademarkia/middleware"
    "trademarkia/internal/background" 
)

func main() {
    err := db.InitDB()
    if err != nil {
        log.Fatal("Error connecting to the database: ", err)
    }

    background.StartFileDeletionWorker()

    router := mux.NewRouter()

    router.HandleFunc("/register", handlers.RegisterUser).Methods("POST")
    router.HandleFunc("/login", handlers.Login).Methods("POST")

    router.Handle("/upload", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileUpload))).Methods("POST")
    router.Handle("/search", middlewares.JWTMiddleware(http.HandlerFunc(handlers.HandleFileSearch))).Methods("GET")
    router.Handle("/files", middlewares.JWTMiddleware(http.HandlerFunc(handlers.GetFiles))).Methods("GET")
    router.Handle("/share/{file_id}", middlewares.JWTMiddleware(http.HandlerFunc(handlers.ShareFile))).Methods("GET")
    router.Handle("/file/update/{file_id}", middlewares.JWTMiddleware(http.HandlerFunc(handlers.UpdateFileMetadata))).Methods("POST")

    // Starting the server
    log.Println("Server is running on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", router))
}
