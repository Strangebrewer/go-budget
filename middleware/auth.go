package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey int

const (
	userIDKey contextKey = iota
	isDemoKey
	expiresAtKey
)

var ErrInvalidToken = errors.New("invalid token")

// RequireAuth parses the RSA public key PEM once at startup and returns a
// middleware that validates Bearer JWTs on every request.
func RequireAuth(publicKeyPEM string) (func(http.Handler) http.Handler, error) {
	pub, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := bearerToken(r)
			if !ok {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			userID, isDemo, demoExpiresAt, err := verifyAccessJWT(tokenStr, pub)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, isDemoKey, isDemo)
			ctx = context.WithValue(ctx, expiresAtKey, demoExpiresAt)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// UserIDFromContext retrieves the authenticated user ID injected by RequireAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// IsDemoFromContext reports whether the authenticated user is a demo account.
func IsDemoFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(isDemoKey).(bool)
	return v
}

// ExpiresAtFromContext returns the demo account expiry injected by RequireAuth, or nil.
func ExpiresAtFromContext(ctx context.Context) *time.Time {
	v, _ := ctx.Value(expiresAtKey).(*time.Time)
	return v
}

func verifyAccessJWT(tokenStr string, pub *rsa.PublicKey) (string, bool, *time.Time, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		return pub, nil
	})
	if err != nil || !tok.Valid {
		return "", false, nil, ErrInvalidToken
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", false, nil, ErrInvalidToken
	}

	if typ, _ := claims["typ"].(string); typ != "access" {
		return "", false, nil, ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", false, nil, ErrInvalidToken
	}

	isDemo, _ := claims["isDemo"].(bool)

	var demoExpiresAt *time.Time
	if ts, ok := claims["demoExpiresAt"].(float64); ok {
		t := time.Unix(int64(ts), 0).UTC()
		demoExpiresAt = &t
	}

	return sub, isDemo, demoExpiresAt, nil
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(header, "Bearer "), true
}

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, ErrInvalidToken
	}

	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if pub, ok := pubAny.(*rsa.PublicKey); ok {
			return pub, nil
		}
	}

	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}

	return nil, ErrInvalidToken
}
