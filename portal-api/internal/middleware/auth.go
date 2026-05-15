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
		sub := parseJWTSub(token)
		if sub == "" {
			sub = "unknown"
		}
		c.Set("user_id", sub)
		c.Next()
	}
}

// parseJWTSub extracts the "sub" claim from a JWT without verifying its signature.
// Returns empty string on any parse failure.
func parseJWTSub(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return ""
	}
	// JWT is always header.payload.signature — payload is the second-to-last part
	payload, err := base64.RawURLEncoding.DecodeString(parts[len(parts)-2])
	if err != nil {
		// Try standard encoding as fallback
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}
