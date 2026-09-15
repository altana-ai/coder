package codersdk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/codersdk"
)

// TestOAuth2ClientRegistrationValidateNativeSchemes covers the opt-in allow-list
// that lets an operator permit a fixed bare custom redirect scheme (e.g. Cursor's
// cursor://) past the default RFC 8252 reverse-domain-notation requirement.
func TestOAuth2ClientRegistrationValidateNativeSchemes(t *testing.T) {
	t.Parallel()

	const cursorURI = "cursor://anysphere.cursor-mcp/oauth/callback"

	newReq := func(uris ...string) *codersdk.OAuth2ClientRegistrationRequest {
		return &codersdk.OAuth2ClientRegistrationRequest{
			ClientName:              "test",
			RedirectURIs:            uris,
			TokenEndpointAuthMethod: codersdk.OAuth2TokenEndpointAuthMethodNone,
		}
	}

	t.Run("BareSchemeRejectedByDefault", func(t *testing.T) {
		t.Parallel()
		err := newReq(cursorURI).Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "reverse domain notation")
	})

	t.Run("BareSchemeAllowedWhenListed", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, newReq(cursorURI).Validate("cursor"))
	})

	t.Run("AllowListIsCaseInsensitive", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, newReq(cursorURI).Validate("Cursor"))
	})

	t.Run("UnlistedBareSchemeStillRejected", func(t *testing.T) {
		t.Parallel()
		err := newReq("vscode://callback").Validate("cursor")
		require.Error(t, err)
		require.Contains(t, err.Error(), "reverse domain notation")
	})

	t.Run("DangerousSchemeNotOverridable", func(t *testing.T) {
		t.Parallel()
		// Allow-listing must not let a browser-dangerous scheme through: the
		// dangerous-scheme block runs before custom-scheme validation.
		err := newReq("javascript://x").Validate("javascript")
		require.Error(t, err)
		require.Contains(t, err.Error(), "dangerous scheme")
	})

	t.Run("ReverseDomainStillAllowedWithoutList", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, newReq("com.example.app://auth/callback").Validate())
	})
}
