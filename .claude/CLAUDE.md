# Altana fork rules

This repo is Altana's fork of `coder/coder`. The root `CLAUDE.md` and `AGENTS.md` are upstream's and still apply. This file adds the fork's own rules. It lives on `release-ci` only; patch branches are cut from upstream tags and do not carry it.

- **Branches**: read [docs/altana/RELEASE.md](../docs/altana/RELEASE.md) before any change. Never commit to `main`, which mirrors upstream.
  - Patches go on `patch/altana-v<version>`, one commit per patch.
  - `release-ci` holds only the build workflow and these docs.
- **Which branch to patch**: the newest `patch/altana-v<version>` (check `gh release list --repo altana-ai/coder`). Port to an older branch only for an environment still running it.
- **Reviewing a patch**: open a pull request against the patch branch, so the patch is reviewed as a single commit.
- **Patch ledger**: every pull request that adds, ports or drops a patch also updates [docs/altana/PATCHES.md](../docs/altana/PATCHES.md), through its own pull request to `release-ci`.
  - Record the commit, its kind (altana, backport or pin), and its upstream status.
  - An upgrade starts from the ledger. Drop backports the new upstream tag already contains, and re-port Altana patches by hand where upstream code has moved.
- **Releases**: built by the `build-release` GitHub workflow on `release-ci`, dispatched by hand with `-f ref=<patch branch> -f tag=v<version>-altana.<n>`. A merge publishes nothing.
  - Patches must stay server-side. The signed client binaries come from upstream, so a client or agent patch breaks the VS Code extension's signature check.
- **Consumers**: the monorepo installs a release by `coder_version` in `apps/terraform/applications/coder/config/<env>.tfvars`. A release reaches prod only through a monorepo pull request.
