package coderd_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/db2sdk"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbgen"
	"github.com/coder/coder/v2/coderd/database/dbtestutil"
	"github.com/coder/coder/v2/coderd/httpmw"
	"github.com/coder/coder/v2/coderd/rbac"
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

func TestUserRouteAllowlistTemplateACL(t *testing.T) {
	t.Parallel()

	partnerID := uuid.New()
	list, err := httpmw.ParseUserRouteAllowlist(`{"` + partnerID.String() + `":` +
		`["GET /api/v2/templates/{[0-9a-f-]{36}}", "POST /api/v2/users/me/workspaces"]}`)
	require.NoError(t, err)

	db, ps := dbtestutil.NewDB(t)
	ownerClient, _, api := coderdtest.NewWithAPI(t, &coderdtest.Options{
		Database: db, Pubsub: ps, UserRouteAllowlist: list, IncludeProvisionerDaemon: true,
	})
	first := coderdtest.CreateFirstUser(t, ownerClient)
	partner := dbgen.User(t, db, database.User{ID: partnerID})
	dbgen.OrganizationMember(t, db, database.OrganizationMember{UserID: partner.ID, OrganizationID: first.OrganizationID})
	_, partnerToken := dbgen.APIKey(t, db, database.APIKey{UserID: partner.ID})
	partnerClient := codersdk.New(ownerClient.URL)
	partnerClient.SetSessionToken(partnerToken)
	memberClient, member := coderdtest.CreateAnotherUser(t, ownerClient, first.OrganizationID)
	group := dbgen.Group(t, db, database.Group{OrganizationID: first.OrganizationID})
	dbgen.GroupMember(t, db, database.GroupMemberTable{GroupID: group.ID, UserID: partner.ID})
	dbgen.GroupMember(t, db, database.GroupMemberTable{GroupID: group.ID, UserID: member.ID})

	newTemplate := func() codersdk.Template {
		version := coderdtest.CreateTemplateVersion(t, ownerClient, first.OrganizationID, nil)
		coderdtest.AwaitTemplateVersionJobCompleted(t, ownerClient, version.ID)
		return coderdtest.CreateTemplate(t, ownerClient, first.OrganizationID, version.ID)
	}
	everyoneOnly, groupOnly, direct := newTemplate(), newTemplate(), newTemplate()
	use := db2sdk.TemplateRoleActions(codersdk.TemplateRoleUse)
	setACL := func(id uuid.UUID, groupACL, userACL database.TemplateACL) {
		//nolint:gocritic // Test setup sets template ACLs directly, as enterprise's ACL route would.
		err := db.UpdateTemplateACLByID(dbauthz.AsSystemRestricted(testutil.Context(t, testutil.WaitShort)), database.UpdateTemplateACLByIDParams{
			ID: id, GroupACL: groupACL, UserACL: userACL,
		})
		require.NoError(t, err)
	}
	setACL(groupOnly.ID, database.TemplateACL{group.ID.String(): use}, database.TemplateACL{})
	setACL(direct.ID, database.TemplateACL{first.OrganizationID.String(): use}, database.TemplateACL{partner.ID.String(): use})

	t.Run("partner reads only the directly granted template", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := partnerClient.Template(ctx, everyoneOnly.ID)
		require.Error(t, err)
		_, err = partnerClient.Template(ctx, groupOnly.ID)
		require.Error(t, err)
		_, err = partnerClient.Template(ctx, direct.ID)
		require.NoError(t, err)
	})

	t.Run("partner builds only from the directly granted template", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := partnerClient.CreateUserWorkspace(ctx, codersdk.Me, codersdk.CreateWorkspaceRequest{TemplateID: everyoneOnly.ID, Name: "everyone"})
		require.Error(t, err)
		_, err = partnerClient.CreateUserWorkspace(ctx, codersdk.Me, codersdk.CreateWorkspaceRequest{TemplateID: direct.ID, Name: "direct"})
		require.NoError(t, err)
	})

	t.Run("unlisted member keeps the Everyone grant", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := memberClient.Template(ctx, everyoneOnly.ID)
		require.NoError(t, err)
		_, err = memberClient.Template(ctx, groupOnly.ID)
		require.NoError(t, err)
		coderdtest.CreateWorkspace(t, memberClient, everyoneOnly.ID)
	})

	t.Run("chat owner subject honours it", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitShort)
		owner, _, err := httpmw.UserRBACSubject(ctx, api.Database, partner.ID, rbac.ScopeAll)
		require.NoError(t, err)
		ownerCtx := dbauthz.As(ctx, owner)

		_, err = api.Database.GetTemplateByID(ownerCtx, everyoneOnly.ID)
		require.True(t, dbauthz.IsNotAuthorizedError(err), "err %v", err)
		_, err = api.Database.GetTemplateByID(ownerCtx, groupOnly.ID)
		require.True(t, dbauthz.IsNotAuthorizedError(err), "err %v", err)
		_, err = api.Database.GetTemplateByID(ownerCtx, direct.ID)
		require.NoError(t, err)

		visible, err := api.Database.GetTemplatesWithFilter(ownerCtx, database.GetTemplatesWithFilterParams{OrganizationID: first.OrganizationID})
		require.NoError(t, err)
		ids := make([]uuid.UUID, 0, len(visible))
		for _, tpl := range visible {
			ids = append(ids, tpl.ID)
		}
		require.Equal(t, []uuid.UUID{direct.ID}, ids)
	})
}

func allowlistStatus(code int) func(int) bool    { return func(got int) bool { return got == code } }
func allowlistNotStatus(code int) func(int) bool { return func(got int) bool { return got != code } }
