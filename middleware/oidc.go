package middleware

import (
	"net/http"
	"strings"

	"google.golang.org/api/idtoken"
)

// RequirePubSubOIDC validates the Google OIDC token attached by Pub/Sub push subscriptions.
// If audience is empty (local dev), validation is skipped and the request passes through.
func RequirePubSubOIDC(audience string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if audience == "" {
				next.ServeHTTP(w, r)
				return
			}

			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")

			if _, err := idtoken.Validate(r.Context(), token, audience); err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
