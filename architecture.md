# Architecture

This document describes the high-level architecture of the State Tool and
related applications (currently: State Service, State Tray, State Installer, and
State Update Dialog).

## Applications Overviews

### State Tool

The State Tool is a CLI application providing access to the ActiveState Platform
features. It is the primary consumer of ActiveState Platform APIs which provide
runtime, library and application builds. Using these reproducible, indemnified,
and self-serve artifacts, the State Tool modifies the host environment in order
to provide package and virtual environment management.

### State Service

The State Service provides the State Tool with a central point of access to the
ActiveState Platform APIs, host environment modification, and other features
like caching.

### State Tray

The State Tray provides a GUI element which helps to bring important information
to the visual forefront of a user's experience (e.g. available updates, host
environment status, etc.), as well as quick access launchers (e.g. ActiveState
Platform Dashboard and support links, project entry points, etc.).

### State Installer

The State Installer provides a cross-platform installer for the suite of State
Tool applications. Shell script wrappers are provided as a convenience.

### State Update Dialog

The State Update Dialog provides a cross-platform updater for the suite of State
Tool Applications.

## Directory Structure

### assets/

Various files (e.g. images, templates, etc.) used by any application.

### build/

Artifacts resulting from building applications.

### cmd/*/

Individual "main" applications.

#### cmd/*/internal/

Packages used exclusively for the parent application.

### docs/

Developer-focused documentation.

### .github/workflows-src/

YAML-formatted config that is processed by `ytt`
(https://github.com/vmware-tanzu/carvel-ytt) to produce yml files used for CI
(stored in `.github/workflows/`).

### installers/

Shell scripts wrapping the installer application for user convenience.

### internal/

Packages that are made available for use by any application, but are restricted
from use by external code.

#### internal/runbits/

Packages that are made available for use by "runner" packages. In essence,
`internal/runners/internal/runbits`.

#### internal/runners/

Packages that provide command behavior for the State Tool application.

### locale/

Localization keys and associated values.

### pkg/

Packages that are made available for use by any application, including external
code. Note we are changing our strategy with regards to these packages, 
and these will eventually be moved into the `internal/` directory.

#### pkg/cmdlets/

Packages that are made available for use by "runner" packages. Synonymous with
`internal/runbits/`, and all new runner-common packages should be placed there.

#### pkg/platform/

Packages focused on interacting with platform API's. Much of the behavior is
generated, but there are also critical components providing platform-dependent
client-side logic (e.g. `pkg/platform/runtime`).

#### pkg/{project,projectfile}/

Packages that provide setup and interaction with the activestate.yaml files.

### scripts/

Helper scripts for development and deployment processes.

### test/integration/

End-to-end tests.

### vendor/

Go language dependencies.

## Code Change Entry Points

While the entry point of control flow in every application is some version of a
main file (e.g. `cmd/state/main.go`), the code that tends to be modified the
most lives in `internal/runners/`, `pkg/`, and `internal/`.

State Tool interactions are routed from command line arguments to the
appropriate "runner" where the runner can be thought of as a "handler" or
"controller". Various output formats (which can be thought of as "views") are
available and defined by logic within `internal/output`. Similarly, commonly
used logic is defined in packages like `pkg/project`.

Currently, the State Service is in its beginning stages of importance. As it
matures, it will become more central to code changes. For the State Service, the
interactions are GraphQL queries that are routed to handlers.

## activestate.yaml

The project file contains information that connects a project to the platform,
and also provides local behavior similar to `make`. The most commonly used
scripts are `preprocess` and `build`. Run `state scripts` for a full listing.
