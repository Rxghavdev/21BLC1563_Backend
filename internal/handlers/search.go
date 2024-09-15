package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "time"
    "trademarkia/internal/db"
)

// FileSearchResult defines the structure for search results with nullable file_url
type FileSearchResult struct {
    FileName   string         `json:"file_name"`
    FileURL    string         `json:"file_url"`
    UploadDate time.Time      `json:"upload_date"`
    FileSize   int64          `json:"file_size"`
}

// HandleFileSearch handles file search based on various criteria
func HandleFileSearch(w http.ResponseWriter, r *http.Request) {
    // Extract query parameters
    userID := r.Context().Value("userID").(int)  // Extract the user ID from the JWT middleware
    fileName := r.URL.Query().Get("file_name")
    uploadDate := r.URL.Query().Get("upload_date")
    fileType := r.URL.Query().Get("file_type")
    page := r.URL.Query().Get("page")
    limit := r.URL.Query().Get("limit")

    // Set default pagination values if not provided
    if page == "" {
        page = "1"
    }
    if limit == "" {
        limit = "10"  // Default limit is 10 results per page
    }

    // Convert page and limit to integers
    pageInt, _ := strconv.Atoi(page)
    limitInt, _ := strconv.Atoi(limit)
    offset := (pageInt - 1) * limitInt

    // Build the SQL query dynamically based on provided filters
    query := "SELECT file_name, file_url, upload_date, file_size FROM files WHERE user_id = $1"
    args := []interface{}{userID}
    argIndex := 2

    // Modify the file name filter to match files that start with the provided name
    if fileName != "" {
        query += fmt.Sprintf(" AND file_name ILIKE $%d", argIndex)
        args = append(args, fileName+"%") // Match files that start with the search term
        argIndex++
    }

    if uploadDate != "" {
        query += fmt.Sprintf(" AND upload_date::date = $%d", argIndex)
        args = append(args, uploadDate) // Ensure the date is in YYYY-MM-DD format
        argIndex++
    }

    if fileType != "" {
        query += fmt.Sprintf(" AND file_name ILIKE $%d", argIndex)
        args = append(args, "%."+fileType) // Search for files by extension (e.g., ".pdf")
        argIndex++
    }

    query += fmt.Sprintf(" ORDER BY upload_date DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
    args = append(args, limitInt, offset)

    // Execute the query
    rows, err := db.DB.Query(query, args...)
    if err != nil {
        log.Println("Error executing search query:", err)
        http.Error(w, "Error retrieving files", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var results []FileSearchResult

    // Fetch the results and handle NULL values for file_url
    for rows.Next() {
        var result FileSearchResult
        var fileURL sql.NullString

        // Scan the result, using sql.NullString for file_url
        if err := rows.Scan(&result.FileName, &fileURL, &result.UploadDate, &result.FileSize); err != nil {
            log.Println("Error scanning search results:", err)
            http.Error(w, "Error processing results", http.StatusInternalServerError)
            return
        }

        // If fileURL is NULL, provide a default empty string
        if fileURL.Valid {
            result.FileURL = fileURL.String
        } else {
            result.FileURL = "No URL available"
        }

        results = append(results, result)
    }

    // Check if any rows were returned
    if len(results) == 0 {
        w.Write([]byte("No files found matching the search criteria."))
        return
    }

    // Return the search results as JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}
