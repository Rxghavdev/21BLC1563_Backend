# File Sharing & Management System

## Description

This project is a file-sharing platform that allows users to upload, manage, and share files. The system is designed to handle multiple users, store metadata in PostgreSQL, manage file uploads to S3 (or locally if cloud integration is unavailable), and implement caching for file metadata. The backend is built using Go, with a focus on handling concurrency and performance optimizations.

## Features

### User Authentication & Authorization

Users can register and log in with email and password. Upon successful login, a JWT token is generated to authenticate and authorize users for subsequent requests.

- **Register:**
  ```http
  POST /register
  ```
  Request body:
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```

- **Login:**
  ```http
  POST /login
  ```
  Request body:
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```

### File Upload & Management

Users can upload files to S3 or local storage. The system stores metadata of uploaded files in PostgreSQL and handles concurrent processing for large uploads using goroutines.

- **Upload File:**
  ```http
  POST /upload
  ```
  Example using `curl`:
  ```bash
  curl -X POST "http://localhost:8080/upload" -H "Authorization: Bearer <JWT_TOKEN>" -F "file=@/path/to/your/file"
  ```

### File Retrieval & Sharing

Users can retrieve metadata for their uploaded files and share file URLs via a public link.

- **Retrieve Files:**
  ```http
  GET /files
  ```
  Example using `curl`:
  ```bash
  curl -X GET "http://localhost:8080/files" -H "Authorization: Bearer <JWT_TOKEN>"
  ```

- **Share File:**
  ```http
  GET /share/:file_id
  ```
  Example using `curl`:
  ```bash
  curl -X GET "http://localhost:8080/share/12345" -H "Authorization: Bearer <JWT_TOKEN>"
  ```

### File Search

Users can search their files by name, upload date, or file type. The search is optimized to handle large datasets efficiently.

- **Search Files:**
  ```http
  GET /search
  ```
  Example using `curl`:
  ```bash
  curl -X GET "http://localhost:8080/search?name=file.txt" -H "Authorization: Bearer <JWT_TOKEN>"
  ```

### Caching Layer for File Metadata

The system implements a caching mechanism using Redis to reduce database load. Metadata is cached on retrieval and invalidated when updated.

### Database Interaction

The system stores user data and file metadata in PostgreSQL. Efficient queries are designed to retrieve user-specific files.

### Background Job for File Deletion

A background worker periodically deletes expired files from S3 and their metadata from the database, ensuring the system stays clean and efficient.

## Setup Instructions

1. **Clone the repository:**
   ```bash
   git clone https://github.com/21BLC1563_Backend.git
   cd 21BLC1563_Backend
   ```

2. **Set up PostgreSQL database and Redis cache:**
   - Ensure PostgreSQL and Redis are running on your system.

3. **Environment Variables:**
   - Create a `.env` file and add the following variables:
     ```
     DB_HOST=localhost
     DB_PORT=5432
     DB_USER=your_db_user
     DB_PASSWORD=your_db_password
     DB_NAME=your_db_name
     REDIS_HOST=localhost
     REDIS_PORT=6379
     AWS_ACCESS_KEY_ID=your_aws_access_key
     AWS_SECRET_ACCESS_KEY=your_aws_secret_key
     JWT_SECRET=your_jwt_secret
     ```

4. **Run the application:**
   ```bash
   go run main.go
   ```

5. **Dockerize the application:**
   - Build and run the Docker container.
   ```bash
   docker build -t file-sharing-app .
   docker run -p 8080:8080 file-sharing-app
   ```



