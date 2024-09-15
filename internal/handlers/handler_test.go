package handlers

import (
    "bytes"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
)

// Helper function to create a new file upload request
func newFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile(paramName, path)
    if err != nil {
        return nil, err
    }

    _, err = file.Seek(0, 0)
    if err != nil {
        return nil, err
    }

    _, err = part.Write([]byte("fake file content"))
    if err != nil {
        return nil, err
    }

    for key, val := range params {
        _ = writer.WriteField(key, val)
    }
    err = writer.Close()
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequest("POST", uri, body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    return req, err
}

// TestHandleFileUpload tests the file upload functionality
func TestHandleFileUpload(t *testing.T) {
    // Create a mock request
    req, err := newFileUploadRequest("/upload", nil, "file", "testfile.txt")
    if err != nil {
        t.Fatalf("Failed to create upload request: %v", err)
    }

    // Create a ResponseRecorder to record the response
    rr := httptest.NewRecorder()

    // Call the handler
    handler := http.HandlerFunc(HandleFileUpload)
    handler.ServeHTTP(rr, req)

    // Check the status code
    if rr.Code != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusOK)
    }

    // Check the response body
    expected := "File uploaded successfully."
    if rr.Body.String()[:len(expected)] != expected {
        t.Errorf("handler returned unexpected body: got %v want %v",
            rr.Body.String(), expected)
    }
}

// TestShareFile tests the file sharing functionality
func TestShareFile(t *testing.T) {
    // Create a mock request with the file ID as a URL parameter
    req, err := http.NewRequest("GET", "/share/1", nil)
    if err != nil {
        t.Fatal(err)
    }

    // Use httptest.NewRecorder to create a ResponseRecorder (which satisfies http.ResponseWriter)
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(ShareFile)

    // Serve the HTTP request
    handler.ServeHTTP(rr, req)

    // Check the status code is what we expect
    if rr.Code != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            rr.Code, http.StatusOK)
    }

    // Check the response body is as expected
    expected := "Pre-signed URL:"
    if rr.Body.String()[:len(expected)] != expected {
        t.Errorf("handler returned unexpected body: got %v want %v",
            rr.Body.String(), expected)
    }
}
