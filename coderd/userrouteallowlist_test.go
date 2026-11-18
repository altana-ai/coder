package coderd_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/testutil"
)

func TestUserRouteAllowlist(t *testing.T) {
	t.Parallel()

	partnerID := uuid.New()
	list, err := httpmw.ParseUserRouteAllowlist(`{"` + partnerID.String() + `":` +
		`["GET /api/v2/users/me", "GET /api/v2/users/me/keys/{[A-Za-z0-9]{10}}", "POST /api/v2/chats"]}`)
	require.NoError(t, err)

	ownerClient, db := coderdtest.NewWithDatabase(t, &coderdtest.Options{UserRouteAllowlist: list})
	coderdtest.CreateFirstUser(t, ownerClient)
	partner := dbgen.User(t, db, database.User{ID: partnerID})
	partnerKey, partnerToken := dbgen.APIKey(t, db, database.APIKey{UserID: partner.ID})

	send := func(t *testing.T, token, method, path string, cookie *http.Cookie) int {
		t.Helper()
		ctx := testutil.Context(t, testutil.WaitShort)
		req, err := http.NewRequestWithContext(ctx, method, ownerClient.URL.String()+path, nil)
		require.NoError(t, err)
		if cookie != nil {
			req.AddCookie(cookie)
		} else {
			req.Header.Set(codersdk.SessionTokenHeader, token)
		}
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, err := client.Do(req)
		require.NoError(t, err)
		_ = res.Body.Close()
		return res.StatusCode
	}
	appCookie := &http.Cookie{Name: codersdk.PathAppSessionTokenCookie, Value: partnerToken}

	for _, tc := range []struct {
		name   string
		token  string
		method string
		path   string
		cookie *http.Cookie
		want   func(int) bool
	}{
		{"partner reads itself", partnerToken, http.MethodGet, "/api/v2/users/me", nil, allowlistStatus(http.StatusOK)},
		{"partner reads its key", partnerToken, http.MethodGet, "/api/v2/users/me/keys/" + partnerKey.ID, nil, allowlistStatus(http.StatusOK)},
		{"partner lists its tokens", partnerToken, http.MethodGet, "/api/v2/users/me/keys/tokens", nil, allowlistStatus(http.StatusForbidden)},
		{"partner lists chats", partnerToken, http.MethodGet, "/api/v2/chats", nil, allowlistStatus(http.StatusForbidden)},
		{"partner reads chat messages", partnerToken, http.MethodGet, "/api/v2/chats/" + uuid.NewString() + "/messages", nil, allowlistStatus(http.StatusForbidden)},
		{"partner calls MCP", partnerToken, http.MethodPost, "/api/experimental/mcp/http", nil, allowlistStatus(http.StatusForbidden)},
		{"partner opens a path app by cookie", "", http.MethodGet, "/@" + partner.Username + "/ws/apps/app/", appCookie, allowlistStatus(http.StatusForbidden)},
		{"owner lists chats", ownerClient.SessionToken(), http.MethodGet, "/api/v2/chats", nil, allowlistNotStatus(http.StatusForbidden)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := send(t, tc.token, tc.method, tc.path, tc.cookie)
			require.True(t, tc.want(got), "status %d", got)
		})
	}
}

func allowlistStatus(code int) func(int) bool    { return func(got int) bool { return got == code } }
func allowlistNotStatus(code int) func(int) bool { return func(got int) bool { return got != code } }
