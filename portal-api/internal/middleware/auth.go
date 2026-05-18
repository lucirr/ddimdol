package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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
			c.Set("role", "central-operator")
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
		sub, role, err := parseJWTClaims(token)
		if err != nil || sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or malformed token"})
			return
		}
		c.Set("user_id", sub)
		c.Set("role", role)
		c.Next()
	}
}

// parseJWTClaims extracts "sub" and "realm_access.roles[0]" (or "role") claims
// from a JWT without verifying its signature. Returns an error on parse failure.
func parseJWTClaims(token string) (sub, role string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("malformed token: expected 3 parts, got %d", len(parts))
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		return "", "", fmt.Errorf("failed to decode token payload: %w", decodeErr)
	}
	var claims map[string]any
	if unmarshalErr := json.Unmarshal(payload, &claims); unmarshalErr != nil {
		return "", "", fmt.Errorf("failed to parse token claims: %w", unmarshalErr)
	}
	sub, _ = claims["sub"].(string)
	// Keycloak puts roles in realm_access.roles; fall back to a top-level "role" claim.
	if realmAccess, ok := claims["realm_access"].(map[string]any); ok {
		if roles, ok := realmAccess["roles"].([]any); ok && len(roles) > 0 {
			role, _ = roles[0].(string)
		}
	}
	if role == "" {
		role, _ = claims["role"].(string)
	}
	return sub, role, nil
}
