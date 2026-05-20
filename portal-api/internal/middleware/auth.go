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
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", sub)
		c.Set("role", role)
		c.Next()
	}
}

// parseJWTClaims extracts sub and role claims from a JWT without verifying its signature.
func parseJWTClaims(token string) (sub string, role string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("malformed token")
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		payload, decodeErr = base64.StdEncoding.DecodeString(parts[1])
		if decodeErr != nil {
			return "", "", fmt.Errorf("failed to decode token payload: %w", decodeErr)
		}
	}
	var claims map[string]any
	if unmarshalErr := json.Unmarshal(payload, &claims); unmarshalErr != nil {
		return "", "", fmt.Errorf("failed to parse token claims: %w", unmarshalErr)
	}
	sub, _ = claims["sub"].(string)
	if sub == "" {
		return "", "", fmt.Errorf("missing sub claim")
	}
	role = extractRole(claims)
	return sub, role, nil
}

// extractRole pulls the first role from realm_access.roles or the top-level roles array.
func extractRole(claims map[string]any) string {
	if ra, ok := claims["realm_access"].(map[string]any); ok {
		if roles, ok := ra["roles"].([]any); ok && len(roles) > 0 {
			if r, ok := roles[0].(string); ok {
				return r
			}
		}
	}
	if roles, ok := claims["roles"].([]any); ok && len(roles) > 0 {
		if r, ok := roles[0].(string); ok {
			return r
		}
	}
	return ""
}
