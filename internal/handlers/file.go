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
    "bytes"
    "database/sql"

    "github.com/go-redis/redis/v8"
    "github.com/gorilla/mux"
    "trademarkia/internal/db"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

var (
    s3session *s3.S3
    rdb       *redis.Client
    ctx       = context.Background()
)

func init() {
    // Initialize Redis
    rdb = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Default Redis port
        Password: "",               // No password set
        DB:       0,                // Default DB
    })

    // Initialize AWS S3 session
    s3session = s3.New(session.Must(session.NewSession(&aws.Config{
        Region: aws.String("ap-south-1"),
    })))
}

// HandleFileUpload handles file uploads and saves metadata in PostgreSQL
// HandleFileUpload handles file uploads and saves metadata in PostgreSQL
func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("userID").(int)

    // Parse multipart form data
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

    // Start a database transaction
    tx, err := db.DB.Begin()
    if err != nil {
        log.Println("Error starting transaction:", err)
        http.Error(w, "Error initializing database transaction", http.StatusInternalServerError)
        return
    }

    // Save file metadata within the transaction
    var fileID int
    err = tx.QueryRow("INSERT INTO files (user_id, file_name, file_size, upload_date) VALUES ($1, $2, $3, $4) RETURNING id",
        userID, handler.Filename, handler.Size, time.Now()).Scan(&fileID)
    if err != nil {
        log.Println("Error saving file metadata:", err)
        tx.Rollback() // Rollback the transaction if there's an error
        http.Error(w, "Error saving file metadata", http.StatusInternalServerError)
        return
    }

    // Upload the file to S3
    fileURL, err := processFileUpload(handler.Filename, buffer)
    if err != nil {
        log.Println("Error uploading file to S3:", err)
        tx.Rollback() // Rollback the transaction if there's an error with S3 upload
        http.Error(w, "Error uploading file to S3", http.StatusInternalServerError)
        return
    }

    // Update the file URL in the database
    _, err = tx.Exec("UPDATE files SET file_url = $1 WHERE id = $2", fileURL, fileID)
    if err != nil {
        log.Println("Error updating file URL in database:", err)
        tx.Rollback() // Rollback the transaction if there's an error updating the URL
        http.Error(w, "Error updating file URL", http.StatusInternalServerError)
        return
    }

    // Commit the transaction if all steps succeed
    err = tx.Commit()
    if err != nil {
        log.Println("Error committing transaction:", err)
        http.Error(w, "Error finalizing file upload", http.StatusInternalServerError)
        return
    }

    cacheFileMetadata(fileID, handler.Filename)

    w.Write([]byte(fmt.Sprintf("File uploaded successfully. Public URL: %s", fileURL)))
}


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

func cacheFileMetadatad(fileID int, fileName string) {
    err := rdb.Set(ctx, fmt.Sprintf("file_%d", fileID), fileName, 24*time.Hour).Err()
    if err != nil {
        log.Println("Error caching file metadata in Redis:", err)
    } else {
        log.Printf("File metadata cached for file ID: %d", fileID)
    }
}

func processFileUpload(filename string, fileBytes []byte) (string, error) {
    log.Printf("Processing upload for file: %s", filename)

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
        Bucket:               aws.String("trademarkiaa"),
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

    encodedFilename := strings.ReplaceAll(filename, " ", "%20")
    fileURL := fmt.Sprintf("https://trademarkiaa.s3.ap-south-1.amazonaws.com/%s", encodedFilename)
    return fileURL, nil
}

// GetFiles retrieves all files uploaded by the user
func GetFiles(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("userID").(int)

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
        var fileName string
        var fileURL sql.NullString
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
            "file_url":    "No URL available",
            "upload_date": uploadDate,
            "file_size":   fileSize,
        }

        if fileURL.Valid {
            fileData["file_url"] = fileURL.String
        }

        files = append(files, fileData)
    }

    json.NewEncoder(w).Encode(files)
}

// GeneratePreSignedURL generates a pre-signed URL with expiration
func GeneratePreSignedURL(filename string, expiration time.Duration) (string, error) {
    req, _ := s3session.GetObjectRequest(&s3.GetObjectInput{
        Bucket: aws.String("trademarkiaa"),
        Key:    aws.String(filename),
    })

    presignedURL, err := req.Presign(expiration)
    if err != nil {
        return "", err
    }

    return presignedURL, nil
}

// ShareFile allows a user to share a public link for a file by its ID
func ShareFile(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    fileID, err := strconv.Atoi(vars["file_id"])
    if err != nil {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }

    key := fmt.Sprintf("file_%d", fileID)
    cachedFileName, err := rdb.Get(ctx, key).Result()

    if err == redis.Nil {
        // Cache miss, retrieve from DB
        var fileName sql.NullString
        err := db.DB.QueryRow("SELECT file_name FROM files WHERE id = $1", fileID).Scan(&fileName)
        if err != nil {
            log.Println("Error retrieving file name:", err)
            http.Error(w, "Error retrieving file name", http.StatusInternalServerError)
            return
        }

        if !fileName.Valid {
            http.Error(w, "No file available", http.StatusNotFound)
            return
        }

        // Cache the file name in Redis
        cacheFileMetadatad(fileID, fileName.String)

        preSignedURL, err := GeneratePreSignedURL(fileName.String, 1*time.Hour)
        if err != nil {
            log.Println("Error generating pre-signed URL:", err)
            http.Error(w, "Error generating pre-signed URL", http.StatusInternalServerError)
            return
        }

        w.Write([]byte(fmt.Sprintf("Pre-signed URL: %s", preSignedURL)))
    } else if err != nil {
        log.Printf("Error retrieving from Redis for key: %s, err: %v", key, err)
        http.Error(w, "Error retrieving from cache", http.StatusInternalServerError)
    } else {
        preSignedURL, err := GeneratePreSignedURL(cachedFileName, 1*time.Hour)
        if err != nil {
            log.Println("Error generating pre-signed URL:", err)
            http.Error(w, "Error generating pre-signed URL", http.StatusInternalServerError)
            return
        }

        w.Write([]byte(fmt.Sprintf("Pre-signed URL (cached): %s", preSignedURL)))
    }
}
