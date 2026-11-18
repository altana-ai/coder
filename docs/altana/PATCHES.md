# Altana Coder fork: patch ledger

Every commit we carry on a `patch/altana-v<version>` branch, and where it came from. An upgrade
starts here: drop every row upstream already ships in the new base, then re-apply the rest.
Branch model and release steps: [RELEASE.md](RELEASE.md).

## Kinds

- **altana**: our own change. Carry it forward until upstream ships an equivalent.
- **backport**: an upstream commit that is newer than our base. Drop it once the new base tag contains it.
- **pin**: a `go.mod` replace pointing at an `altana-ai` fork (`preview`, `trivy`). Re-check it on every upgrade.

## patch/altana-v2.38.10 (release `v2.38.10-altana.1`)

| Commit | Kind | Change | Upstream status |
| --- | --- | --- | --- |
| `5bd143403e` | altana | feat(site): block workspace creation while dynamic parameters are updating | Not upstream |
| `2dbc31467c` | backport | fix: keep in-flight builds from blocking the prebuild hard limit (#30060) | Upstream `main` `5563b7444d`, not in v2.38.10 |
| `5095780ba1` | altana | fix(coderd/database): back off schedule-only prebuild presets | Not upstream |
| `26ec043ab5` | pin | build: pin coder/trivy to the random_* preset convergence fix (`65abc049`) | `altana-ai/trivy` fork |

## patch/altana-v2.37.4 (releases `v2.37.4-altana.1`, `v2.37.4-altana.2`)

| Commit | Kind | Change | On 2.38.10 |
| --- | --- | --- | --- |
| `8c51e94e42` | altana | feat(site): block workspace creation while dynamic parameters are updating | Carried as `5bd143403e` |
| `2336951070` | pin | build: pin coder/preview and coder/trivy to Altana forks (param-closure) | Only the trivy pin is carried; reason for dropping preview not recorded |
| `76913533a6` | altana | feat(oauth2): match authorize/token redirect_uri against any registered URI | Dropped; upstream made `redirect_uris` the source of truth in 2.38 (`b9e3de8403`) |
| `17e03d8e71` | pin | build: pin altana-ai/trivy to stable random_* presets (`86c30051`) | Replaced by `26ec043ab5` |
| `8265d26c29` | altana | fix(coderd/database): back off schedule-only prebuild presets | Carried as `5095780ba1` |
| `1d5e002f40` | altana, since upstreamed | fix(coderd/database): keep in-flight builds from blocking the prebuild hard limit | Carried as backport `2dbc31467c` |
| `d05e0ba00d` | backport | feat: accept PKCE-only token exchange for public clients (#28047) | Dropped; in v2.38.10 (`9be61a0c65`) |

## In review

| PR | Kind | Change | Branch | Notes |
| --- | --- | --- | --- | --- |
| [#3](https://github.com/altana-ai/coder/pull/3) | altana | feat(coderd): confine listed users to an allowlist of routes (DX-5477) | `patch/altana-v2.37.4` | Server-side only. Cherry-picks onto 2.38.10 with one conflict, in `coderd/coderdtest/coderdtest.go`. On 2.38, re-check that Detail's 3 routes still exist: 2.38 keeps `/api/v2/chats` but removes the experimental chat mounts (#29548). |

## Upgrading to a new base

1. Cut `patch/altana-v<new>` from the upstream tag ([RELEASE.md](RELEASE.md)).
2. For each row on the current branch:
   - **backport**: check whether the new tag contains it (`git merge-base --is-ancestor <upstream sha> v<new>`, or search the subject). Drop it if it does.
   - **altana**: check whether upstream shipped an equivalent. If not, cherry-pick it. If the code moved, re-port it by hand and say so in the commit body.
   - **pin**: confirm the fork commit is still needed against the new base.
3. Build and test each patch on the new base before dispatching the release.
4. Add a section for the new branch here, with the outcome for every row of the previous branch.
