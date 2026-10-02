# Altana Coder fork: how this fork works

This is Altana's fork of `coder/coder`. We use it to **build patched Coder
server binaries** that our infrastructure installs, and to **propose those
patches upstream**. It is a public fork of public code; nothing Altana-specific
lives in the source or the binaries.

The whole design goal is to carry small patches and publish binaries **without
ever blocking our ability to pull in new upstream versions.** That is achieved
by keeping the three concerns on separate branches.

## Branch model

| Branch | What it is | Rule |
| --- | --- | --- |
| `main` | Exact mirror of `coder/coder:main` | **Never commit here.** Fast-forward-sync from upstream only. This is the clean base everything rebases onto. |
| `release-ci` | Repo **default branch**. `main` plus one file: `.github/workflows/build-release.yml` | Holds the build tooling only. GitHub requires a dispatchable workflow to live on the default branch, which is the only reason anything custom sits here. |
| `patch/altana-v<version>` | One upstream release tag (`v<version>`) plus every patch we carry, one commit per patch | What we build. Cut from a `coder/coder` tag. Each commit is also what we PR upstream, so keep them independent. |

Preview/trivy changes ride along in the Coder build via `go.mod` replaces on the
patch branch (pointing at `altana-ai/preview` and `altana-ai/trivy`, which follow
the same `main`-pristine + patch-branch model). Those repos do **not** need their
own release workflow.

## Naming

Tags are `v<upstream version>-altana.<n>`: `v2.35.7-altana.1` is the first Altana build on
upstream v2.35.7, and `n` goes up by one on every rebuild of that base, whatever changed. The
tag does not name patches. The release notes list them, generated from the branch's commits
since the upstream tag. Never bump the upstream patch number, since that would collide with
Coder's own releases.

Releases before 2026-09-28 used `v<version>-param-closure[-customoauth][.n]` and
`patch/param-closure[-customoauth]-v<version>` branches. They stay published for rollback.

## Cutting a release

The workflow is version-agnostic. One branch (`release-ci`), one workflow, N
releases, driven entirely by inputs.

1. Create the patch branch from the upstream tag and apply the change:
   ```
   git fetch upstream                       # upstream = https://github.com/coder/coder.git
   git checkout v2.35.7 -b patch/altana-v2.35.7
   # apply each patch as its own commit
   git commit -am "fix(<area>): <what the patch does>"
   git push origin patch/altana-v2.35.7
   ```
2. Dispatch the build. `--ref` is the branch the *workflow definition* runs from
   (always `release-ci`); `-f ref` is the branch it **checks out and builds**;
   `-f tag` is the release it publishes:
   ```
   gh workflow run build-release.yml --repo altana-ai/coder --ref release-ci \
     -f ref=patch/altana-v2.35.7 \
     -f tag=v2.35.7-altana.1
   ```
3. The workflow builds `linux/arm64` and `linux/amd64`, and publishes a release
   whose assets match the existing convention: a bare binary
   `coder_<tag>_linux_<arch>` plus a `.sha256`.

The client CLI and agent binaries the server serves at `/bin` are **not** built
from the patch branch: the workflow downloads upstream's signed slim binaries
for the tag's base version (`v2.35.7-altana.2` → `2.35.7`) from
`releases.coder.com`, verifies each with `gpgv` against Coder's release key
(`.github/coder-release-signing-key.asc`, fingerprint pinned in the workflow),
and embeds them with their `.asc` signatures. The VS Code/Cursor extension
verifies the CLI it downloads against that key, so a binary we compile always
fails with "Signed digest did not match". This only works while every patch is
server-side; patching CLI, agent, or SSH code would bring the warning back.

Dev and prod can be on different versions: keep one `patch/altana-v<version>` branch per
base, apply a new patch to each, and dispatch once each (e.g. `v2.36.4-altana.1` for dev,
`v2.35.7-altana.1` for prod). `release-ci` never changes between versions.

## How the monorepo consumes a release

`apps/terraform/applications/coder/.../install-coder.sh` (and the workspace-proxy
equivalent) fetch the binary from this fork's releases; the environment's
`coder_version` in tfvars selects the tag (e.g. `2.35.7-altana.1`), and
`coder_provisioner_binary_sha256` pins the arm64 asset's published `.sha256`.

## Pulling in new upstream versions

`main` is a pristine mirror, so it always fast-forwards:

```
git fetch upstream
git checkout main && git merge --ff-only upstream/main && git push origin main
```
(or the GitHub "Sync fork" button.)

`release-ci` has diverged by exactly one isolated file, so it never
fast-forwards, but merging `main` into it is always conflict-free:

```
git checkout release-ci && git merge main && git push origin release-ci
```

`release-ci` rarely needs this: the workflow builds whatever `ref` you pass, not
`release-ci` itself, so it can sit still across upstream releases.

To move to a newer upstream version, cut a fresh branch from the new tag and cherry-pick
the patches upstream has not merged yet, then start that base at `-altana.1`:

```
git checkout v2.37.0 -b patch/altana-v2.37.0
git cherry-pick <patch commits still needed>
```

Because `main` is never polluted and patches are tag-based, upstream rebasing is
never blocked by anything we carry.
