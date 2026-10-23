---
name: release-state-tool
description: Cut a State Tool beta (RC) or stable release from this repo — continue the last release's version branch, cherry-pick the master changes it ships, write the changelog, get the version branch green and in sync with beta, open the PR into beta, then stage, sign, promote and verify. Use when asked to "cut a beta", "make a release", "ship an RC", "release the state tool", or "promote beta to release".
---

# Releasing the State Tool

> **Status:** steps 0–6 were worked through for 0.48.2-RC1 (PROD-240). Steps 7–10 (staging,
> Windows signing, promotion, verification, merge-back) come from the CI config and
> `activestate.yaml`, and have **not** been run end to end yet. Confirm them with the release
> owner and update this file after the next release.

## How the branches fit together

```
version/0-48-1-RC4 ──(+ version.txt bump)──▶ version/0-48-2-RC1 ──PR▶ beta ──PR▶ release
      ▲ last release                          ▲ cherry-picks of the master changes
      └── merged back into master             └── merged back into master after release
```

- **A release continues the previous release's version branch, not `master` and not `beta`.**
  After a release, the next `version/X-Y-Z-RC1` is cut from the previous RC's tip plus a
  one-line `version.txt` bump, and merged into `master`. The bump commit is "Update version.txt".
- **`master` and the RC line have diverged.** `master` carries unreleased `master`-only work
  dating back about a year, plus merge-backs of every RC branch. **Never merge `master` into a
  version branch.** That ships all of the unreleased work, and the PR shows close to 100 commits.
  Cherry-pick only the changes this release ships. Feature PRs land on `master` as single squash
  commits, so `git cherry-pick -x <sha>` works.
- `beta`'s files match the last RC's exactly, but `beta` has one extra commit: the merge commit of
  the last `version/* → beta` PR. See step 4.
- Branch names use dashes: version `0.48.2-RC1` → `version/0-48-2-RC1`. Each RC that ships bumps
  `n`; an attempt that never reached beta keeps its number.
