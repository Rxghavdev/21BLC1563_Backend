package middlewares

import (
    "context"
    "net/http"
    "strings"
    "trademarkia/internal/handlers"
    "github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("my_secret_key")

// JWTMiddleware authenticates requests and extracts user information
func JWTMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
            http.Error(w, "Authorization token required", http.StatusUnauthorized)
            return
        }

        // Ensure the token starts with "Bearer "
        if !strings.HasPrefix(tokenString, "Bearer ") {
            http.Error(w, "Invalid token format", http.StatusUnauthorized)
            return
        }

        tokenString = strings.TrimPrefix(tokenString, "Bearer ")

        claims := &handlers.Claims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
            return jwtKey, nil
        })

        if err != nil || !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Add user_id to context for further use
        ctx := context.WithValue(r.Context(), "userID", claims.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
