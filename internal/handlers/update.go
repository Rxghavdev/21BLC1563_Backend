package handlers

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/gorilla/mux"
    "trademarkia/internal/db"
)

var (
    redisCtx = context.Background() // Avoid conflict with the context variable in file.go
)

// UpdateFileMetadata updates the file metadata (e.g., file name) in the database and invalidates the cache
func UpdateFileMetadata(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    fileID, err := strconv.Atoi(vars["file_id"])
    if err != nil {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }

    newFileName := r.FormValue("new_file_name") // Assuming the new file name is sent as form data
    if newFileName == "" {
        http.Error(w, "New file name is required", http.StatusBadRequest)
        return
    }

    // Update the database with the new file name
    _, err = db.DB.Exec("UPDATE files SET file_name = $1 WHERE id = $2", newFileName, fileID)
    if err != nil {
        log.Printf("Error updating file metadata: %v", err)
        http.Error(w, "Error updating file metadata", http.StatusInternalServerError)
        return
    }

    // Invalidate the Redis cache for the file
    cacheKey := fmt.Sprintf("file_%d", fileID)
    err = rdb.Del(redisCtx, cacheKey).Err()
    if err != nil {
        log.Printf("Error invalidating cache: %v", err)
        http.Error(w, "Error invalidating cache", http.StatusInternalServerError)
        return
    }

    // Cache the updated file metadata with a 5-minute expiration
    cacheFileMetadata(fileID, newFileName)

    fmt.Fprintf(w, "File metadata updated successfully, cache invalidated")
}

// cacheFileMetadata stores file metadata in Redis with 5-minute expiration
func cacheFileMetadata(fileID int, fileName string) {
    err := rdb.Set(redisCtx, fmt.Sprintf("file_%d", fileID), fileName, 5*time.Minute).Err()
    if err != nil {
        log.Println("Error caching file metadata in Redis:", err)
    } else {
        log.Printf("File metadata cached for file ID: %d", fileID)
    }
}

// fetchFileMetadataFromCache fetches the file metadata from Redis cache or database
func fetchFileMetadataFromCache(fileID int) (string, error) {
    cacheKey := fmt.Sprintf("file_%d", fileID)

    // Check if the metadata is cached
    cachedFileName, err := rdb.Get(redisCtx, cacheKey).Result()
    if err == redis.Nil {
        // Cache miss, retrieve from DB and cache it
        var fileName string
        err = db.DB.QueryRow("SELECT file_name FROM files WHERE id = $1", fileID).Scan(&fileName)
        if err != nil {
            log.Printf("Error retrieving file metadata from DB: %v", err)
            return "", err
        }

        // Cache the result with a 5-minute expiration
        cacheFileMetadata(fileID, fileName)

        return fileName, nil
    } else if err != nil {
        return "", err
    }

    // Cache hit, return cached value
    return cachedFileName, nil
}