- `version/*` branches are **protected**: a direct push is rejected ("protected branch hook
  declined"), and so is GitHub's "Update branch" ("Changes must be made through a pull request").
  Every change goes through a PR from a personal branch.
- This skill and other unrelated changes stay out of the release. They go to `master` on their
  own branch.

## 0. Read the state

```bash
git fetch origin
for b in master beta release; do echo "== $b: $(git show origin/$b:version.txt)"; git log -1 --format='%h %ad %s' --date=short origin/$b; done
git branch -r | grep 'version/' | sort -V | tail -3          # last RC branch, and the next one if it exists
git log --first-parent --format='%h %ad %s' --date=short origin/master | head -15
gh pr list -R ActiveState/cli --base beta --state open       # close stale RC PRs so nobody merges them
```

**What ships** is the first-parent commits on `master` after the last
`Merge branch 'version/...'`:

```bash
git log --first-parent --oneline <last-merge-back>..origin/master
```

Don't use `origin/beta..origin/master` for this. The RC branches reach `master` as merges with
different hashes, so that range lists dozens of commits that have already shipped.

Confirm the version (`X.Y.Z-RCn`) and the exact list of commits with the user before you push
anything.

## 1. The version branch

Check that `version/X-Y-Z-RCn` exists and is the last RC tip plus the `version.txt` bump:

```bash
git log --oneline origin/version/<last-RC>..origin/version/X-Y-Z-RCn   # expect only "Update version.txt"
git show origin/version/X-Y-Z-RCn:version.txt                          # exactly X.Y.Z-RCn
```

If it doesn't exist, ask someone with rights to create it. You can't push `version/*`.

## 2. PR the release contents into the version branch

Work in a private worktree:

```bash
git worktree add ~/work/cli-X-Y-Z -b <you>/X-Y-Z-RCn origin/version/X-Y-Z-RCn
cd ~/work/cli-X-Y-Z
git cherry-pick -x <sha> <sha> ...                  # the agreed master commits, oldest first
git diff --stat origin/version/X-Y-Z-RCn HEAD      # only those changes, plus the changelog
```

- **Changelog:** add a `## X.Y.Z` section above the previous release in `changelog.md`
  (`### Added / Changed / Fixed / Security`, Keep-a-Changelog). Write it from the commit bodies.
  Describe what users will notice, not internals, and leave out test and CI-only commits. Later RCs of
  the same version edit that section rather than adding a new one.
- **Check locally before pushing.** The build needs the generated constants first:
  ```bash
  go run scripts/constants-generator/main.go -- internal/constants/generated.go   # gitignored
  go build ./cmd/...
  go test -vet=off <touched packages>       # CI and `state run test` use -vet=off
  ```
  With vet on, the tests fail on old findings unrelated to your change. Off CI,
  `internal/condition`, `internal/constants/preprocess` and `pkg/projectfile` also fail locally.
- Open the PR with `--base version/X-Y-Z-RCn`. Squash-merging it is fine.

## 3. Get CI green

A personal head branch runs **more** of CI than a `version/*` head does. Check Format and the unit
tests run only when the head isn't `version/*`, so problems surface on the PRs into the version branch.
The Scan job (Trivy) runs after all five OS jobs pass, and **it gates Deploy**. A red Scan means the
push to `beta` would build but never stage.

Failures seen on 0.48.2, all unrelated to the release's own changes:

| Symptom | Cause | Fix |
|---|---|---|
| Check Format fails on every OS with an empty log | `check-format`'s `grep -v` exits 1 on an empty diff and `set -e` aborts | `\|\| true` (fixed on the RC line in 0.48.2; `master` still has the bug, but master PRs exit before reaching it) |
| Scan: stdlib and `golang.org/x/net` CVEs | CVEs published since the last release | bump the `go` line in `go.mod` **and** the `build.yml` matrix `go-version`, `go get <mod>@<fixed>`, `go mod tidy`, `go mod vendor`; add a changelog Security line |
| `internal/keypairs` tests fail intermittently after a Go bump | deep struct equality on `*rsa.PrivateKey` internals | compare with `rsa.PrivateKey.Equal` (fixed in 0.48.2) |
| `pkg/sysinfo` `TestPerformance/OSVersion` on `macos-latest` | the cold `sw_vers` call (100 ms–1 s) dominates a 10 ms average | larger budget on darwin (#3866). Before that, it failed 3 of 4 runs |

Check Scan locally before pushing a Go or dependency bump. Use the same Trivy version as
`build.yml` and pass its `--ignore-unfixed`:

```bash
DOCKER_CONFIG=$(mktemp -d) trivy rootfs --scanners vuln --ignore-unfixed --exit-code 1 <built binary>
```

Set `DOCKER_CONFIG` to an empty directory, or the vuln-DB download fails on a credential helper.
Without `--ignore-unfixed`, Trivy also lists GO-2026-5932 (`x/crypto/openpgp` unmaintained), which
has no fix and which CI ignores.

**CI tooling notes:**
- `gh pr checks` in this environment has no `--json`. Use
  `gh api repos/ActiveState/cli/commits/<sha>/check-runs`.
- `gh run view --log` refuses while any job in the run is still going. Use
  `gh api repos/ActiveState/cli/actions/jobs/<job-id>/logs`.
- Unit test results are in the "Check if Unit Tests Failed" step, as go test JSON. Search for
  `--- FAIL`.
- `gh pr edit` can fail silently because of a Projects (classic) error. Use
  `gh api -X PATCH repos/ActiveState/cli/pulls/<n>` and read the result back.

## 4. Sync the version branch with beta

`beta` requires the PR branch to be up to date. The version branch lacks the last release's
`version/* → beta` merge commit, so the release PR shows **BEHIND**. Open a PR from a
personal branch that merges `beta` in. It changes no files.

```bash
git switch -c <you>/X-Y-Z-RCn-sync-beta origin/version/X-Y-Z-RCn
git merge --no-ff -m "Merge branch 'beta' into version/X-Y-Z-RCn" origin/beta
```

**Merge that PR with "Create a merge commit", never squash.** A squash records no ancestry, so
the release PR stays BEHIND. Do this before the release PR's long CI run, because it restarts that run.
If the PR changes no files at all, it hits the Check Format bug above. Fix-up commits can ride
along with it.

## 5. Release PR into beta

```bash
gh pr create -R ActiveState/cli --base beta --head version/X-Y-Z-RCn --title "X.Y.Z-RCn" --body "<changelog section>"
```

It needs a review approval and green CI, including Scan. **Merging it is the release decision. Don't
merge without the user's go-ahead.**

## 6. Before merging: check S3

Check that staging is empty, so a promote moves only this release, and record what the beta channel
serves now:

```bash
aws s3 ls --recursive s3://state-tool/staging/ | head        # expect nothing
for p in linux-amd64 windows-amd64 darwin-arm64; do aws s3 cp s3://state-tool/update/state/beta/$p/info.json - | jq -c '{version}'; done
```

`version` reads `X.Y.Z-RCn-SHA<7-char commit>`. Before 0.48.2 it was `0.48.1-SHA5650ae1`.

## 7. After merge: staging (not yet verified)

The push to `beta` runs Deploy with `DEPLOY_TO_STAGING=true`. That uploads updates and installers under
`s3://state-tool/staging/update/state/…` and the remote installer under `staging/remote-installer`.
Check the `beta` run is green (`gh run list -R ActiveState/cli --branch beta --limit 3`) and that
staging now holds the new build.

## 8. Sign Windows binaries (unknown)

This is a manual step that isn't recorded in the repo. Ask the release owner (historically mitchell-as)
and write the procedure here.

## 9. Promote (not yet verified)

Run from a checkout of the merged `beta`, with AWS credentials for `state-tool`:

```bash
state run promote-staging-to-production   # interactive y/N
```

`scripts/ci/s3-promoter` **moves** staged objects. How far it reaches depends on that checkout's
`promote-staging-to-production` line in `activestate.yaml`:
- `... state-tool`, the RC line as of 0.48.x: everything under `staging/`, including the
  remote installer and any other staged channel.
- `... state-tool update/state`, `master` since 589d443b4: only `staging/update/state/`. The
  remote installer then needs promoting separately.

## 10. Verify and merge back (not yet verified)

```bash
for p in linux-amd64 windows-amd64 darwin-arm64; do aws s3 cp s3://state-tool/update/state/beta/$p/info.json - | jq -c '{version}'; done
sh <(curl -q https://platform.activestate.com/dl/cli/install.sh) -b beta && ~/.local/ActiveState/StateTool/beta/bin/state --version
```

Then merge the version branch back into `master` (`Merge branch 'version/X-Y-Z-RCn'`). The next
release's version branch is cut from this RC's tip plus a `version.txt` bump.

## Stable release (beta → release)

Once a beta has soaked, a PR with `--base release --head beta`, titled `X.Y.Z`, as in #3852. Merging it
stages under `staging/`. Steps 8–10 then follow the same way, using `-b release` to verify.
