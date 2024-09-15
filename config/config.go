package config

import (
    "os"
)

func GetEnv(key string, defaultValue string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return defaultValue
    }
    return value
}
