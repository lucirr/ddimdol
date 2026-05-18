package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func makeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encoded + ".fakesig"
}

func TestParseJWTClaims(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantSub  string
		wantRole string
		wantErr  bool
	}{
		{
			name:     "valid JWT with sub and realm_access role",
			token:    makeJWT(map[string]any{"sub": "user-123", "realm_access": map[string]any{"roles": []any{"central-operator"}}}),
			wantSub:  "user-123",
			wantRole: "central-operator",
			wantErr:  false,
		},
		{
			name:     "valid JWT with sub and top-level role",
			token:    makeJWT(map[string]any{"sub": "user-456", "role": "viewer"}),
			wantSub:  "user-456",
			wantRole: "viewer",
			wantErr:  false,
		},
		{
			name:     "valid JWT with sub only, no role",
			token:    makeJWT(map[string]any{"sub": "user-789"}),
			wantSub:  "user-789",
			wantRole: "",
			wantErr:  false,
		},
		{
			name:    "malformed token (missing parts)",
			token:   "onlyone",
			wantErr: true,
		},
		{
			name:    "invalid base64 payload",
			token:   "header.!!!invalid!!!.sig",
			wantErr: true,
		},
		{
			name:    "payload is not JSON",
			token:   "header." + base64.RawURLEncoding.EncodeToString([]byte("notjson")) + ".sig",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, role, err := parseJWTClaims(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (sub=%q role=%q)", sub, role)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if sub != tt.wantSub {
				t.Errorf("sub: got %q, want %q", sub, tt.wantSub)
			}
			if role != tt.wantRole {
				t.Errorf("role: got %q, want %q", role, tt.wantRole)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validToken := makeJWT(map[string]any{"sub": "user-123", "role": "admin"})

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantUserID     string
		wantRole       string
	}{
		{
			name:       "valid Bearer token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-123",
			wantRole:   "admin",
		},
		{
			name:       "missing token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token",
			authHeader: "Bearer notavalidjwt",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer prefix only",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			var gotUserID, gotRole string
			router.GET("/test", Auth(), func(ctx *gin.Context) {
				gotUserID = ctx.GetString("user_id")
				gotRole = ctx.GetString("role")
				ctx.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			c.Request = req
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				if gotUserID != tt.wantUserID {
					t.Errorf("user_id: got %q, want %q", gotUserID, tt.wantUserID)
				}
				if gotRole != tt.wantRole {
					t.Errorf("role: got %q, want %q", gotRole, tt.wantRole)
				}
			}
		})
	}
}
