---
name: release-state-tool
description: Cut a State Tool beta (RC) or stable release from this repo — continue the last release's version branch, cherry-pick the master changes it ships, write the changelog, get the version branch green and in sync with beta, open the PR into beta, then stage, sign, promote and verify; then test on beta and cut fix-up RCs. Use when asked to "cut a beta", "make a release", "ship an RC", "release the state tool", or "promote beta to release".
---

# Releasing the State Tool

> **Status:** steps 0–10 were worked through for the 0.48.2-RC1 beta (PROD-240, 2026-10-23 to
> 10-26), except the merge back into `master`. Testing RC1 on beta found `state publish` bugs, so
> 0.48.2-RC2 followed as a fix-up RC (step 11); it was in review on 2026-10-29. The stable
> (beta → release) path hasn't been run with this skill yet.

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

A **fix-up RC** of a version already on beta (`X.Y.Z-RC2` after RC1) is cut from `beta`'s tip
instead, since that tip *is* the squashed RC1. `version/0-48-2-RC2` is `89954c675` (the
0.48.2-RC1 squash on `beta`) plus "Update version.txt". Because it starts at `beta`'s tip, its
release PR isn't BEHIND, and step 4 is skipped unless something else lands on `beta` first.

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
| Scan: stdlib, `golang.org/x/net` or other module CVEs | CVEs published since the last release. They keep arriving during the release: a branch green one day can be red the next (0.48.2: Go 1.26.10 and x/net 0.60.0 on RC1; echo 4.16.1 on #3867; x/net 0.62.0 on #3868, all within a week) | bump the `go` line in `go.mod` **and** the `build.yml` matrix `go-version`, `go get <mod>@<fixed>`, `go mod tidy`, `go mod vendor`; add a changelog Security line. Re-check Scan on the release PR right before merging |
| macOS: `TestPerformanceIntegrationTestSuite` fails in setup with `dial unix .../activestate/cli-<branch>/state-<…>.sock: connect: invalid argument` | the socket path includes the branch name and passes macOS's 104-byte `sun_path` limit. It fails when the head branch is longer than **21 characters**; seen on PRs into `master` (#3850, #3861) | use a short head branch, e.g. `marcg/rel-skill`. Renaming means a new PR. `version/X-Y-Z-RCn` (18) fits |
| `internal/keypairs` tests fail intermittently after a Go bump | deep struct equality on `*rsa.PrivateKey` internals | compare with `rsa.PrivateKey.Equal` (fixed in 0.48.2) |
| `pkg/sysinfo` `TestPerformance/OSVersion` on `macos-latest` | the cold `sw_vers` call (100 ms–1 s) dominates a 10 ms average | larger budget on darwin (#3866). Before that, it failed 3 of 4 runs. #3866 is on the RC line only, so PRs into `master` still hit it until the merge-back |
| `internal/hail` `TestOpen` on Windows: "The process cannot access the file because it is being used by another process" | the temp dir is still locked at cleanup | flake; re-run the failed job (`gh run rerun <run-id> --failed`) |

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

#3864 (0.48.2-RC1) was squash-merged, as earlier releases were. So `beta` again has a commit the
next version branch lacks, and the next release repeats step 4.

## 6. Before merging: check S3

Check that staging is empty, so a promote moves only this release, and record what the beta channel
serves now:

```bash
aws s3 ls --recursive s3://state-tool/staging/ | head        # expect nothing
for p in linux-amd64 windows-amd64 darwin-arm64; do aws s3 cp s3://state-tool/update/state/beta/$p/info.json - | jq -c '{version}'; done
```

The served `version` is `X.Y.Z-SHA<7-char commit>`. It has **no `-RCn`**: 0.48.2-RC1 shipped as
`0.48.2-SHA89954c6`, under `beta/0.48.2/`. Before that, beta served `0.48.1-SHA5650ae1`.

## 7. After merge: staging

The push to `beta` runs Deploy with `DEPLOY_TO_STAGING=true`. Wait for the run to be green
(`gh run list -R ActiveState/cli --branch beta --limit 3`). For 0.48.2 Deploy staged 37 objects:

- `staging/update/state/beta/<platform>/info.json` and `staging/update/state/beta/<version>/<platform>/info.json`
  (two copies of each platform's `info.json`, both written by `scripts/ci/update-generator`)
- `staging/update/state/beta/<version>/<platform>/state-<platform>-<version>-SHA<sha>.{tar.gz,zip}`
- `staging/update/state/beta/<platform>/state-installer.{tar.gz,zip}`, the `install*.sh/ps1` and
  `stop.*` scripts, and `staging/update/state/versions.json`
- `staging/remote-installer/beta/{,0.1.0/}<platform>/state-remote-installer[.exe]`

```bash
aws s3 ls --recursive s3://state-tool/staging/
```

## 8. Sign Windows binaries (TI team)

The TI team holds the Authenticode certificate (ActiveState Software, Sectigo). File a Linear
issue for them; Linear has no TI team, so it goes on the Product team as a sub-issue of the
release story (0.48.2: PROD-1352, done by Pablo Bleck). Give them:

- the staged update zip's S3 path and its sha256
- the four exes inside it to sign: `state-install/bin/state.exe`, `state-svc.exe`, `state-exec.exe`
  and `state-install/state-installer.exe`

For 0.48.2 TI returned a re-signed zip of the same name in a `sign/` folder next to the original
(`staging/update/state/beta/<version>/windows-amd64/sign/`). Earlier releases (0.48.1) also signed
only the update zip: the beta zip, both channels' `state-installer.zip` and the remote installer
were not signed.

**Check the returned zip** before using it:

```bash
unzip -Z1 unsigned.zip | sort > a; unzip -Z1 signed.zip | sort > b; diff a b   # same file list
osslsigncode verify -in state.exe          # or check the PE certificate table; all 4 exes signed
# every non-.exe file byte-identical to the unsigned zip
```

**Then fix staging.** The updater checks the zip against `info.json`'s `sha256`
(`internal/updater/fetcher.go`, "sha256 did not match"), so a new zip without a new hash breaks
every Windows update:

1. Save both Windows `info.json` files and keep the unsigned zip locally, so this can be undone.
2. Copy the signed zip over `staging/update/state/beta/<version>/windows-amd64/<zip>`.
3. Set `.sha256` to the signed zip's hash in **both** `staging/update/state/beta/windows-amd64/info.json`
   and `staging/update/state/beta/<version>/windows-amd64/info.json`
   (`jq --arg s $SHA '.sha256=$s'`, upload with `--content-type application/json`). Read both back.
4. Delete the `sign/` folder once its hash matches the copy at the normal path. The promoter moves
   everything under `staging/`, so a leftover `sign/` ends up in production.

## 9. Promote

The promote changes what every beta user gets, so run it only on the user's go-ahead. Claude Code's
auto mode blocks it as a production deploy, so the user runs it (`! <command>` at the prompt).

Use a clean checkout of the merged `beta`. The promoter imports `internal/condition`, which needs the
gitignored generated constants, or it fails with `undefined: constants.ChannelName`:

```bash
git worktree add --detach ~/work/cli-beta-promote origin/beta
cd ~/work/cli-beta-promote
go run scripts/constants-generator/main.go -- internal/constants/generated.go
go build -o /dev/null ./scripts/ci/s3-promoter/    # check it compiles first
state run promote-staging-to-production           # interactive y/N, then the go run below
# equivalent, without the prompt:
go run scripts/ci/s3-promoter/main.go us-east-1 state-tool
```

It ends with `Successfully promoted <n> files from staging to production.` An AWS SSO PowerUser
role was enough for it.

`scripts/ci/s3-promoter` **moves** staged objects (copy, then delete the staging copy). How far
it reaches depends on that checkout's `promote-staging-to-production` line in `activestate.yaml`:
- `... state-tool`, the RC line as of 0.48.x: everything under `staging/`, including the
  remote installer and any other staged channel.
- `... state-tool update/state`, `master` since 589d443b4: only `staging/update/state/`. The
  remote installer then needs promoting separately.

## 10. Verify and merge back

Staging must now be empty. Check what the update service serves, which is what installs and
auto-updates read, and that the Windows binaries are signed:

```bash
aws s3 ls --recursive s3://state-tool/staging/ | wc -l          # 0
for pa in "linux amd64" "linux arm64" "darwin amd64" "darwin arm64" "windows amd64"; do set -- $pa
  for src in install update; do
    curl -sS "https://platform.activestate.com/sv/state-update/api/v1/info?channel=beta&source=$src&platform=$1&arch=$2" | jq -c '{version,sha256}'
  done; done
curl -sSfLO https://state-tool.s3.amazonaws.com/update/state/beta/<version>/windows-amd64/<zip>
sha256sum <zip>                       # = the windows info.json sha256
unzip <zip> && osslsigncode verify -in state-install/bin/state.exe   # each of the 4 exes
curl -sSfLO https://state-tool.s3.amazonaws.com/update/state/beta/<version>/linux-amd64/<tar.gz>
tar xzf <tar.gz> && state-install/bin/state --version   # Version X.Y.Z-SHA<sha>, Channel beta
```

Running the downloaded binary checks the build without running `install.sh -b beta`, which edits
the user's shell profile and their real install.

Then merge the version branch back into `master` (`Merge branch 'version/X-Y-Z-RCn'`), through a
PR from a personal branch with a short name (see the macOS row in step 3). Merge it as a merge
commit, not a squash, so the next release's "what ships" range in step 0 starts after it. If a
fix-up RC is coming (step 11), merge back only the last RC: it contains the earlier ones. The
next release's version branch is cut from this RC's tip plus a `version.txt` bump.

Tear down the promote worktree: `git worktree remove ~/work/cli-beta-promote`.

## 11. Test on beta, then fix-up RCs

Install the beta on each OS (`install.sh -b beta` / `install.ps1 -b beta`) and try what the release
changed. Bugs found here go to a fix-up RC: step 1's note on cutting `version/X-Y-Z-RC(n+1)` from
`beta`, then steps 2–10 again, including **re-signing** (step 8) because every Windows binary is
rebuilt. The served version is still `X.Y.Z-SHA<sha>` in the same `beta/X.Y.Z/` folder, with a new
SHA. Fix PRs target the new version branch; port them to `master` afterwards through the
merge-back, not separately.

**Testing encrypted private ingredients** (`state publish --build`, 0.48.2) needs an org key.
Rather than run a key service, put a key contract in `ACTIVESTATE_CLI_PRIVATE_INGREDIENT_KEY_CONTRACT`.
It works in release builds and takes precedence over the disk cache and the key service
(`internal/runbits/orgkey/https.go`):

```json
{"schema":"activestate.pim.orgkey/v1","org":"<OrgName>","key_id":"test","algorithm":"AES-256-GCM",
 "encoding":"base64","key":"<32 random bytes, base64>","fingerprint":"sha256:<hex sha256 of the raw key>"}
```

- `org` is the org **name** from the project URL in `activestate.yaml` (e.g. `ActiveStateBE`), compared
  case-insensitively. The org UUID fails with "org key does not belong to this project's organization".
- Keep the key file: everything published with it can only be installed with the same key.
- On Windows `cmd`, write the contract to a file as one ASCII line with no BOM (PowerShell 5's
  `Out-File` adds one; use `[IO.File]::WriteAllText` with `ASCIIEncoding`), then
  `set /p ACTIVESTATE_CLI_PRIVATE_INGREDIENT_KEY_CONTRACT=<contract.json`.
- `scripts/orgkeyserver.py` is the alternative: a local HTTPS key service for the
  `privateingredient.key_service_url` config.
- The publish, dependency and edit workflow is in `internal/runners/publish/readme.md`.

## Stable release (beta → release)

Once a beta has soaked, a PR with `--base release --head beta`, titled `X.Y.Z`, as in #3852. Merging it
stages under `staging/`. Steps 8–10 then follow the same way, using `-b release` to verify.
