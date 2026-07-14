package api

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// AuthConfig holds the single-tenant credentials accepted by the API. Either
// basic username/password or one of the configured API keys grants access.
type AuthConfig struct {
	Username string
	Password string
	APIKeys  []string
}

// Middleware enforces authentication. It accepts:
//   - Basic auth:   Authorization: Basic base64(user:pass)
//   - API key:      X-API-Key: <key>   or   Authorization: Bearer <key>
//
// For single-tenant operation the same credentials apply to every caller.
func (c AuthConfig) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health and GraphQL introspection over GET are still gated; the
		// caller must authenticate for all routes.
		if c.authenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="tpt-conduit"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func (c AuthConfig) authenticated(r *http.Request) bool {
	// API key via header.
	if key := r.Header.Get("X-API-Key"); key != "" && c.matchKey(key) {
		return true
	}
	// Bearer / Basic via Authorization header.
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return c.matchKey(strings.TrimPrefix(auth, "Bearer "))
	}
	if strings.HasPrefix(auth, "Basic ") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			return false
		}
		parts := strings.SplitN(string(raw), ":", 2)
		if len(parts) != 2 {
			return false
		}
		return c.matchUser(parts[0], parts[1])
	}
	return false
}

func (c AuthConfig) matchUser(user, pass string) bool {
	u := []byte(user)
	p := []byte(pass)
	if len(c.Username) == 0 || len(c.Password) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(u, []byte(c.Username)) == 1 &&
		subtle.ConstantTimeCompare(p, []byte(c.Password)) == 1
}

func (c AuthConfig) matchKey(key string) bool {
	got := []byte(key)
	for _, k := range c.APIKeys {
		if subtle.ConstantTimeCompare(got, []byte(k)) == 1 {
			return true
		}
	}
	return false
}
