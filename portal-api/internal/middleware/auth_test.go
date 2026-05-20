package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func makeToken(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encoded + ".sig"
}

func TestParseJWTClaims(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantSub   string
		wantRole  string
		wantErr   bool
	}{
		{
			name: "valid token with realm_access roles",
			token: makeToken(map[string]any{
				"sub":          "user-123",
				"realm_access": map[string]any{"roles": []any{"central-operator"}},
			}),
			wantSub:  "user-123",
			wantRole: "central-operator",
			wantErr:  false,
		},
		{
			name: "valid token with top-level roles",
			token: makeToken(map[string]any{
				"sub":   "user-456",
				"roles": []any{"admin"},
			}),
			wantSub:  "user-456",
			wantRole: "admin",
			wantErr:  false,
		},
		{
			name: "valid token without role",
			token: makeToken(map[string]any{
				"sub": "user-789",
			}),
			wantSub:  "user-789",
			wantRole: "",
			wantErr:  false,
		},
		{
			name:    "missing sub claim",
			token:   makeToken(map[string]any{"name": "no-sub"}),
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid",
			wantErr: true,
		},
		{
			name:    "empty string",
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sub, role, err := parseJWTClaims(tc.token)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (sub=%q, role=%q)", sub, role)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if sub != tc.wantSub {
				t.Errorf("sub: got %q, want %q", sub, tc.wantSub)
			}
			if role != tc.wantRole {
				t.Errorf("role: got %q, want %q", role, tc.wantRole)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validToken := makeToken(map[string]any{
		"sub":          "user-abc",
		"realm_access": map[string]any{"roles": []any{"operator"}},
	})

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantUserID string
		wantRole   string
	}{
		{
			name:       "missing token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer bad.token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-abc",
			wantRole:   "operator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Auth())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"user_id": c.GetString("user_id"),
					"role":    c.GetString("role"),
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}
