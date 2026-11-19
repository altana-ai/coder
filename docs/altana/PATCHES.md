# Altana Coder fork: patch ledger

What each `patch/altana-v<version>` branch carries, and why each dropped patch was dropped. A
dropped commit never appears in `git log` on the next branch, so this file is the only place its
reason survives.

The facts an agent needs to carry or drop a patch come from the patch commit's trailer
(`Upstream-Commit: <sha>`, `Upstream-PR: coder/coder#<n>`, or `Upstream-PR: none` with a reason).
This file is the human summary and the drop history. The upgrade process is written once, in the
monorepo runbook [`docs/coder/admin/coder-fork-upgrade.md`](https://github.com/altana-tech/monorepo/blob/main/docs/coder/admin/coder-fork-upgrade.md).

Rows are keyed on the commit subject and its upstream reference. Rebase-merge rewrites commits,
so add a row's fork SHA only after its PR merges.

## patch/altana-v2.39.4 (no release yet)

Cut from upstream `v2.39.4` for DX-5488. Fork SHAs are added after the work-branch PR merges.

| Change | Kind | Upstream | On 2.39.4 |
| --- | --- | --- | --- |
| feat(site): block workspace creation while dynamic parameters are updating | altana | `Upstream-PR: coder/coder#30061`, closed unmerged by the stale bot. Trailer added on this branch. | Carried |
| fix: keep in-flight builds from blocking the prebuild hard limit | backport | `Upstream-Commit: 5563b7444d` (coder/coder#30060), on upstream `main` but not in v2.39.4 | Carried, cherry-picked from the upstream commit |
| fix(coderd/database): back off schedule-only prebuild presets | altana | `Upstream-PR: coder/coder#30059`, closed unmerged by the stale bot. Trailer added on this branch. | Carried |
| build: pin coder/trivy to the random_* preset convergence fix | pin | `github.com/coder/trivy` `65abc049` | Dropped. v2.39.4's `go.mod` pins `coder/trivy` `65abc04915bd` itself. |
| feat(coderd): confine listed users to an allowlist of routes | altana | `Upstream-PR: none`. Altana-specific (DX-5477). | Carried |

## patch/altana-v2.38.10 (releases `v2.38.10-altana.1`, `v2.38.10-altana.2`)

| Change | Kind | Upstream | Fork SHA |
| --- | --- | --- | --- |
| feat(site): block workspace creation while dynamic parameters are updating | altana | Proposed as coder/coder#30061, closed unmerged. The commit has no trailer. | `5bd143403e` |
| fix: keep in-flight builds from blocking the prebuild hard limit | backport | `Upstream-Commit: 5563b7444d` (coder/coder#30060), on upstream `main` but not in v2.38.10 | `2dbc31467c` |
| fix(coderd/database): back off schedule-only prebuild presets | altana | Proposed as coder/coder#30059, closed unmerged. The commit has no trailer. | `5095780ba1` |
| build: pin coder/trivy to the random_* preset convergence fix | pin | `github.com/coder/trivy` `65abc049`, Coder's own trivy fork, newer than v2.38.10's `15b94953`. Drop once the base pins `65abc049` or later. | `26ec043ab5` |
| feat(coderd): confine listed users to an allowlist of routes | altana | `Upstream-PR: none`. Altana-specific: confines a partner service account by user ID (DX-5477). Added in `v2.38.10-altana.2`. | `319d2dacf9` |
| feat(coderd): ignore template group ACLs for allowlisted users | altana | `Upstream-PR: none`. Altana-specific: a listed user reaches a template only through a direct user ACL entry, not a group such as Everyone (DX-5477). Not yet in a release. | `441dae40ab` |

## patch/altana-v2.37.4 (releases `v2.37.4-altana.1`, `v2.37.4-altana.2`)

| Change | Kind | On 2.38.10 |
| --- | --- | --- |
| feat(site): block workspace creation while dynamic parameters are updating | altana | Carried |
| build: pin coder/preview and coder/trivy to Altana forks (param-closure) | pin | Dropped. Upstream merged our param-closure fix as coder/preview#226 (`c0dfdeecbd89`), and v2.38.10 pins exactly that commit. The trivy half is replaced by the `coder/trivy` pin. |
| feat(oauth2): match authorize/token redirect_uri against any registered URI | altana | Dropped. 2.38 matches against the registered `redirect_uris`. Accepted behaviour change: 2.38 allows any port on a registered loopback redirect URI (RFC 8252), which our patch did not. |
| build: pin altana-ai/trivy to stable random_* presets (`86c30051`) | pin | Dropped. The fix is in `coder/trivy` `65abc049`, pinned on 2.38.10. |
| fix(coderd/database): back off schedule-only prebuild presets | altana | Carried |
| fix(coderd/database): keep in-flight builds from blocking the prebuild hard limit | altana, since upstreamed | Carried as a backport of the upstream commit (coder/coder#30060) |
| feat: accept PKCE-only token exchange for public clients (coder/coder#28047) | backport | Dropped. In v2.38.10 (`9be61a0c65`). |
