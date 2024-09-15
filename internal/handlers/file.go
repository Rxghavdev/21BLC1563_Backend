package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"
    
    "github.com/go-redis/redis/v8" // Redis client
    "github.com/gorilla/mux"
    "trademarkia/internal/db"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

var (
    s3session *s3.S3
    rdb       *redis.Client // Redis client instance
    ctx       = context.Background()
)

func init() {
    // Initialize Redis
    rdb = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Default Redis port
        Password: "",               // No password set
        DB:       0,                // Default DB
    })

    s3session = s3.New(session.Must(session.NewSession(&aws.Config{
        Region: aws.String("ap-south-1"), // Use your AWS region
    })))
}

// HandleFileUpload handles file uploads and saves metadata in PostgreSQL
func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("userID").(int) // Extract user_id from the context (added by JWT middleware)

    r.ParseMultipartForm(10 << 20) // Limit file size to 10 MB
    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error reading file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Read the file into a buffer
    buffer := make([]byte, handler.Size)
    _, err = file.Read(buffer)
    if err != nil {
        http.Error(w, "Error reading file", http.StatusInternalServerError)
        return
    }

    // Save file metadata in PostgreSQL (using user_id)
    fileID := storeFileMetadata(handler.Filename, handler.Size, userID)

    // Upload the file to S3
    fileURL, err := processFileUpload(handler.Filename, buffer)
    if err != nil {
        http.Error(w, "Error uploading file to S3", http.StatusInternalServerError)
        return
    }

    // Update the database with the S3 URL
    _, err = db.DB.Exec("UPDATE files SET file_url = $1 WHERE id = $2", fileURL, fileID)
    if err != nil {
        log.Println("Error updating file metadata:", err)
        http.Error(w, "Error updating file metadata", http.StatusInternalServerError)
        return
    }

    // Cache the metadata in Redis
    cacheFileMetadata(fileID, fileURL)

    w.Write([]byte(fmt.Sprintf("File uploaded successfully. Public URL: %s", fileURL)))
}

// storeFileMetadata saves the file metadata in the database and returns the file ID
func storeFileMetadata(filename string, fileSize int64, userID int) int {
    var fileID int
    err := db.DB.QueryRow("INSERT INTO files (user_id, file_name, file_size, upload_date) VALUES ($1, $2, $3, $4) RETURNING id",
        userID, filename, fileSize, time.Now()).Scan(&fileID)
    if err != nil {
        log.Println("Error saving file metadata:", err)
        return 0
    }
    return fileID
}

// cacheFileMetadata stores file metadata in Redis
func cacheFileMetadata(fileID int, fileURL string) {
    // Cache file URL with expiration (optional)
    rdb.Set(ctx, fmt.Sprintf("file_%d", fileID), fileURL, 24*time.Hour)
}

// processFileUpload handles the actual file upload to S3
func processFileUpload(filename string, fileBytes []byte) (string, error) {
    log.Printf("Processing upload for file: %s", filename)

    // Upload the file to S3
    fileURL, err := uploadToS3(filename, fileBytes)
    if err != nil {
        log.Println("Error uploading to S3:", err)
        return "", err
    }

    return fileURL, nil
}

// uploadToS3 handles uploading the file to S3
func uploadToS3(filename string, fileBytes []byte) (string, error) {
    _, err := s3session.PutObject(&s3.PutObjectInput{
        Bucket:               aws.String("trademarkiaa"),  // Your bucket name
        Key:                  aws.String(filename),
        Body:                 bytes.NewReader(fileBytes),
        ContentLength:        aws.Int64(int64(len(fileBytes))),
        ContentType:          aws.String(http.DetectContentType(fileBytes)),
        ContentDisposition:   aws.String("attachment"),
        ServerSideEncryption: aws.String("AES256"),
    })
    if err != nil {
        return "", err
    }

    // Replace spaces with +
    encodedFilename := strings.ReplaceAll(filename, " ", "+")

    // Generate the public URL (for ap-south-1 region)
    fileURL := fmt.Sprintf("https://trademarkiaa.s3.ap-south-1.amazonaws.com/%s", encodedFilename)
    return fileURL, nil
}

// GetFiles retrieves all files uploaded by the user
func GetFiles(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("userID").(int) // Extract user_id from JWT

    rows, err := db.DB.Query("SELECT id, file_name, file_url, upload_date, file_size FROM files WHERE user_id = $1", userID)
    if err != nil {
        log.Println("Error retrieving files:", err)
        http.Error(w, "Error retrieving files", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var files []map[string]interface{}

    for rows.Next() {
        var fileID int
        var fileName, fileURL string
        var uploadDate time.Time
        var fileSize int64

        if err := rows.Scan(&fileID, &fileName, &fileURL, &uploadDate, &fileSize); err != nil {
            log.Println("Error scanning files:", err)
            http.Error(w, "Error scanning files", http.StatusInternalServerError)
            return
        }

        fileData := map[string]interface{}{
            "file_id":     fileID,
            "file_name":   fileName,
            "file_url":    fileURL,
            "upload_date": uploadDate,
            "file_size":   fileSize,
        }
        files = append(files, fileData)
    }

    json.NewEncoder(w).Encode(files)
}

// ShareFile allows a user to share a public link for a file by its ID
func ShareFile(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    fileID, err := strconv.Atoi(vars["file_id"])
    if err != nil {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }

    // Check Redis for cached file metadata
    cachedURL, err := rdb.Get(ctx, fmt.Sprintf("file_%d", fileID)).Result()
    if err == redis.Nil {
        // If not in cache, retrieve from the database
        var fileURL string
        err := db.DB.QueryRow("SELECT file_url FROM files WHERE id = $1", fileID).Scan(&fileURL)
        if err != nil {
            log.Println("Error retrieving file URL:", err)
            http.Error(w, "Error retrieving file URL", http.StatusInternalServerError)
            return
        }

        // Cache the file URL in Redis
        cacheFileMetadata(fileID, fileURL)

        w.Write([]byte(fmt.Sprintf("File URL: %s", fileURL)))
    } else if err != nil {
        log.Println("Error retrieving from cache:", err)
        http.Error(w, "Error retrieving from cache", http.StatusInternalServerError)
    } else {
        // If found in cache, return cached URL
        w.Write([]byte(fmt.Sprintf("File URL (cached): %s", cachedURL)))
    }
}
