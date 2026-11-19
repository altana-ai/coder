package httpmw

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/codersdk"
)

// UserRouteAllowlistEnv holds a JSON object mapping a user ID to the only
// "METHOD /path" routes that user may call. A final "/{regex}" segment matches
// one path segment against the anchored regex. Listed users may never use workspace apps.
const UserRouteAllowlistEnv = "CODER_ALTANA_USER_ROUTE_ALLOWLIST"

// UserRouteRefusal is the message a listed user gets for any route outside their allowlist.
const UserRouteRefusal = "This account may not call this route."

type userRoute struct {
	method  string
	path    string
	segment *regexp.Regexp
}

// UserRouteAllowlist confines listed users to their routes. Other users are unaffected.
type UserRouteAllowlist map[uuid.UUID][]userRoute

// Listed reports whether the user is confined by the allowlist.
func (l UserRouteAllowlist) Listed(userID uuid.UUID) bool {
	_, ok := l[userID]
	return ok
}

// ParseUserRouteAllowlist parses the UserRouteAllowlistEnv value. An empty value yields an empty allowlist.
func ParseUserRouteAllowlist(raw string) (UserRouteAllowlist, error) {
	list := UserRouteAllowlist{}
	if strings.TrimSpace(raw) == "" {
		return list, nil
	}
	var parsed map[string][]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, xerrors.Errorf("parse %s: %w", UserRouteAllowlistEnv, err)
	}
	for userID, routes := range parsed {
		id, err := uuid.Parse(userID)
		if err != nil {
			return nil, xerrors.Errorf("parse %s: user id %q: %w", UserRouteAllowlistEnv, userID, err)
		}
		if len(routes) == 0 {
			return nil, xerrors.Errorf("parse %s: user %s has no routes", UserRouteAllowlistEnv, id)
		}
		for _, route := range routes {
			rule, err := parseUserRoute(route)
			if err != nil {
				return nil, xerrors.Errorf("parse %s: %w", UserRouteAllowlistEnv, err)
			}
			list[id] = append(list[id], rule)
		}
	}
	return list, nil
}

func parseUserRoute(route string) (userRoute, error) {
	method, path, ok := strings.Cut(strings.TrimSpace(route), " ")
	if !ok || method == "" || !strings.HasPrefix(path, "/") || strings.Contains(path, " ") {
		return userRoute{}, xerrors.Errorf("route %q is not \"METHOD /path\"", route)
	}
	rule := userRoute{method: strings.ToUpper(method), path: path}
	if i := strings.LastIndex(path, "/{"); i >= 0 && strings.HasSuffix(path, "}") {
		re, err := regexp.Compile("^(?:" + path[i+2:len(path)-1] + ")$")
		if err != nil {
			return userRoute{}, xerrors.Errorf("route %q: segment pattern: %w", route, err)
		}
		rule.path, rule.segment = path[:i+1], re
	}
	if strings.ContainsAny(rule.path, "*{}") {
		return userRoute{}, xerrors.Errorf("route %q may only end in one /{regex} segment", route)
	}
	return rule, nil
}

func (rule userRoute) matches(r *http.Request) bool {
	if r.Method != rule.method {
		return false
	}
	if rule.segment == nil {
		return r.URL.Path == rule.path
	}
	segment, ok := strings.CutPrefix(r.URL.Path, rule.path)
	return ok && !strings.Contains(segment, "/") && rule.segment.MatchString(segment)
}

// RestrictUserRoutes returns 403 when a listed user calls a route outside their allowlist.
// It must run after PrecheckAPIKey and before any routing. Workspace-app cookies skip the
// precheck, so the workspace-app token provider refuses listed users separately.
func RestrictUserRoutes(list UserRouteAllowlist) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(list) == 0 {
			return next
		}
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			pc, ok := r.Context().Value(apiKeyPrecheckedContextKey{}).(APIKeyPrechecked)
			if !ok || pc.Result == nil {
				next.ServeHTTP(rw, r)
				return
			}
			routes, listed := list[pc.Result.Key.UserID]
			if !listed {
				next.ServeHTTP(rw, r)
				return
			}
			for _, rule := range routes {
				if rule.matches(r) {
					next.ServeHTTP(rw, r)
					return
				}
			}
			httpapi.Write(r.Context(), rw, http.StatusForbidden, codersdk.Response{Message: UserRouteRefusal})
		})
	}
}
