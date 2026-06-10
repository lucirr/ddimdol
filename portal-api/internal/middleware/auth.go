package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	devModeWarnOnce sync.Once
	authLogger      *zap.Logger

	verifierOnce sync.Once
	verifier     *oidc.IDTokenVerifier
	verifierErr  error
)

func init() {
	authLogger, _ = zap.NewProduction()
}

// initVerifier lazily creates the OIDC token verifier.
// Required env vars:
//   KEYCLOAK_ISSUER   — https://keycloak.host/realms/<realm>
//   KEYCLOAK_AUDIENCE — client ID that portal-api expects in the "aud" claim
func initVerifier() (*oidc.IDTokenVerifier, error) {
	verifierOnce.Do(func() {
		issuer := os.Getenv("KEYCLOAK_ISSUER")
		if issuer == "" {
			verifierErr = fmt.Errorf("KEYCLOAK_ISSUER is not set")
			return
		}
		audience := os.Getenv("KEYCLOAK_AUDIENCE")
		if audience == "" {
			verifierErr = fmt.Errorf("KEYCLOAK_AUDIENCE is not set")
			return
		}
		provider, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			verifierErr = err
			return
		}
		verifier = provider.Verifier(&oidc.Config{ClientID: audience})
	})
	return verifier, verifierErr
}

// Auth validates Keycloak OIDC JWT tokens via JWKS signature verification.
// Falls back to DEV_MODE bypass when DEV_MODE=true.
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

		rawToken := extractBearerToken(c)
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		v, err := initVerifier()
		if err != nil {
			authLogger.Error("OIDC provider init failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth provider unavailable"})
			return
		}
		if v == nil {
			authLogger.Error("KEYCLOAK_ISSUER not configured")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth provider not configured"})
			return
		}

		idToken, err := v.Verify(c.Request.Context(), rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Extract custom claims from verified token.
		var extra struct {
			Sub          string            `json:"sub"`
			RealmAccess  map[string][]string `json:"realm_access"`
			TenantID     string            `json:"tenant_id"`
		}
		if err := idToken.Claims(&extra); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "failed to parse token claims"})
			return
		}

		sub := extra.Sub
		if sub == "" {
			sub = "unknown"
		}
		c.Set("user_id", sub)
		c.Set("roles", extra.RealmAccess["roles"])
		if extra.TenantID != "" {
			c.Set("tenant_id", extra.TenantID)
		}

		c.Next()
	}
}

// AgentMTLSIdentity extracts the edge ID from the mTLS client certificate CN
// and stores it in the gin context as "mtls_edge_id". Call this middleware on
// the Agent API router (port :8081) where mTLS is required.
//
// DEV_MODE bypass is allowed only when AGENT_TLS_ENABLED is not "true".
// When TLS is enabled in production, the bypass is blocked regardless of DEV_MODE.
func AgentMTLSIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		tlsEnabled := os.Getenv("AGENT_TLS_ENABLED") == "true"

		if os.Getenv("DEV_MODE") == "true" && !tlsEnabled {
			eid := c.GetHeader("X-Edge-ID")
			if eid == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Edge-ID header required in DEV_MODE"})
				return
			}
			c.Set("mtls_edge_id", eid)
			c.Next()
			return
		}

		if c.Request.TLS == nil || len(c.Request.TLS.VerifiedChains) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "mTLS client certificate required"})
			return
		}

		// The leaf certificate CN carries the edge UUID.
		cn := c.Request.TLS.VerifiedChains[0][0].Subject.CommonName
		if cn == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "client certificate CN is empty"})
			return
		}
		c.Set("mtls_edge_id", cn)
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

func extractBearerToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return c.Query("access_token")
}
