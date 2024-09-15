package middleware

import (
    "net/http"
    "sync"
    "time"
)

var userRequests sync.Map

type requestCount struct {
    Count    int
    LastSeen time.Time
}

const requestLimit = 100
const windowDuration = time.Hour

// RateLimitMiddleware enforces a rate limit per user.
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userIP := getUserIP(r)

        value, _ := userRequests.LoadOrStore(userIP, &requestCount{
            Count:    0,
            LastSeen: time.Now(),
        })

        reqData := value.(*requestCount)

        if time.Since(reqData.LastSeen) > windowDuration {
            reqData.Count = 0
            reqData.LastSeen = time.Now()
        }

        if reqData.Count >= requestLimit {
            http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
            return
        }

        reqData.Count++
        userRequests.Store(userIP, reqData)

        next.ServeHTTP(w, r)
    })
}

func getUserIP(r *http.Request) string {
    ip := r.Header.Get("X-Real-Ip")
    if ip == "" {
        ip = r.Header.Get("X-Forwarded-For")
    }
    if ip == "" {
        ip = r.RemoteAddr
    }
    return ip
}
