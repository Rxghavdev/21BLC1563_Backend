package handlers

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "time"
    "trademarkia/internal/db"

    "github.com/dgrijalva/jwt-go"
    "golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("my_secret_key")

// User struct defines the registration credentials
type User struct {
    ID        int       `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"password"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Credentials struct for login
type Credentials struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// Claims defines the structure for JWT payload
type Claims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    jwt.StandardClaims
}

// HashPassword hashes the plain password using bcrypt
func HashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashedPassword), nil
}

// CheckPasswordHash compares the provided password with the hashed password stored in the database
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// RegisterUser handles user registration
func RegisterUser(w http.ResponseWriter, r *http.Request) {
    var user User
    err := json.NewDecoder(r.Body).Decode(&user)
    if err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    // Hash the password before storing it
    hashedPassword, err := HashPassword(user.Password)
    if err != nil {
        http.Error(w, "Error hashing password", http.StatusInternalServerError)
        return
    }

    // Insert user into the database
    _, err = db.DB.Exec("INSERT INTO users (email, password) VALUES ($1, $2)", user.Email, hashedPassword)
    if err != nil {
        log.Println("Error saving user:", err)
        http.Error(w, "Error saving user", http.StatusInternalServerError)
        return
    }

    w.Write([]byte("User registered successfully"))
}

// Login handles user login and returns JWT
func Login(w http.ResponseWriter, r *http.Request) {
    var creds Credentials
    err := json.NewDecoder(r.Body).Decode(&creds)
    if err != nil {
        http.Error(w, "Invalid input", http.StatusBadRequest)
        return
    }

    var storedPassword string
    var userID int
    err = db.DB.QueryRow("SELECT id, password FROM users WHERE email=$1", creds.Email).Scan(&userID, &storedPassword)
    if err == sql.ErrNoRows || !CheckPasswordHash(creds.Password, storedPassword) {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Set the expiration time for the token (1 hour from now)
    expirationTime := time.Now().Add(1 * time.Hour).Unix()

    // Create the JWT claims, including the user_id and expiration time
    claims := &Claims{
        UserID: userID,
        Email:  creds.Email,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: expirationTime,
        },
    }

    // Generate the token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString(jwtKey)
    if err != nil {
        http.Error(w, "Error generating token", http.StatusInternalServerError)
        return
    }

    // Set token as a cookie in the response
    http.SetCookie(w, &http.Cookie{
        Name:    "token",
        Value:   tokenString,
        Expires: time.Unix(expirationTime, 0),
    })

    // Return the token in the response body
    w.Write([]byte(tokenString))
}
