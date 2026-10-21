---
name: release-state-tool
description: Cut a State Tool beta (RC) or stable release from this repo — refresh the version/X-Y-Z-RCn branch, write the changelog, open the PR into beta (or beta into release), and walk the post-merge staging → Windows signing → S3 promotion → verification steps. Use when asked to "cut a beta", "make a release", "ship an RC", "release the state tool", or "promote beta to release".
---

# Releasing the State Tool

Releases move along a branch ladder. Each rung is a PR; merging it is what deploys.

```
                beta ─▶ version/X-Y-Z-RCn ─PR▶ beta ─PR▶ release
master ─cherry-pick─▶   (version.txt = X.Y.Z-RCn)  (beta channel)  (stable channel)
```

- **A release is cut from `beta`, not from `master`.** `master` and `beta` have diverged:
  `master` carries work that was never released (much of it CI and test churn, some of it
  behaviour changes), plus merge-backs of every RC branch. So **never merge `master` into a
  version branch.** That pulls in all of that unreleased work, and the PR shows close to a
  hundred commits. Pick the specific changes this release ships from `master`. Each feature PR
  lands on `master` as a single squash commit, so `git cherry-pick <sha>` works.
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
# candidates: master commits since the last release. Hashes differ between the branches,
# so match on subject; anything already in changelog.md's last section has shipped.
git log --oneline --no-merges --since="<date beta was last cut>" origin/master
```

Confirm two things with the user before you push anything: the version being cut
(`X.Y.Z-RCn`) and **the list of master commits it ships**.

## 1. Prepare the release branch

Work in a separate worktree, not someone's checkout. Start from `beta`. If a
`version/X-Y-Z-RCn` branch already exists, check that it is `beta` plus a `version.txt` bump
(`git log --oneline origin/beta..origin/version/X-Y-Z-RCn`):

```bash
git worktree add ~/work/cli-X-Y-Z -b <you>/release-X-Y-Z origin/beta
cd ~/work/cli-X-Y-Z
git cherry-pick <sha> <sha> ...      # the agreed master commits, oldest first
git diff --stat origin/beta HEAD     # must show only those changes, the changelog and version.txt
```

- `version.txt` must read exactly `X.Y.Z-RCn` (no trailing text). Fix it in its own commit
  ("Update version.txt") if not.
- **Changelog:** add a `## X.Y.Z` section above the previous release in `changelog.md`,
  in Keep-a-Changelog form (`### Added / Changed / Fixed / Security`). Write it from
  the commit bodies of `git log --no-merges origin/beta..HEAD`. Describe the user-visible
  behaviour, not the internals, and leave out test and CI-only commits. One section per
  X.Y.Z. Later RCs of the same version edit that section rather than adding a new one.
- `version/*` branches are protected, and a direct push is rejected with "protected branch hook
  declined". Push the personal branch and open the PR from it.

## 2. PR into beta

```bash
gh pr create --base beta --head <you>/release-X-Y-Z --title "X.Y.Z-RCn" --body "<changelog section>"
```

Any PR targeting `beta` runs the full integration suite on all five OS runners
(`.github/workflows/build.yml`). The PR-time branch-channel deploy only happens when the head
branch is named `version/*`, so a personal head branch skips it. That deploy isn't needed for
the release, because the push to `beta` after merge is what stages it. Red integration tests
block the release. Look at the failure before re-running; some suites are known to be flaky on
macOS and Windows. Land fixes on `master` first and cherry-pick them, so the two branches don't
drift further apart.

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

`scripts/ci/s3-promoter` **moves** staged objects to production, and how much it moves depends
on the branch you run it from. Read the `promote-staging-to-production` line in
`activestate.yaml` first:

- `... s3-promoter/main.go us-east-1 state-tool` (`beta`/`release` as of 0.48.x) moves
  **everything** under `staging/`. That includes the remote installer, and any other channel that
  was staged and not yet promoted.
- `... state-tool update/state` (master's fix 589d443b4, not yet released) moves only
  `staging/update/state/`. `staging/remote-installer` then has to be promoted separately.

Either way, list what is staged before running it:

```bash
aws s3 ls --recursive s3://state-tool/staging/ | head -50
```

## 6. Verify

```bash
sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -b beta   # or ./installers/install.sh -b beta
~/.local/ActiveState/StateTool/beta/bin/state --version                     # expect X.Y.Z-RCn
```

An existing install can switch with `state update --set-channel beta`.

## 7. Merge back to master

Merge `beta` into `master` once the release has merged, as was done for each earlier RC
(`Merge branch 'version/...'`). This keeps the changelog and RC-only fixes, and keeps `beta`'s
history an ancestor of `master`. The merge goes one way only: `beta` into `master`, never the
reverse. Then bump `master`'s `version.txt` to the next RC
if a further RC is expected.

## Stable release (beta → release)

Once a beta has soaked, open a PR with `--base release --head beta`, titled `X.Y.Z`. Merging it
stages under `staging/`, and steps 4–6 then run the same way, with `-b release` when verifying.
Close stale `version/*` → `beta` PRs.
