---
name: release-state-tool
description: Cut a State Tool beta (RC) or stable release from this repo — branch from master after the previous beta was merged back, write the changelog, open the PR into beta (or beta into release), and walk the post-merge staging → Windows signing → S3 promotion → merge-back steps. Use when asked to "cut a beta", "make a release", "ship an RC", "release the state tool", or "promote beta to release".
---

# Releasing the State Tool

A release follows on from the previous one on `master`:

```
            ┌──────────── merge back after each beta ◀────────────┐
            ▼                                                       │
master ─▶ release branch (master + changelog) ─PR▶ beta ─PR▶ release
          version.txt = X.Y.Z-RCn                 (beta channel)  (stable channel)
```

- When a beta ships, its branch is merged back into `master` (`Merge branch 'version/X-Y-Z-RCn'`).
  `master` then carries the next RC in `version.txt` (e.g. `0.48.2-RC1` once 0.48.1 has gone out).
- Feature PRs merge to `master` after that point. The next release is **`master` as of now**: the
  last merge-back plus everything merged since.
- Branch names use dashes: version `0.48.2-RC1` → `version/0-48-2-RC1`.
- Each RC that ships to beta bumps `n`. An attempt that never reached beta keeps its number.
- Keep unrelated changes, such as edits to this skill, out of the release PR. They go to
  `master` on their own branch.

## Before starting: read the state

```bash
git fetch origin
for b in master beta release; do echo "== $b: $(git show origin/$b:version.txt)"; git log -1 --format='%h %ad %s' --date=short origin/$b; done
git log --first-parent --format='%h %ad %s' --date=short origin/master | head -15   # find the last "Merge branch 'version/...'"
gh pr list --base beta --state open      # stale RC PRs must be closed, not merged by accident
```

What the release ships is the first-parent commits on `master` after the last
`Merge branch 'version/...'`. List them:

```bash
git log --first-parent --oneline <last-merge-back>..origin/master
```

**Don't use `origin/beta..origin/master` to work out what ships.** The RC branches reach `master`
as merges with different hashes, so that range lists dozens of commits that have already shipped.

Confirm with the user which version is being cut (`X.Y.Z-RCn`) and which changes it ships
before you push anything.

## 1. Prepare the release branch

Work in a separate worktree, not someone's checkout. Branch from `master`:

```bash
git worktree add ~/work/cli-X-Y-Z -b <you>/X-Y-Z-RCn origin/master
cd ~/work/cli-X-Y-Z
```

- `version.txt` must read exactly `X.Y.Z-RCn`. If not, fix it in its own commit ("Update version.txt").
- **Changelog:** add a `## X.Y.Z` section above the previous release in `changelog.md`,
  in Keep-a-Changelog form (`### Added / Changed / Fixed / Security`). Write it from the
  bodies of the commits listed above. Describe what users will notice, leave out test and
  CI-only commits, and don't describe internals. There is one section per X.Y.Z; later RCs
  of the same version edit it.
- The release branch holds the `master` commits plus the changelog commit, and nothing else.
- `version/*` branches are protected, and a direct push is rejected with "protected branch hook
  declined". Push the branch under your own name. Whoever has rights can push it as
  `version/X-Y-Z-RCn` if the PR-time branch-channel deploy is wanted (see step 2).

## 2. PR into beta

```bash
gh pr create --base beta --head <you>/X-Y-Z-RCn --title "X.Y.Z-RCn" --body "<changelog section>"
```

`gh pr edit` can fail silently in this repo because of a Projects (classic) error.
To change the title, body or base, use `gh api -X PATCH repos/ActiveState/cli/pulls/<n>`
and read the result back.

Any PR into `beta` runs the full integration suite on all five OS runners
(`.github/workflows/build.yml`). A branch-channel deploy happens at PR time only when the head
branch is named `version/*`. The release doesn't need that deploy, because the push to `beta`
after merging is what stages it. Red integration tests block the release. Look at a failure
before re-running it; some suites are flaky on macOS and Windows. Fixes land on `master` and the
release branch is rebuilt from it.

**Merging this PR is the release decision. Do not merge without the user's go-ahead.**

## 3. After merge: staging

The push to `beta` runs the Deploy job with `DEPLOY_TO_STAGING=true`. That uploads updates and
installers to `s3://state-tool/staging/update/state/…` and the remote installer to
`staging/remote-installer`. Users see nothing yet. Check that the run on `beta` is green:

```bash
gh run list -R ActiveState/cli --branch beta --limit 3
```

## 4. Sign Windows binaries

This is a manual step, and how it is done is not recorded in this repo. Ask the release owner
(historically mitchell-as) and write the procedure here once it is known.

## 5. Promote staging to production

Run this from a checkout of the merged `beta`, with AWS credentials for the `state-tool` bucket:

```bash
state run promote-staging-to-production   # interactive y/N prompt
```

`scripts/ci/s3-promoter` **moves** staged objects to production. How far it reaches depends on
the `promote-staging-to-production` line in that checkout's `activestate.yaml`:

- `... s3-promoter/main.go us-east-1 state-tool update/state` (`master` since 589d443b4) moves
  only `staging/update/state/`. That covers every channel staged there, so if `release` was
  also staged and not yet promoted, it goes live too. `staging/remote-installer` is not moved.
- `... s3-promoter/main.go us-east-1 state-tool` (older branches) moves everything under `staging/`.

Either way, list what is staged before you run it:

```bash
aws s3 ls --recursive s3://state-tool/staging/ | head -50
```

## 6. Verify

```bash
sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -b beta
~/.local/ActiveState/StateTool/beta/bin/state --version     # expect X.Y.Z-RCn
```

An existing install can switch with `state update --set-channel beta`.

## 7. Merge back to master

Merge the released branch back into `master` (`Merge branch 'version/X-Y-Z-RCn'`), so the changelog
and any RC-only fixes are on `master`. Then bump `master`'s `version.txt` to the next RC
(`X.Y.(Z+1)-RC1`, or `X.Y.Z-RC(n+1)` if another RC of the same version is expected). That
merge-back is the starting point for the next release.

## Stable release (beta → release)

Once a beta has soaked, open a PR with `--base release --head beta`, titled `X.Y.Z`. Merging it
stages under `staging/`, and steps 4–6 then run the same way, using `-b release` when you verify.
Close stale `version/*` → `beta` PRs.
