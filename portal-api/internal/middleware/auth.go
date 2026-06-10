package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	devModeWarnOnce sync.Once
	authLogger      *zap.Logger
)

func init() {
	authLogger, _ = zap.NewProduction()
}

// Auth validates Keycloak OIDC JWT tokens.
// When DEV_MODE=true, skips authentication for local development.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("DEV_MODE") == "true" {
			devModeWarnOnce.Do(func() {
				if authLogger != nil {
					authLogger.Warn("DEV_MODE enabled: authentication is bypassed. DO NOT USE IN PRODUCTION.")
				}
			})
			c.Set("user_id", "dev-user")
			c.Set("roles", []string{"central-operator"})
			c.Next()
			return
		}

		token := ""
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if queryToken := c.Query("access_token"); queryToken != "" {
			token = queryToken
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		// TODO: Validate JWT signature against Keycloak JWKS endpoint.
		// Signature validation is currently pending — only claims are parsed here.
		claims := parseJWTClaims(token)

		sub, _ := claims["sub"].(string)
		if sub == "" {
			sub = "unknown"
		}
		c.Set("user_id", sub)

		// Keycloak stores roles under realm_access.roles
		roles := extractRealmRoles(claims)
		c.Set("roles", roles)

		// Custom claim: tenant_id (set in Keycloak via mapper)
		if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
			c.Set("tenant_id", tid)
		}

		c.Next()
	}
}

// RequireRole returns a middleware that aborts with 403 if the caller does not have one of the allowed roles.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		roles, _ := c.Get("roles")
		roleList, _ := roles.([]string)
		for _, r := range roleList {
			if _, ok := allowedSet[r]; ok {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient role"})
	}
}

// HasRole reports whether the current request context carries the given role.
func HasRole(c *gin.Context, role string) bool {
	roles, _ := c.Get("roles")
	roleList, _ := roles.([]string)
	for _, r := range roleList {
		if r == role {
			return true
		}
	}
	return false
}

// parseJWTClaims decodes the payload of a JWT without verifying its signature.
// Returns nil on any parse failure.
func parseJWTClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

// extractRealmRoles extracts Keycloak realm_access.roles from parsed JWT claims.
func extractRealmRoles(claims map[string]any) []string {
	if claims == nil {
		return nil
	}
	realmAccess, _ := claims["realm_access"].(map[string]any)
	if realmAccess == nil {
		return nil
	}
	raw, _ := realmAccess["roles"].([]any)
	roles := make([]string, 0, len(raw))
	for _, r := range raw {
		if s, ok := r.(string); ok {
			roles = append(roles, s)
		}
	}
	return roles
}
