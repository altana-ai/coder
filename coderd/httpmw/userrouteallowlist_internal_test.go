package httpmw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/database"
)

func TestRestrictUserRoutes(t *testing.T) {
	t.Parallel()

	partner := uuid.New()
	other := uuid.New()
	list, err := ParseUserRouteAllowlist(`{"` + partner.String() + `":` +
		`["GET /api/v2/users/me", "GET /api/v2/users/me/keys/{[A-Za-z0-9]{10}}", "POST /api/v2/chats"]}`)
	require.NoError(t, err)

	serve := func(userID *uuid.UUID, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		if userID != nil {
			req = req.WithContext(context.WithValue(req.Context(), apiKeyPrecheckedContextKey{}, APIKeyPrechecked{
				Result: &ValidateAPIKeyResult{Key: database.APIKey{UserID: *userID}},
			}))
		}
		rw := httptest.NewRecorder()
		RestrictUserRoutes(list)(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
		})).ServeHTTP(rw, req)
		return rw.Code
	}

	for _, tc := range []struct {
		name   string
		user   *uuid.UUID
		method string
		path   string
		want   int
	}{
		{"partner own user", &partner, http.MethodGet, "/api/v2/users/me", http.StatusOK},
		{"partner own key", &partner, http.MethodGet, "/api/v2/users/me/keys/abcde12345", http.StatusOK},
		{"partner creates chat", &partner, http.MethodPost, "/api/v2/chats", http.StatusOK},
		{"partner reads chat messages", &partner, http.MethodGet, "/api/v2/chats/" + uuid.NewString() + "/messages", http.StatusForbidden},
		{"partner lists chats", &partner, http.MethodGet, "/api/v2/chats", http.StatusForbidden},
		{"partner calls v2 MCP", &partner, http.MethodPost, "/api/v2/mcp/http", http.StatusForbidden},
		{"partner calls experimental MCP", &partner, http.MethodPost, "/api/experimental/mcp/http", http.StatusForbidden},
		{"partner lists own tokens", &partner, http.MethodGet, "/api/v2/users/me/keys/tokens", http.StatusForbidden},
		{"partner key segment is one segment", &partner, http.MethodGet, "/api/v2/users/me/keys/abcde12345/x", http.StatusForbidden},
		{"partner key segment must match", &partner, http.MethodGet, "/api/v2/users/me/keys/abcde-1234", http.StatusForbidden},
		{"partner key segment needs a value", &partner, http.MethodGet, "/api/v2/users/me/keys/", http.StatusForbidden},
		{"partner wrong method", &partner, http.MethodDelete, "/api/v2/users/me", http.StatusForbidden},
		{"other user unaffected", &other, http.MethodGet, "/api/v2/chats/" + uuid.NewString() + "/messages", http.StatusOK},
		{"unauthenticated unaffected", nil, http.MethodPost, "/api/experimental/mcp/http", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, serve(tc.user, tc.method, tc.path))
		})
	}
}

func TestParseUserRouteAllowlist(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	empty, err := ParseUserRouteAllowlist("")
	require.NoError(t, err)
	require.Empty(t, empty)

	for _, raw := range []string{
		`not json`,
		`{"not-a-uuid": ["GET /api/v2/users/me"]}`,
		`{"` + id + `": []}`,
		`{"` + id + `": ["/api/v2/users/me"]}`,
		`{"` + id + `": ["GET api/v2/users/me"]}`,
		`{"` + id + `": ["GET /api/*/users"]}`,
		`{"` + id + `": ["GET /api/v2/users/me/keys/*"]}`,
		`{"` + id + `": ["GET /api/{v2}/users"]}`,
		`{"` + id + `": ["GET /api/v2/keys/{[a-z}"]}`,
	} {
		_, err := ParseUserRouteAllowlist(raw)
		require.Error(t, err, raw)
	}
}
