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

type ParseFailReason string

const (
	MalformedSegments   ParseFailReason = "malformed_segments"
	Base64DecodeFailed  ParseFailReason = "base64_decode_failed"
	JSONUnmarshalFailed ParseFailReason = "json_unmarshal_failed"
	SubMissing          ParseFailReason = "sub_missing"
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
				authLogger.Warn("DEV_MODE enabled: authentication is bypassed. DO NOT USE IN PRODUCTION.")
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
		sub, reason, err := parseJWTClaims(token)
		if err != nil {
			authLogger.Warn("jwt parse failed",
				zap.String("reason", string(reason)),
				zap.String("path", c.Request.URL.Path),
				zap.String("remote_ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		c.Set("user_id", sub)
		c.Next()
	}
}

// parseJWTClaims extracts the "sub" claim from a JWT without verifying its signature.
// TODO(PR-2): JWKS 기반 서명 검증 후 realm_access.roles 추출 예정.
func parseJWTClaims(token string) (sub string, reason ParseFailReason, err error) {
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		return "", MalformedSegments, fmt.Errorf("token has %d parts, need at least 3", len(parts))
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		return "", Base64DecodeFailed, fmt.Errorf("base64 decode failed: %w", decodeErr)
	}
	var claims map[string]any
	if jsonErr := json.Unmarshal(payload, &claims); jsonErr != nil {
		return "", JSONUnmarshalFailed, fmt.Errorf("json unmarshal failed: %w", jsonErr)
	}
	subVal, ok := claims["sub"].(string)
	if !ok || subVal == "" {
		return "", SubMissing, fmt.Errorf("sub claim missing or not a string")
	}
	return subVal, "", nil
}
