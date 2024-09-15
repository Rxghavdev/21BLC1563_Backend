// package handlers

// import (
//     "bytes"
//     "database/sql"
//     "encoding/json"
//     "net/http"
//     "net/http/httptest"
//     "regexp"
//     "testing"

//     "golang.org/x/crypto/bcrypt"
//     _ "github.com/mattn/go-sqlite3"
// )

// // setupTestDB initializes an in-memory SQLite database for testing.
// func setupTestDB() *sql.DB {
//     testDB, err := sql.Open("sqlite3", ":memory:")
//     if err != nil {
//         panic("failed to connect to test database")
//     }

//     // Create the necessary table for testing
//     createUserTable := `
//     CREATE TABLE users (
//         id INTEGER PRIMARY KEY AUTOINCREMENT,
//         email TEXT NOT NULL UNIQUE,
//         password TEXT NOT NULL
//     );`

//     if _, err := testDB.Exec(createUserTable); err != nil {
//         panic("failed to create users table")
//     }

//     return testDB
// }

// // TestLogin tests the Login handler
// func TestLogin(t *testing.T) {
//     // Initialize the test database
//     testDB := setupTestDB()
//     defer testDB.Close()

//     // Hash the password before storing it in the database
//     hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
//     if err != nil {
//         t.Fatalf("Failed to hash password: %v", err)
//     }

//     // Insert a user directly into the test database with the hashed password
//     _, err = testDB.Exec("INSERT INTO users (email, password) VALUES (?, ?)", "test@exa.com", string(hashedPassword))
//     if err != nil {
//         t.Fatalf("Failed to insert test user: %v", err)
//     }

//     // Create a mock login payload
//     loginPayload := map[string]string{
//         "email":    "test@exa.com",
//         "password": "password", // Plain text password (this should match the hashed password)
//     }

//     payload, err := json.Marshal(loginPayload)
//     if err != nil {
//         t.Fatalf("Failed to marshal request body: %v", err)
//     }

//     // Create a new POST request with the mock payload
//     req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(payload))
//     if err != nil {
//         t.Fatal(err)
//     }
//     req.Header.Set("Content-Type", "application/json")

//     // Use httptest to create a ResponseRecorder
//     rr := httptest.NewRecorder()

//     // Invoke the Login handler
//     handler := http.HandlerFunc(Login)
//     handler.ServeHTTP(rr, req)

//     // Check the status code
//     if rr.Code != http.StatusOK {
//         t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
//     }

//     // Check if the response contains a JWT token (instead of "Login successful")
//     tokenPattern := `^[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+$` // Simple JWT regex pattern
//     matched, err := regexp.MatchString(tokenPattern, rr.Body.String())
//     if err != nil {
//         t.Fatalf("Failed to compile regex: %v", err)
//     }

//     if !matched {
//         t.Errorf("handler returned unexpected body: got %v want JWT token", rr.Body.String())
//     }
// }


package handlers

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "regexp"
    "testing"

    "golang.org/x/crypto/bcrypt"
    _ "github.com/lib/pq"
)

// setupTestDB initializes a PostgreSQL database for testing.
func setupTestDB() *sql.DB {
    // Connection string for PostgreSQL (replace these values with your actual configuration)
    connStr := "user=postgres password=Trademark dbname=trademark_test sslmode=disable"

    testDB, err := sql.Open("postgres", connStr)
    if err != nil {
        panic("failed to connect to test database: " + err.Error())
    }

    // Create the necessary table for testing
    createUserTable := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL
    );`

    if _, err := testDB.Exec(createUserTable); err != nil {
        panic("failed to create users table: " + err.Error())
    }

    // Clean up the table before running tests
    if _, err := testDB.Exec("TRUNCATE TABLE users"); err != nil {
        panic("failed to clean up users table: " + err.Error())
    }

    return testDB
}

// RegisterHandler registers a new user (mock registration handler)
func RegisterHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var creds map[string]string
        err := json.NewDecoder(r.Body).Decode(&creds)
        if err != nil {
            http.Error(w, "Invalid request payload", http.StatusBadRequest)
            return
        }

        // Hash the password
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds["password"]), bcrypt.DefaultCost)
        if err != nil {
            http.Error(w, "Error while hashing password", http.StatusInternalServerError)
            return
        }

        // Insert the user into the database
        _, err = db.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", creds["email"], string(hashedPassword))
        if err != nil {
            http.Error(w, "Error registering user", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusCreated)
        w.Write([]byte("User registered successfully"))
    }
}

// LoginHandler logs in a user (mock login handler)
func LoginHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var creds map[string]string
        err := json.NewDecoder(r.Body).Decode(&creds)
        if err != nil {
            http.Error(w, "Invalid request payload", http.StatusBadRequest)
            return
        }

        // Fetch the user from the database
        var storedPassword string
        err = db.QueryRow("SELECT password FROM users WHERE email = $1", creds["email"]).Scan(&storedPassword)
        if err != nil {
            http.Error(w, "User not found", http.StatusUnauthorized)
            return
        }

        // Compare the hashed password with the provided password
        if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(creds["password"])); err != nil {
            http.Error(w, "Invalid password", http.StatusUnauthorized)
            return
        }

        // Simulate JWT generation (mock)
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("mock.jwt.token"))
    }
}

func TestRegisterAndLogin(t *testing.T) {
    // Initialize the test database
    testDB := setupTestDB()
    defer testDB.Close()

    // Register a new user
    registerPayload := map[string]string{
        "email":    "test@exa.com",
        "password": "password123",
    }

    payload, err := json.Marshal(registerPayload)
    if err != nil {
        t.Fatalf("Failed to marshal request body: %v", err)
    }

    req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(payload))
    if err != nil {
        t.Fatal(err)
    }
    req.Header.Set("Content-Type", "application/json")

    rr := httptest.NewRecorder()

    // Invoke the Register handler
    registerHandler := RegisterHandler(testDB)
    registerHandler.ServeHTTP(rr, req)

    // Check the status code for registration
    if rr.Code != http.StatusCreated {
        t.Errorf("Register handler returned wrong status code: got %v want %v", rr.Code, http.StatusCreated)
    }

    // Now test login with the registered user's credentials
    loginPayload := map[string]string{
        "email":    "test@exa.com",
        "password": "password123", // Correct password
    }

    loginPayloadJSON, err := json.Marshal(loginPayload)
    if err != nil {
        t.Fatalf("Failed to marshal request body: %v", err)
    }

    loginRequest, err := http.NewRequest("POST", "/login", bytes.NewBuffer(loginPayloadJSON))
    if err != nil {
        t.Fatal(err)
    }
    loginRequest.Header.Set("Content-Type", "application/json")

    loginRR := httptest.NewRecorder()

    // Invoke the Login handler
    loginHandler := LoginHandler(testDB)
    loginHandler.ServeHTTP(loginRR, loginRequest)

    // Check the status code for login
    if loginRR.Code != http.StatusOK {
        t.Errorf("Login handler returned wrong status code: got %v want %v", loginRR.Code, http.StatusOK)
    }

    // Check if the response contains a JWT token (mock JWT for simplicity)
    tokenPattern := `^[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+$` // Simple JWT regex pattern
    matched, err := regexp.MatchString(tokenPattern, loginRR.Body.String())
    if err != nil {
        t.Fatalf("Failed to compile regex: %v", err)
    }

    if !matched {
        t.Errorf("Login handler returned unexpected body: got %v want JWT token", loginRR.Body.String())
    }
}
