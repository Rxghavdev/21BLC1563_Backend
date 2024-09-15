package background

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/s3"
    "trademarkia/internal/db"
    "github.com/aws/aws-sdk-go/aws/session"
)

var (
    s3session *s3.S3
    ctx       = context.Background()
)

func init() {
    s3session = s3.New(session.Must(session.NewSession(&aws.Config{
        Region: aws.String("ap-south-1"),
    })))
}

func StartFileDeletionWorker() {
    ticker := time.NewTicker(20 * time.Minute)

    go func() {
        for {
            select {
            case <-ticker.C:
                log.Println("Running background job for file deletion...")
                deleteExpiredFiles()
            }
        }
    }()
}

func deleteExpiredFiles() {
	expiryThreshold := time.Now().Add(-20 * time.Minute)


    rows, err := db.DB.Query("SELECT id, file_name FROM files WHERE upload_date < $1", expiryThreshold)
    if err != nil {
        log.Printf("Error fetching expired files: %v", err)
        return
    }
    defer rows.Close()

    var expiredFiles []struct {
        ID       int
        FileName string
    }

    for rows.Next() {
        var fileID int
        var fileName string
        if err := rows.Scan(&fileID, &fileName); err != nil {
            log.Printf("Error scanning expired files: %v", err)
            continue
        }
        expiredFiles = append(expiredFiles, struct {
            ID       int
            FileName string
        }{fileID, fileName})
    }

    for _, file := range expiredFiles {
        err := deleteFileFromS3(file.FileName)
        if err != nil {
            log.Printf("Error deleting file from S3: %v", err)
            continue
        }

        _, err = db.DB.Exec("DELETE FROM files WHERE id = $1", file.ID)
        if err != nil {
            log.Printf("Error deleting file metadata from DB: %v", err)
            continue
        }

        log.Printf("Deleted file and metadata for file ID: %d", file.ID)
    }
}

func deleteFileFromS3(fileName string) error {
    _, err := s3session.DeleteObject(&s3.DeleteObjectInput{
        Bucket: aws.String("trademarkiaa"), 
        Key:    aws.String(fileName),       
    })
    if err != nil {
        return fmt.Errorf("failed to delete object %s from S3: %v", fileName, err)
    }

    err = s3session.WaitUntilObjectNotExists(&s3.HeadObjectInput{
        Bucket: aws.String("trademarkiaa"),
        Key:    aws.String(fileName),
    })
    if err != nil {
        return fmt.Errorf("error waiting for S3 object %s to be deleted: %v", fileName, err)
    }

    return nil
}
