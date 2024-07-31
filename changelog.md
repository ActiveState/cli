# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 0.45.1

### Fixed

* Fixed issue where installation on Windows would fail with a message from powershell saying script running is disabled.
    * Context: We use a powershell script to create start menu shortcuts for the State Tool, as there are no solutions
      in Golang to do this through system APIs.

## 0.45.0

### Added

* On Linux we will now automatically detect the most appropriate platform based on the system glibc version.
    * This only applies if your project has multiple linux platforms defined.
    * If you were using the `runtime.preferred.glibc` config option it will still be respected, but you likely won't
      need it anymore.
* We now show failed builds when running `state artifacts`. You can still instrument the artifacts that did not fail.

### Changed

* We no longer support running shell-builtins through `state exec`, instead we require that the executable passed in
  exists on the system. Running shell-builtins was never the intended behavior, but was a side-effect. You can still
  access shell built-ins by running it through a shell, eg. `state exec -- cmd /C "where python3"`.
* Our installers now give slightly better indication that a download is happening, avoiding confusion about the process
  hanging.
* We will now produce an error if you have an invalid `if` conditional in your activestate.yaml. Previously these
  conditionals were simply assumed to be `false`.
* We will now inform you there is nothing new to commit when running `state commit` with no changes to commit.
* The `LOCAL` and `REMOTE` targets for `state reset` are now case-insensitive.
* You can now `state checkout` a project without a language defined in its configuration.
    * Note making changes to such a project in the State Tool is not yet fully supported.
* When the State Tool encounters an unexpected internal we now relay this internal error to the user. Previously you
  only received a generic "execute failed" error, which is far less helpful than an internal error.
* Running `state pull` will now fail if the configured commit does not belong to the configured project.
    * This is a corrupted state that the user can encounter by manually editing their activestate.yaml (eg. by resolving
      a git conflict).

### Fixed

* Updates would retry and eventually timeout if it took longer than 30 seconds to download.
* On Windows, arguments were not always escaped properly when using commands like `state exec`, or runtime executors.
* Sometimes row/column text was not aligned properly.
* Build progress and an empty build log would sometimes be produced when the build was already done.
* The "Platforms" command group was shown twice on `state --help`, one of these should have been the "Authors" group.
* When running `state artifacts` with the `--commit` flag; it was not respected when `--namespace` was also supplied.
* Running `state publish` with a custom `EDITOR` environment variable would not respect the configured editor.
* Sometimes `state pull` would produce conflicts when there shouldn't have been any.
* Our service process now has a longer timeout window, eliminating state-svc errors when the system is under heavy load
  and the service hasn't started yet (eg. when booting up).
* We no longer show redundant bullets for each change under `state history`.
* Running `state publish --edit` will no longer require you to supply a file (eg. you just want to change the
  description).

### Security

