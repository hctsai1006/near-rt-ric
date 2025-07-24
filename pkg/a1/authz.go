package a1

import (
	"context"
	"net/http"
)

// Role represents a user role.
type Role string

const (
	AdminRole    Role = "admin"
	OperatorRole Role = "operator"
	ViewerRole   Role = "viewer"
)

// HasRole checks if a user has a specific role.
func HasRole(ctx context.Context, role Role) bool {
	roles, ok := ctx.Value("roles").([]string)
	if !ok {
		return false
	}

	for _, r := range roles {
		if r == string(role) {
			return true
		}
	}

	return false
}

// Authorize is a middleware for authorization.
func Authorize(next http.Handler, requiredRole Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !HasRole(r.Context(), requiredRole) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
