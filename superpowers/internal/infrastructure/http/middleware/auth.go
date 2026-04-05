package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// APIKey returns a middleware that requires a valid Bearer token in the Authorization header.
// Uses constant-time comparison to prevent timing attacks.
func APIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			key := strings.TrimPrefix(auth, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}) //nolint:errcheck
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