* Addressed several CVEs, packages in question [have been updated](https://github.com/ActiveState/cli/pull/3306/files)
  to more recent appropriate versions.

## 0.44.1

### Changed

* Authentication information is now cached between commands, increasing performance of State Tool commands.

## 0.44.0

### Added

* We've added the `state manifest` command, which lists all your package, bundle and language requirements in your
  project. This command deprecates the old `state packages`, `state languages` and `state bundles` commands.
* `state artifacts` now indicates which artifacts are still building.
* You can now use wildcard versions when running `state install` and `state languages install`, just like you can
  with `state init`. eg. `state install datetime@1.x`.
* We now warn you when installing the State Tool as an administrator on Windows.
* `state publish` now lets you specify the type of a dependency (ie. runtime, buildtime, or testing) via the new
  `--depend-build`, `--depend-runtime` and `--depend-test` flags. The existing `--depend` flag maps to `--depend-build`.
* We now ensure the runtime is not currently in use before making changes to it. If the runtime is the user will receive
  an error asking them to stop using the runtime before proceeding.
* You can now supply multiple packages to the `state install` and `state uninstall` commands.

### Changed

* Increased the time State Tool will wait for the state-svc to start, which is required for using the State Tool. This
  is mainly intended for when you just started your system, when it is still under heavy load and thus processes can
  take longer to start.
* You will now receive an actionable error when your activestate.yaml does not contain a commit ID.
* State revert "HEAD" has been renamed to "REMOTE", though "HEAD" is still supported with a deprecation warning. This
  is to avoid confusion with how git uses the HEAD term, which isn't how we were using it.
* `state run` and `state exec` now use powershell under the hood instead of cmd. This means that invoking shell
  built-ins like `where` will no longer work as powershell has different built-ins. But if you're invoking shell
  built-ins you probably shouldn't be using these commands anyway.
* State tool now identifies more network errors, making for more actionable error messages.
* We have introduced a new project file versioning mechanic. You'll start seeing a new `config_version` property in your
  activestate.yaml. Do not edit or delete this or you may run into issues.
* Buildscripts are no longer automatically created when opted in. Instead you need to run `state reset LOCAL` in order
  to first create the buildscript. You only need to do this once, after it exists it will be updated as needed.
    * This is merely a growing pain while this feature is optin only. Once buildscripts ship as stable you will not need
      to do this.
* CVE information provided when running `state install` is now recursive, meaning we show CVE information for the
  requested package as well as all its dependencies.
* `state init` now automatically assumes wildcards when a partial version is specified.

### Fixed

* Fixed issue where `state refresh` would tell you you have uncommitted changes when you are up to date (when opted into
  buildscripts).
* Fixed issue where `state uninstall` would give you superfluous output intended for `state install`.
* Fixed issue where running `state update` would show deprecation messages for the version you are updating from.
* Fixed issue where `state run` and `state exec` would not forward arguments correctly on Windows. This is a limitation
  of the `cmd` shell. We now utilize powershell instead to facilitate commands.
* Fixed issue where `state languages` would show a larger than / smaller than version notation rather than `auto`.
* Fixed issue where `state checkout` would save the `main` branch to your activestate.yaml when you checked out a commit
  on a sub-branch. It now correctly saves the sub-branch.
* Fixed issue where `state search` gave an unintended "wrapped tips" error when given an invalid version number.
* Fixed issue where text was wrapped when producing non-interactive output, making it impossible to use the output in
  automations without first cleaning up the output.
* Fixed issue where manually resolving conflicts when running `state pull` would not allow you to commit these changes
  if you picked only the local changes. This only applies if opted in to buildscripts.
* Fixed issue where conflicts on the `at_time` buildscript field would show the wrong local value.

### Removed

* `state export recipe` has been removed, as the platform no longer uses recipes.

## 0.43.0

### Added

* Added `state publish` command, which lets you publish your own ingredients to the ActiveState platform.
* Added user configuration for setting the preferred glibc for projects which produce multiple glibc variants, eg.
  `state config set runtime.preferred.glibc 2.17`.
* You can now disable the behavior of State Tool updating your shell prompt to include the project name. Use
  `state config shell.preserve.prompt false`.
* Added the `state languages search` command, allowing you to discover what languages you may want to use.
* Added a new `state export log` command, which lets you easily inspect the debug log.
* State tool executables on Windows now provide sufficient meta information.
* Added preview version of buildscripts, the ActiveState platform equivalent of a pyproject.toml file. Currently they
  only work with the new `state commit` and `state eval` commands.
* Added the `state artifacts` command, which lists all available artifacts produced for your project (eg. installers,
  docker
  images, python wheels, ..)
* Added the `state artifacts dl` command, which lets you download individual builds for your project.

### Changed

* `state refresh` has been marked stable.
* `state checkout` will now revert any changes made to the filesystem if the runtime fails to source.
    * You can now specify the `--force` flag in order for it to always checkout the project even if it cannot be
      installed.
      Allowing you to work on the project via the CLI and fix the underlying issue.
* `state import` no longer overwrites your project with the imported requirements. Instead the requirements are
  appended.
* `state platforms search` will no longer show platforms that are unsupported.
* `state packages --filter` is now case-insensitive.
* You can now revert to the head of a branch by specifying `state revert HEAD`, rather than having to figure out the
  commit ID yourself.
* Constants defined in the activestate.yaml are no longer exported as environment variables unless you specify
  the `export: true` property.
* We now clearly communicate that you are using a project runtime whenever you open a new shell.
* If installation fails due to a networking issue we will now clearly communicate the cause.
* We have introduced a new mechanic for handling errors, which should produce more actionable error messages. This
  mechanic is being rolled out to various commands over time, as of right now we have targeted common commands
  like `init`, `checkout`, and `install`.
* The installer will no longer refuse to install if you already have the State Tool installed, but the installed State
  Tool is not properly configured.
* The installer will no longer leave behind temporary files in your system temp dir.
* Structured output (ie. `--output json`) now includes tips when errors occur.
* All our executables now support the `--version` flag.
* Reduced the verbosity of the debug log.
* `state info` has been updated to give more meta information, including CVE details across versions of the requested
  package.
* `state info` now respects the case sensitivity rules of your projects language.
* `state install` and `state checkout` will now show a dependency tree for which dependencies were brought in along with
  the requested package.
* `state install` will now warn you above CVE's in the package being installed. If there are CVE's of level critical it
  will prompt you to confirm before installing. This mechanic is configurable.
* `state history` now shows catalog revision information.
* `state update lock` will now disable auto updates, in addition to locking your project down to the current State Tool
  version.
* We now show both the requested and resulting version of packages across different commands,
  eg. `state packages`, `state languages`, ..
* Auto update will no longer run for certain critical commands or when using structured output (eg. `--output json`).
* Raised the artifact cache size from 500MB to 1GB.

### Fixed

* Fixed issue where specifying multiple version constraints (eg. `>1.0,<2.0`) only preserved the last constraint (
  eg. `<2.0`).
* Fixed `state init` being unable to initialize a ruby project without an explicit version.
* Fixed using `state shell` through `tcsh` not setting up environment correctly.
* Fixed `state activate` checking out the wrong commit if the commit you specified does not exist. It will now error
  out.
* Fixed issue where typing during a selection prompt would make the prompt disappear.
* Fixed various localisation issues.
* Fixed issue where `state shell` would improperly set up your PATH if there are entries with spaces.
* Fixed update available message being printed twice.
* Fixed issue where local cache was not always used to install artifacts.

### Removed

* Removed the `--set-project` flag from `state pull` as it covered a use-case that no longer exists.
* Removed `state security report`, you can now access everything through `state security` instead.

## 0.42.1

### Fixed

* Fixed `state import` no longer working due to important API changes.

## 0.42.0

### Added

* The State Tool now has better user-facing errors. These should provide more context
  and actionable information when an error occurs. Currently, this is only implemented
  on a subset of commands, but will be expanded to all commands in the future.
* The State Tool now fully supports Ruby as a language.
* Users can now install their runtime with all of the build dependencies by using
  the `ACTIVESTATE_INSTALL_BUILD_DEPENDENCIES` environment variable.
* Users can now install a specific version of the State Tool by passing the version
  to the installer script without the SHA suffix. Previously, users would have to
  know the SHA of the version they wanted to install.

### Changed

* `state init` now uses the new buildplanner API to create a project on the
  platform.
* Runtime counts and limits are no longer surfaced to the user.
* `state revert` now uses the new buildplanner API to revert a previous commit
  as well as revert to a specific commit using `state revert --to`.
* `state pull` now uses the buildplanner API to merge changes from the platform
  into the local project.
* We no longer depend on the `file` utility when installing the State Tool. This
  should make the installation process more reliable.
* Updated detection for requirement changes when installing packages. This brings
  the installation process in line with new APIs and should make the process more
  reliable.
* Update checks have been moved to the `state-svc`. This should speed up State
  Tool execution and improve the reliability of the update process.

### Removed

* Support for headless commits has been removed. Users can no longer get into a
  state where they have a headless commit. Existing projects that are headless
  will now be prompted to convert their project before it can be used.
* The State Tool no longer supports signup via the prompt. Users may still sign
  up with `state signup` however this will open the signup page in their
  browser.

### Fixed

* Fixed an issue where the `state-svc` log file path was not being presented
  correctly in the `state-svc status` output.
* Fixed an issue where the version number could be empty when running
  `state update` and no update is available.
* Fixed an issue where executors would halt on most CI systems.
* The install scripts now verify the checksum of the downloaded archive before
  extracting it even when a full version is provided as a flag.
* Fixed unneccessary repetition of certain error tips.
* The commit date has been fixed when running `state history`.
* Fixed an issue where the commit message would be missing on some commits when
  running `state history`.

## 0.41.0

### Added

* `state init` is now a stable command, meaning you no longer need to opt-in to
  unstable commands to use it.
* Signing up for a new account now opens the account creation page in your
  browser instead of the login page.
* `state shell` can now detect currently active subshells preventing nested
  shells from being created.
* Wildcard and partial version matching is now supported for `state install`
  and for language versions with `state init`. For example:
  `state install pytest@2.x`.
* Added messaging on the potentially disruptive nature of editing or moving a
  project.
* Users can now check out a project without cloning the associated git
  repository. For example: `state checkout <orgname/project> --no-clone`.

### Changed

* Errors encountered while sourcing a runtime now have more informative error
  messages.
* The default Ruby version is now 3.2.2.
* Improved parsing to reduce runtime installation errors.
* Updated help details of `state use` to be more informative.
* The State Tool can now be installed by extracting its archive file to a
  directory of your choice.
* Some runtime installations will now be faster due to improved artifact
  handling.

### Fixed

* Improved error messages when unauthenticated to better indicate that
  authentication may resolve the error.
* Fixed issue where build time dependencies ended up being downloaded/installed.
* Fixed issue where API requests would not be retried, sometimes resulting in
  a "decoding response: EOF" error.
* Fixed issue where we would sometimes print the wrong project name when
  interacting via the current working directory.
* Fixed `state history` showing JSON formatted data in changes section.
* Fixed "Error setting up runtime" message that would happen in some rare cases.
* Fixed "Invalid value for 'project' field" error that could sometimes happen
  when running checkout/init/fork.
* Fixed issue where `state init` created the project on the platform even though
  an error happened, leaving the project in an uncertain state.
* Fixed issue where `state checkout` and `state init` would not respect the
  casing of the owner and/or project on the platform.

## 0.40.1

### Added

* State tool will now warn users if its executables are deleted during
  installation, indicating a false-positive action from antivirus software.

### Fixed

* Fixed auto updates not being run (if you are on an older version:
  run `state update`).
* Fixed a rare parsing panic that would happen when running particularly complex
  builds.
* Fixed race condition during artifact installation that could lead to errors
  like "Could not unpack artifact .. file already exists".

## 0.40.0

### Added

- New command `state projects edit` which allows you to edit a projects name,
  visibility, and linked git repository. You will need to opt-in to unstable
  commands to use it.
- New command `state projects delete` which allows you to delete a project.
  You will need to opt-in to unstable commands to use it.
- New command `state projects move` which allows you to move a project to a
  different organization. You will need to opt-in to unstable commands to use
  it.

### Changed

- Runtime installations have been updated to use our new buildplanner API. This
  will enable us to develop new features in future versions. There should be no
  impact to the user experience in this version.
- Runtime installation is now atomic, meaning that an interruption to the
  installation progress will not leave you in a corrupt state.
- Requirement names are now normalized, avoiding requirement name collisions as
  well as making it easier to install packages by their non-standard naming.
- Commands which do not produce JSON output will now error out and say they do
  not support JSON, rather than produce empty output.
- When `state clean uninstall` cannot uninstall the State Tool because of third
  party files in its installation dir it will now report what those files are.

### Removed

- The output format `--output=editor.v0` has been removed. Instead
  use `--output=editor` or `--output=json`.

### Fixed

- Fixed issue where the `--namespace` flag on `state packages` was not
  respected.
- Fixed issue where `PYTHONTPATH` would not be set to empty when sourcing a
  runtime, making it so that a system runtime can contaminate the sourced
  runtime.
- Several localization improvements.

## 0.39.0

### Added

- Added new messaging functionality that allows us to send messages for things
  like new releases, deprecations, etc.

### Changed

- Running `state init` and `state refresh` successfully will now give the same
  environment information as commands like `state checkout` and `state use`.
- The `state init` command now takes a `path` argument, replacing the previous
  `language` argument which is now a flag. This makes the command more
  consistent with other state tool commands.
- Running `state init` will now source the initialized runtime.
- We've revisited JSON output to now be much more consistent. Previously certain
  commands would oblige by the request to give JSON output, but would give JSON
  output that isn't actually curated for machine consumption. As a result you
  may now get an error saying a given command does not support JSON, but ones
  that do now generally give far more useful JSON output.
    - Commands that support JSON
      output: `auth`, `branch`, `bundles install`, `bundles search`,
      `bundles uninstall`, `checkout`, `config get`, `config set`, `cve`,
      `cve report`, `events`, `export config`, `export env`, `export jwt`,
      `export new-api-key`, `export private-key`, `export recipe`, `fork`,
      `history`, `info`, `init`, `install`, `languages`, `organizations`,
      `packages`, `platforms`, `platforms search`, `projects`,
      `projects remote`, `pull`, `reset`, `revert`, `scripts`, `search`,
      `secrets`, `secrets get`, `show`, `switch`, `uninstall`, `update lock`,
      `use`, `use show`.
    - Note that the format of the JSON output itself should be considered
      *unstable* at this time (ie. subject to change).
- As a result of the revised JSON output we will no longer print NIL characters
  as delimiter between JSON objects. So you no longer need to account for these.
- Requirement names (eg. when running `state install <pkg>` or
  `state bundle install <bundle>` are now normalized to the casing of the
  matching ingredient rather than that of the user input.
- On macOS, the `State Service.app` now installs to the installation directory
  of the State Tool rather than the users Applications directory, as this is a
  daemon application and not a user facing application.
- Log rotation has moved to the State Service, making the State Tool itself
  perform faster.

### Removed

- Google cloud secret integration has been removed. In order to use secrets you
  will need to configure them yourself as you would with any other CI.

### Fixed

- Fixed bug where events and executors would not set up the PATH correctly when
  used from cmd.exe on Windows.
- Fixed bug where `state auth` would panic if there was no internet connection.
- Fixed installation placing a pointless copy of the installer inside the
  install dir.
- Fixed ZSH not being configured if no .zshrc file existed yet.
- Fixed installer `--force` flag not being respected when the target dir is
  non-empty.
- Fixed issues where incomplete error information was sometimes reported.
- Fixed various errors and success messages to more clearly indicate what
  happened.
- Fixed issue where State Tool would retry network requests that had no change
  of succeeding, resulting in longer wait times for the user.

## 0.38.1

### Fixed

- We've reverted a change that would cause runtime installation to fail when we
  received out of order progress events. This was causing some runtimes to
  report failure when in actuality they were installed successfully.
- Fixed reinstalling/updating on macOS resulting in a "Installation of service
  app failed" error.

## 0.38.0

### Added

- There is a new `state refresh` command which simply refreshes your cached
  runtime files. This is particularly useful when using git, eg. when
  you `git pull` in changes to your activestate.yaml you can now simply
  run `state refresh` to have State Tool source the related runtime changes.
- The activestate.yaml now features a convenient shorthand syntax for defining
  scripts, constants, etc. This does not replace the old syntax, the old syntax
  is still appropriate when you want to define more than a simple "key" and "
  value" field.

  **Example:**

  ```yaml
  scripts:
    # Full syntax notation:
    - name: build
      language: bash
      value: go build .
    # Short syntax notation:
    - build: go build
  ```
- The `state revert` command has a new `--to` flag, which will make it create a
  commit that effectively reverts you back to the state of the provided commit.
- Progress indication when installing a runtime now supports non-interactive
  mode. When run from non-interactive mode it will simply print dots to indicate
  that progress is still happening.

### Changed

- We have revisited the behavior of `state init` to be less error prone and more
  intuitive. Our goal is to stabilize this command by version 0.39.0.
  These changes include:
    - Immediately creating the project on the platform, rather than waiting for
      the user to run `state push`.
    - Assume Python 3 rather than Python 2 when initializing a Python project
      without specifying a version.
    - Assume the most recently used language when no language is specified.
    - Drop the `--skeleton` flag.
- Changed the sorting and grouping of `--help` output to be more intuitive.
- Made the `--help` output wrap on words rather than characters.
- Using secrets without having set up a keypair now gives a more informative
  error message.
- Running `state clean uninstall` will now only uninstall the application files.
  In order to also uninstall the cache and config files you need to specify
  the `--all` flag, eg. `state clean uninstall --all`. This brings the behavior
  of the uninstaller in line with other uninstallers.
- The `--help` output will now always show a warning about unstable commands if
  you are opted in to using them.
- Specifying the `--exact-term` flag when searching
  with `state search --exact-term` will now also make the search term
  case-sensitive. This is to bring the behavior in line with that
  of `state info`.
- The state service daemon now autostarts as an .app on macOS, rather than a
  shell file. Making for a friendlier user experience as it is now easier for
  users to understand what this newly added login item is.

### Fixed

- Fixed issue where user would be interrupted when auto update fails.
- Fixed issue where the installer would never exit under CI environments as it
  did not detect them as non-interactive.
- Fixed confusing error message when trying to check out a project in a location
  that already has a project.
- Fixed the uninstall command window closing without showing what happened to
  the user when running it from the start menu shortcut on Windows.
- Fixed new checkouts of Python projects on Windows showing a
  "UnicodeEncodeError" error message when activating them.
- Fixed `state pull --set-project` updating the activestate.yaml even though the
  command failed due to an incompatible project being provided.
- Fixed `state exec <bogus-command>` resulting in a State Tool error rather than
  just the expected shell error.
- Fixed autostart behavior on Linux sometimes resulting in the user having two
  separate autostart entries due to running the installer and the update in
  different modes (interactive vs non-interactive).

### Removed

- Removed the `--force` flag from `state update lock` and `state update unlock`,
  as it is redundant with the `--non-interactive` flag.

## 0.37.1

### Fixed

- Fixed some runtimes not being installable due to a "Failed to download
  artifact" error.
- Fixed `state update lock` throwing a panic when run outside of the context of
  a project.

## 0.37.0

### Changed

- The following commands have been marked as stable, you no longer need to
  opt-in to unstable to use them:
    - `state checkout`
    - `state info`
    - `state scripts`
    - `state shell`
    - `state switch`
    - `state use reset`
    - `state use show`
    - `state use`
- All titles/headings are now consistently formatted.
- Better use of whitespace in the error output.
- `state clean uninstall` now only removes the application files. Use `--all` to
  also delete config and cache files.
- Runtime progress will now fail rather than silently continue if we received
  out of progress events, preventing vague failures later on.
- Dropped the `--force` flag from `state import`. The same use-case is addressed
  with `--non-interactive`.
- Using `state shell` with an invalid SHELL environment variable will now give
  a more informative error message.
- `state init` now uses more recent default language versions.

### Added

- `state checkout` has a new flag named `--runtime-path`, which allows you to
  specify where the runtime files should be stored.

### Fixed

- Fixed commit messages containing empty information.
- Fixed installation failing because "State Tool is already installed" even
  though it was uninstalled.
- Fixed `state revert` failing when not authenticated, even when no
  authentication is required.
- Fixed `state revert` failing with a vague error if provided an invalid commit
  ID.
- Fixed `state clean uninstall` giving a success message even when there were
  failures.
- Fixed `state import` giving a vague error message when the file specified does
  not exist.
- Fixed issue where a panic in the code would not be handled gracefully.

## 0.36.0

### Added

- All commands have been updated to proactively mention project and runtime
  information, making it easier to understand what is going on and how to
  configure
  your tooling.
- State Tool will now give you a heads-up if the organization you're accessing
  has gone over its runtime limit.
- State Tool will now configure itself for all supported shells on your system,
  rather than just the currently active shell.
- Better support for Bash on Windows.

### Changed

- Significantly improved the performance of runtime executors.
- `state revert` now reverts "a" commit, rather than reverting "to" a commit.
  This
  is meant to bring the user-experience in line with that of git.
- Bash on macOS is no longer supported as a shell. This is due to the fact that
  macOS has deprecated the use of bash in favor of zsh. Using bash should still
  work, but you will receive warnings, and it may stop working in the future.
- The state-svc is now installed as an App on macOS. Solving the issue of macOS
  referring to it as an sh script which isn't very useful for end-users.
- Progress indication for runtime installations will now show build progress for
  all artifacts, even if they are cached.
- Reorganized the `--help` output.

### Fixed

- Fixed error message received when running State Tool without the `HOME` env
  var
  not being indicative of that root cause.
- Fixed progress count being off when installing runtimes.
- Fixed progress sometimes hangs or panics while installing runtimes.
- Fixed `state languages install` and `state platforms add` should not modify
  the
  remote project (that's what `state push` is for).
- Fixed `state import` panics when ran outside of a project folder.
- Fixed malformed error message when `state clean uninstall` fails.
- Fixed `state push` creating the remote project even if the user told it not
  to.
- Fixed unstable subcommands not showing a warning explaining that they are
  unstable.
- Fixed `state shell` giving a misleading error when no default project is
  configured.
- Fixed `state update` showing redundant output.
- Fixed `state import --non-interactive` cancelling out of import rather than
  continuing without prompting.
- Fixed `state revert <commit ID>` should not work on a commit that doesn't
  exist in
  the history.
- Fixed `state clean cache` not giving a success or abort messaging.
- Fixed `state export private-key` giving an uninformative error message when
  improperly authenticated.
- Fixed `state show` not working with commits that haven't been pushed to the
  platform.
- Fixed `state checkout` failing if target dir is non-empty but does not contain
  an activestate.yaml.

### Removed

- Removed the `--set-version` flag from `state update`. Instead, you can run the
  installation script with the `-v` flag.
- The experimental tray tool (ActiveState Desktop) has been removed. It will be
  making a reappearance in the future.
- The `--namespace` flag has been removed from `state history`. To inspect
  projects
  without checking them out you can use the website.

## 0.35.0

We are introducing a set of new environment management commands that will
eventually replace `state activate`. The intend behind this is to make the
use-cases currently covered by the activate command more explicit, so that users
have more control over their workflow.

In short; we're introducing the following commands:

- *checkout* - Checkout the given project and setup its runtime
    - A checkout is required before you can use any of the following commands
- *use* - Use the given project runtime as the default for your system
    - *reset* - Reset your default project runtime (this also resets the project
      configured via `state activate --default`)
    - *show* - Show your default project runtime
- *shell* - Starts a shell/prompt for the given project runtime (equivalent of
  virtualenv)
- *switch* - Switch to a branch or commit

All of the above commands are currently marked as unstable, meaning you cannot
use them unless you opt-in to unstable commands with
`state config set optin.unstable true`.
This is to give us time to test and improve the commands without necessarily
ensuring backward compatibility. These commands have been thoroughly tested, but
since they are new bugs are still more likely than with stable commands.

Note that `state activate` will still be available for the foreseeable future.

### Added

- Added new environment management commands (see above for details)
    - Added `state checkout` command.
    - Added `state use` command.
    - Added `state use reset` command.
    - Added `state use show` command.
    - Added `state shell` command.
    - Added `state switch` command.
- Added `state export env` command - Export the environment variables associated
  with your runtime.
- Added `state deploy uninstall` command for reverting a `state deploy`.
- Added `state update unlock` command, which undoes what `state update lock`
  does.
- Runtime artifacts are now cached, speeding up runtime setup and reducing
  network traffic.
    - The cache is capped at 500mb. This can be overridden with
      the `ACTIVESTATE_ARTIFACT_CACHE_SIZE_MB` environment variable (value is
      MB's as an int).

### Changed

- State tool will now error out when passed superfluous arguments (
  eg. `state activate name/space superfluos-arg`).
- The installer will no longer show debug error messages.
- We now start the background service automatically when you boot your machine.
- State tool now configures all compatible shells that were found on the users
  system.
- We now report how far ahead / behind you are from your branch when
  running `state show`.

### Fixed

- Fixed State Tool being unusable on M1 Macs running Ventura.
- Fixed `~/.cshrc` not being respected when using `tcsh`.
- Fixed `-v` flag not working when using `install.sh` to install State Tool.
- Fixed state tool background service closing prematurely.
- Fixed bash scripts on Windows using the wrong path format.
- Fixed a variety of missing/wrong localisation issues.
- Fixed `state invite` resulting with response code error message.
- Fixed various issues where running with `--non-interactive` would not have
  the desired behavior.
- Fixed `state config set` accepting invalid values for booleans.
- Fixed `state exec` not respecting the `--path` flag.
- Fixed issue where PYTHONPATH would be set up with a temp directory on macOS.
    - This still worked as expected in the end, but is obviously awkward.
- Fixed panic when running `state secrets get` without a project.
- Fixed issue where `state learn` would give an unhelpful error when it could
  not reach the browser.
- Fixed `state show` not working for private projects.
- Fixed variables as arguments to executors (eg. python3.exe) not being expanded
  properly.
- Fixed state tool interpreting `-v` flag when its passed through `state run` or
  `state exec` but not intended for the state tool.
- Fixed State Tool being added to PATH multiple times.
- Fixed unstable commands reporting `--help` info when passed invalid
  arguments, instead of saying the command is unstable and you should opt in.
- Fixed `state uninstall` with a non-existent package reporting the wrong error.

## 0.34.1

### Changed

* The `state use` command has been marked unstable.

### Fixed

* Fixed issue where activating a second project with an identical name to the
  first would instead activate the first project.
* Fixed issue where error output was sometimes missing important details about
  what went wrong.
* Fixed issue where build errors were incorrectly reported.
* Fixed issue where service could not run due to filepath size limits on macOS.
* Fixed issue where passing a relative path to `state activate --path` would
  sometimes not resolve to the correct path.
* Fixed issues where installer would sometimes give the update user experience.

## 0.34.0

### Added

* We've started flagging commands as stable and unstable, and by default will
  only support execution of stable commands. To run unstable commands you must
  first opt-in to them using `state config set optin.unstable true`.
* We've added a new `state use <orgname/project>` command, which will allow you
  configure the given project as the default runtime on your system.
* Automatic updates can now be disabled with `state config set autoupdate false`
  .
* On Windows we now add an Uninstall shortcut to the start menu.
* Analytics can now also be disabled with an environment variable:
  `ACTIVESTATE_CLI_DISABLE_ANALYTICS=true`.

### Changed

* The state-svc (our background daemon) has seen significant improvements to its
  start / stop behavior. Primarily intended to improve the reliability of our
  update process.
    * As a result our minimum Windows version required to run the state tool is
      now *Windows 10 Build 17134 (Codename Redstone 4)*.
* The State tool will now error out when it can't communicate with the
  state-svc.
  Preventing the user from running into much more vague errors as a result of
  the
  missing daemon.
* `state config` can now only act on valid config keys.
* A number of error messages have been improved to give a better idea of how the
  user can remedy the error.
* Our installer has been optimized to use a smaller file size and reduce the
  number of processes as part of the installation.

### Fixed

* Fixed issue where variables in command line arguments were not properly
  interpolated. Causing the command to receive an empty value rather than
  the variable name.
* Fixed issue where `state clean uninstall` would fail to clean up the
  environment.
* Fixed issue where `state activate --branch` would sometimes error out.
* Various issues leading to corrupt, miss-placed, or error-prone installation
  directories.
* Fixed issue where the State Tool installation directory was added to PATH
  multiple times.
* Fixed issue where calling `state clean cache` with `--non-interactive`
  did not clean the cache.
* Fixed issue where `state history` would fail if history had an author that is
  no longer a member of the organization.
* Fixed issue where automated tools and integrations (including Komodo IDE)
  could not get the list of organizations for the authenticated user due to a
  backwards incompatible change.
* Fixed cases of missing localization.

### Removed

* The `--replace` flag has been dropped from `state activate`, its use-case has
  been addressed by `state pull --set-project`.

## 0.33.0

### Added

* Authentication now uses your browser for a more secure and transparent
  authentication process.

    * The old behavior is still available as well, and use-cases where you
      provide
      the api key or credentials in the command are unaffected.

* Added a new `state config` command, which can be used to change behavior of
  the State Tool itself.

    * Currently can be used to disable analytics and error reporting, eg.

  ```bash
  state config set report.analytics false # Turns off analytics
  state config set report.errors false # Turns off error reporting
  ```

### Fixed

* Fixed issue where temporary files were not cleaned up in a timely manner.
* Fixed issue where the `state-svc` process would not be shut down correctly.
* Fixed issue where `state clean uninstall` would say it succeeded but the State
  Tool would still be installed.

### Changed

* Several performance enhancements have been made affecting all parts of the
  State Tool.
* Activating an already activated project won't error out anymore.
* The local project is no longer affected if `state install` fails.

### Removed

* The `-c` flag has been removed from `state activate` as this is now handled
  by `state exec`.

## 0.32.2

### Fixed

* Fixed issue where auto-update could not complete for certain older versions

## 0.32.1

### Fixed

* Fixed issue that could sometimes cause recursion in our logging

## 0.32.0

### Added

* Added PPM and PIP shims to help educate people about the State Tool.
* Added support for Ruby projects

## 0.31.1

### Fixed

* Fixed issue where a failed solve was reported incorrectly.

## 0.31.0

### Changed

* More progress indicators are now given when sourcing runtimes and installing
  packages.
* Package operations are now much faster
* Binary sizes have been significantly reduced
* You no longer need to start a new shell when installing the State Tool (
  provided you're running an interactive session)

## 0.30.7

### Fixed

* Fixed issue where environment would not always be sourced properly

## 0.30.6

### Fixed

* Fixed issue where certain runtime executables could not be resolved

## 0.30.5

### Changed

* Recursion has been disabled while we improve the mechanic for a future version

## 0.30.4

### Fixed

* Fixed recursion issue when running certain State Tool commands

## 0.30.3

### Changed

* Enriched the installer with analytics to allow us to diagnose installation
  failures

## 0.30.2

### Fixed

* Fixed issue where State Tool sometimes could not identify its service daemon

## 0.30.1

### Fixed

* Fixed issue where our analytics events would send the full executable paths

## 0.30.0

### Added

* New Command: `state learn`. Opens
  the [State Tool Cheat Sheet](https://platform.activestate.com/state-tool-cheat-sheet)
  in your browser.

### Changed

* The install and activate user experience have been overhauled to be much more
  concise and avoid unnecessary prompts.
* Several performance enhancements have been made. Note that some of these will
  require at least one more release before they can realise their potential.
* Running `state update` will now immediately perform the update, rather than
  deferring it to a background process.
* State Tool should now attempt to use the latest version available for a given
  language, when initializing a project.

### Fixed

* Fixed issue where on macOS the `state` executable would sometimes not be added
  to your PATH.
* Resolved issue where `state exec` or certain invocations of the language
  runtime could lead to recursion errors.
* Fixed issues where sometimes State Tool would say it have a new version
  available when it didn't.

## 0.29.5

### Fixed

- Fixed race condition in anonymized analytics

## 0.29.4

### Changed

- Improved error reporting to help direct stability improvements

## 0.29.3

### Fixed

- Fixed race condition that could lead to logs being written to stderr

## 0.29.2

### Fixed

- Uninstalling no longer leaves a stale executable

## 0.29.1

### Fixed

- Auto updating from earlier versions no longer results in error

## 0.29.0

### Added

- Package management is now performed only locally, meaning you have
  to `state push` your changes back to your project when you are ready to save
  them.
- Enhanced error reporting when attempting package operations on an out of sync
  project ([PR #1353](https://github.com/ActiveState/cli/pull/1353))
- Enhanced error reporting for errors that occur when cloning a project's
  associated git
  repository ([PR #1351](https://github.com/ActiveState/cli/pull/1351))
- The State Tool now comes with a preview of the ActiveState Desktop
  application, which facilitates shortcuts to commonly used actions, including
  activating your projects.
- You can now switch to specific State Tool versions by
  running `state update --set-version <version>` ([PR #1385](https://github.com/ActiveState/cli/pull/1385))

### Changed

- Enhanced error reporting for errors that occur when cloning a project's
  associated git
  repository ([PR #1351](https://github.com/ActiveState/cli/pull/1351))

### Removed

- We no longer produce 32bit Windows builds of the State Tool

### Fixed

- Removed unwanted output (eg. `%!s(<nil>)`) when running scripts
  ([PR #1354](https://github.com/ActiveState/cli/pull/1354))
- Fixed issue where `state clean uninstall` would not remove expected files on
  Windows ([PR #1349](https://github.com/ActiveState/cli/pull/1349))
- Fixed a rare case where the configuration file can get corrupted when two
  processes access it
  simultaneously.  ([PR #1370] (https://github.com/ActiveState/cli/pull/1370))

## 0.28.1

### Fixed

* Fixed package installs / uninstalls not using the
  cache ([PR #1331](https://github.com/ActiveState/cli/pull/1331))

## 0.28.0

### Changed

- New runtimes are installed in parallel and 2-4 times faster.
  ([PR #1275](https://github.com/ActiveState/cli/pull/1275))

### Fixed

- `state push` updates project name in activestate.yaml.
  ([PR1297](https://github.com/ActiveState/cli/pull/1297))

## 0.27.1

### Fixed

- Fixed issue where `state uninstall` would not completely remove package files
  ([PR #1304](https://github.com/ActiveState/cli/pull/1304))

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

- The `--replace` flag for `state activate` is now deprecated in favour
  of `state pull --set-project`
