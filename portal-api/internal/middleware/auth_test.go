package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func makeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encoded + ".fakesig"
}

// --- parseJWTClaims unit tests ---

func TestParseJWTClaims(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		wantSub    string
		wantReason ParseFailReason
		wantErrNil bool
	}{
		{
			name:       "valid JWT with sub",
			token:      makeJWT(map[string]any{"sub": "user-123", "exp": 9999999999}),
			wantSub:    "user-123",
			wantReason: "",
			wantErrNil: true,
		},
		{
			name:       "malformed segments",
			token:      "onlyone",
			wantSub:    "",
			wantReason: MalformedSegments,
			wantErrNil: false,
		},
		{
			name:       "two segments only",
			token:      "header.payload",
			wantSub:    "",
			wantReason: MalformedSegments,
			wantErrNil: false,
		},
		{
			name:       "invalid base64 payload",
			token:      "header.!!!invalid!!!.sig",
			wantSub:    "",
			wantReason: Base64DecodeFailed,
			wantErrNil: false,
		},
		{
			name:       "payload not JSON",
			token:      "header." + base64.RawURLEncoding.EncodeToString([]byte("notjson")) + ".sig",
			wantSub:    "",
			wantReason: JSONUnmarshalFailed,
			wantErrNil: false,
		},
		{
			name:       "sub missing",
			token:      makeJWT(map[string]any{"exp": 9999999999}),
			wantSub:    "",
			wantReason: SubMissing,
			wantErrNil: false,
		},
		{
			name:       "sub empty string",
			token:      makeJWT(map[string]any{"sub": ""}),
			wantSub:    "",
			wantReason: SubMissing,
			wantErrNil: false,
		},
		{
			name:       "empty token",
			token:      "",
			wantSub:    "",
			wantReason: MalformedSegments,
			wantErrNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, reason, err := parseJWTClaims(tt.token)
			if tt.wantErrNil && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.wantErrNil && err == nil {
				t.Errorf("expected error, got nil")
			}
			if sub != tt.wantSub {
				t.Errorf("sub: got %q, want %q", sub, tt.wantSub)
			}
			if reason != tt.wantReason {
				t.Errorf("reason: got %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

// --- Auth middleware integration tests ---

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth())
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestAuthMiddleware_Integration(t *testing.T) {
	os.Unsetenv("DEV_MODE")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid Bearer token → 200",
			authHeader: "Bearer " + makeJWT(map[string]any{"sub": "user-abc"}),
			wantStatus: http.StatusOK,
			wantBody:   `"user_id":"user-abc"`,
		},
		{
			name:       "missing token → 401",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"error":"missing or invalid authorization header"`,
		},
		{
			name:       "malformed JWT → 401 invalid_token",
			authHeader: "Bearer bad.token",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"error":"invalid_token"`,
		},
		{
			name:       "Bearer prefix only → 401",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"error":"missing or invalid authorization header"`,
		},
		{
			name:       "no sub in JWT → 401 invalid_token",
			authHeader: "Bearer " + makeJWT(map[string]any{"exp": 9999999999}),
			wantStatus: http.StatusUnauthorized,
			wantBody:   `"error":"invalid_token"`,
		},
	}

	router := setupRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantBody != "" && !contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body: got %s, want to contain %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestAuthMiddleware_DevMode(t *testing.T) {
	t.Setenv("DEV_MODE", "true")

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DEV_MODE: expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), `"user_id":"dev-user"`) {
		t.Errorf("DEV_MODE: expected dev-user in body, got %s", w.Body.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
