---
name: release-state-tool
description: Cut a State Tool beta (RC) or stable release from this repo — refresh the version/X-Y-Z-RCn branch, write the changelog, open the PR into beta (or beta into release), and walk the post-merge staging → Windows signing → S3 promotion → verification steps. Use when asked to "cut a beta", "make a release", "ship an RC", "release the state tool", or "promote beta to release".
---

# Releasing the State Tool

Releases move along a branch ladder. Each rung is a PR; merging it is what deploys.

```
feature PRs ─▶ master ─▶ version/X-Y-Z-RCn ─PR▶ beta ─PR▶ release
                         (version.txt = X.Y.Z-RCn)  (beta channel)  (stable channel)
```

- `version.txt` holds the version being built. `master` carries the *next* RC
  (e.g. `0.48.2-RC1` once 0.48.1 has shipped).
- Branch names use dashes: version `0.48.2-RC1` → branch `version/0-48-2-RC1`.
- Each RC that ships to beta bumps `n`. A failed attempt that never reached beta keeps its number.

## Before starting: read the state

```bash
git fetch origin
for b in master beta release; do echo "== $b: $(git show origin/$b:version.txt)"; git log -1 --format='%h %ad %s' --date=short origin/$b; done
git branch -r | grep 'version/' | sort -V | tail -5
gh pr list --base beta --state open      # stale RC PRs must be closed, not merged by accident
git log --oneline --no-merges origin/beta..origin/master   # what this release ships
```

Confirm with the user which version is being cut (`X.Y.Z-RCn`) before you push anything.

## 1. Prepare the version branch

Work in a separate worktree, not someone's checkout:

```bash
git worktree add ~/work/cli-X-Y-Z -B version/X-Y-Z-RCn origin/version/X-Y-Z-RCn   # or origin/master if the branch doesn't exist
cd ~/work/cli-X-Y-Z
git merge --no-edit origin/master
```

- `version.txt` must read exactly `X.Y.Z-RCn` (no trailing text). Fix it in its own commit
  ("Update version.txt") if not.
- **Changelog:** add a `## X.Y.Z` section above the previous release in `changelog.md`,
  in Keep-a-Changelog form (`### Added / Changed / Fixed / Security`). Write it from
  the commit bodies of `git log --no-merges origin/beta..HEAD`. Describe the user-visible
  behaviour, not the internals, and leave out test and CI-only commits. One section per
  X.Y.Z. Later RCs of the same version edit that section rather than adding a new one.
- Push the branch.

## 2. PR into beta

```bash
gh pr create --base beta --head version/X-Y-Z-RCn --title "X.Y.Z-RCn" --body "<changelog section>"
```

PRs from `version/*` run the full integration suite on all five OS runners and a
branch-channel deploy (`.github/workflows/build.yml`). Red integration tests block the
release. Look at the failure before re-running; some suites are known to be flaky on macOS and Windows.
Fixes go to `master` (or straight onto the version branch, then merge it back to `master`).

Merging is the release decision. **Do not merge without the user's go-ahead.**

## 3. After merge: staging

The push to `beta` runs the Deploy job with `DEPLOY_TO_STAGING=true`, which uploads
updates and installers to `s3://state-tool/staging/update/state/…` and the remote installer to
`staging/remote-installer`. Users see nothing yet. Check that the run on `beta` is green:

```bash
gh run list --branch beta --limit 3
```

## 4. Sign Windows binaries

This is a manual step, and how to do it is not recorded in this repo. Ask the release owner
(historically mitchell-as) and add the procedure here once it is known.

## 5. Promote staging to production

Run this from a checkout of the merged commit, with AWS credentials for the `state-tool` bucket:

```bash
state run promote-staging-to-production   # interactive y/N prompt
```

`scripts/ci/s3-promoter` **moves** everything under `staging/update/state/` to `update/state/`.
That covers every channel staged there, so if `release` was also staged and not yet promoted,
it goes live too. List the staging prefix first:

```bash
aws s3 ls --recursive s3://state-tool/staging/update/state/ | head
```

The promoter does not touch `staging/remote-installer`. If the remote installer changed,
it has to be promoted separately.

## 6. Verify

```bash
sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -b beta   # or ./installers/install.sh -b beta
~/.local/ActiveState/StateTool/beta/bin/state --version                     # expect X.Y.Z-RCn
```

An existing install can switch with `state update --set-channel beta`.

## 7. Merge back to master

Merge the version branch into `master` (`Merge branch 'version/X-Y-Z-RCn'`) so that
changelog and RC fixes are not lost. Then bump `master`'s `version.txt` to the next RC
if a further RC is expected.

## Stable release (beta → release)

Once a beta has soaked, open a PR with `--base release --head beta`, titled `X.Y.Z`. Merging it
stages under `staging/`, and steps 4–6 then run the same way, with `-b release` when verifying.
Close stale `version/*` → `beta` PRs.
