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
| `patch/<change>-v<version>` | One upstream release tag (`v<version>`) plus the change | The unit of work. Cut from a `coder/coder` tag, carries only the patch. This is what we build **and** what we PR upstream. |

Preview/trivy changes ride along in the Coder build via `go.mod` replaces on the
patch branch (pointing at `altana-ai/preview` and `altana-ai/trivy`, which follow
the same `main`-pristine + patch-branch model). Those repos do **not** need their
own release workflow.

## Cutting a release

The workflow is version-agnostic. One branch (`release-ci`), one workflow, N
releases, driven entirely by inputs.

1. Create the patch branch from the upstream tag and apply the change:
   ```
   git fetch upstream                       # upstream = https://github.com/coder/coder.git
   git checkout v2.35.4 -b patch/param-closure-v2.35.4
   # apply the change (e.g. go.mod replaces to the altana-ai preview/trivy forks)
   git commit -am "build: <what the patch does>"
   git push origin patch/param-closure-v2.35.4
   ```
2. Dispatch the build. `--ref` is the branch the *workflow definition* runs from
   (always `release-ci`); `-f ref` is the branch it **checks out and builds**;
   `-f tag` is the release it publishes:
   ```
   gh workflow run build-release.yml --repo altana-ai/coder --ref release-ci \
     -f ref=patch/param-closure-v2.35.4 \
     -f tag=v2.35.4-param-closure
   ```
3. The workflow builds `linux/arm64` and `linux/amd64`, and publishes a release
   whose assets match the existing convention: a bare binary
   `coder_<tag>_linux_<arch>` plus a `.sha256`.

Dev and prod can be on different versions: just cut one patch branch per version
and dispatch once each (e.g. `v2.36.1-param-closure` for dev,
`v2.35.4-param-closure` for prod). `release-ci` never changes between versions.

## How the monorepo consumes a release

`apps/terraform/applications/coder/.../install-coder.sh` (and the workspace-proxy
equivalent) fetch the binary from this fork's releases; the environment's
`coder_version` in tfvars selects the tag (e.g. `2.35.4-param-closure`).

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

To move a patch to a newer upstream version, cut a fresh patch branch from the
new tag and cherry-pick the change:

```
git checkout v2.37.0 -b patch/param-closure-v2.37.0
git cherry-pick <old patch commit>
```

Because `main` is never polluted and patches are tag-based, upstream rebasing is
never blocked by anything we carry.
