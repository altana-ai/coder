# Altana fork rules

This repo is Altana's fork of `coder/coder`. The root `CLAUDE.md` and `AGENTS.md` are upstream's and still apply. This file adds the fork's own rules.

It lives on `release-ci` only. Patch branches are cut from upstream tags and do not carry it, so the monorepo runbook links here.

- **Process**: the monorepo runbook [`docs/coder/admin/coder-fork-upgrade.md`](https://github.com/altana-tech/monorepo/blob/main/docs/coder/admin/coder-fork-upgrade.md) is the one written process, for adding a patch and for upgrading. The branch model and release workflow are in [docs/altana/RELEASE.md](../docs/altana/RELEASE.md). Follow those; do not restate them here.
- **Which branch to patch**: the one prod runs. That is the upstream part of `coder_version` in the monorepo's `apps/terraform/applications/coder/config/production.tfvars`, not the newest release. During an upgrade, the newest branch may only be on dev.
- **Never commit to `main`**, which mirrors upstream.
- **Patch commits**: one per patch, server-side only, each ending in a trailer:
  - `Upstream-Commit: <sha>` for a backport;
  - `Upstream-PR: coder/coder#<n>` for our patch proposed upstream;
  - or `Upstream-PR: none` with the reason in the body.
- **Merging**: open a PR into the patch branch and merge with rebase, never squash.
- **Releasing**: release only after the merge, as the next `v<version>-altana.<n>`. Never reuse a tag.
- **Ledger**: after a patch merges, or an upgrade drops one, update [docs/altana/PATCHES.md](../docs/altana/PATCHES.md) on `release-ci`, recording any drop with its reason.
