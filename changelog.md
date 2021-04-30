# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.29.0

### Changed

- The State Tool now comes with the ActiveState Desktop application, which is
  currently a system tray application.
- We've replaced the auto updating system with a manual updating system, updates
  will from now on be made available through the system tray application and you
  will have the choice to install it now or later.

## 0.28.0

### Changed

- New runtimes are installed in parallel and 2-4 times faster.
  ([PR #1275](https://github.com/ActiveState/cli/pull/1275))

### Fixed

- `state push` updates project name in activestate.yaml.
  ([PR1297](https://github.com/ActiveState/cli/pull/1297))

## 0.27.0

### Added

- New system tray executable for the Windows platform
  ([PR #1285](https://github.com/ActiveState/cli/pull/1285))

### Changed

- Enhanced error reporting for errors that happened early on in the application
  logic ([PR #1280](https://github.com/ActiveState/cli/pull/1280))
- Updated name of `state cve` command to `state security`. Aliased `state cve`
  to `state security` ([PR #1286](https://github.com/ActiveState/cli/pull/1286))

### Fixed

- Fixed issue where `state push` would fail on existing projects.
  ([PR #1287](https://github.com/ActiveState/cli/pull/1287))

## 0.26.0

### Added

- New command `state cve open <cve-id>` opens the National Vulnerability
  Database entry for the given
  CVE ([PR #1269](https://github.com/ActiveState/cli/pull/1269))

### Fixed

- Fixed issue where `state deploy` would fail without the `--path` flag
  ([PR #1270](https://github.com/ActiveState/cli/pull/1270))

## 0.25.1

### Fixed

- Fixed issue where `state pull` would not pull in the latest
  changes ([PR #1272](https://github.com/ActiveState/cli/pull/1272))

## 0.25.0

**Warning:** This update will force a change to your activestate.yaml which is
incompatible with earlier state tool versions. As long as everyone on your
project updates their state tool there should be no interruption to your
workflow.

### Added

- New command `state cve` allows for reviewing security vulnerabilities on your
  project ([PR #1209](https://github.com/ActiveState/cli/pull/1209))
- You can now specify a package version when calling `state info`,
  eg. `state info <name>@<version>` ([PR #1201](https://github.com/ActiveState/cli/pull/1201))
- You can now specify a new project name by
  running `state pull --set-project OWNER/NAME` (primarily for converting
  headless projects) ([PR #1198](https://github.com/ActiveState/cli/pull/1198))
- You can now switch between update channels
  via `state update --set-channel` ([PR #1190](https://github.com/ActiveState/cli/pull/1190))
- State tool will now provide instructions on how to get out of a detached
  state ([PR #1249](https://github.com/ActiveState/cli/pull/1249))
- State tool now supports branches via flags in `state activate` and
  the `state branch` subcommand. See `state branch --help` for more information.

### Changed

- Activating a new project non-interactively no longer makes that project "
  default" (you can pass the `--default` flag for this
  use-case) ([PR #1210](https://github.com/ActiveState/cli/pull/1210))
- The user experience of `state secrets` is now consistent with the rest of the
  State Tool ([PR #1197](https://github.com/ActiveState/cli/pull/1197))
- `state import` now updates your runtime, so you don't need to re-activate
  after importing
  anymore ([PR #1241](https://github.com/ActiveState/cli/pull/1241))

### Fixed

- Progressbar sometimes hangs while waiting for build to
  complete ([PR #1218](https://github.com/ActiveState/cli/pull/1218))
- Fixed issue where some unicode characters were not printed
  properly ([PR #1207](https://github.com/ActiveState/cli/pull/1207))
- Prompts for default project should now only happen once per
  project ([PR #1210](https://github.com/ActiveState/cli/pull/1210))
- Fixed issue where `state activate` sometimes used the wrong
  activestate.yaml ([PR #1194](https://github.com/ActiveState/cli/pull/1194))
- Fixed issue where `state info owner/name` would fail if not currently in a
  project directory ([PR #1255](https://github.com/ActiveState/cli/pull/1255))
- Fixed issue where running tooling from the global default project with
  the `-v` flag would spew out state tool debug
  info ([PR #1239](https://github.com/ActiveState/cli/pull/1239))
- Fixed issue where sometimes perl/python is still pointing at the system
  install after
  activation ([PR #1238](https://github.com/ActiveState/cli/pull/1238))
- Fix issue where state tool sometimes throws "panic" errors when updating the
  configuration ([PR #1232](https://github.com/ActiveState/cli/pull/1232))
- Fix issue where `state activate` sometimes throws a "
  panic" ([PR #1229](https://github.com/ActiveState/cli/pull/1229))

### Deprecated

- The `--replace` flag for `state activate` is now deprecated in favour of `state pull --set-project`
