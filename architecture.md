# Architecture

This document describes the high-level architecture of the State Tool and
related applications (currently: State Service, State Tray, State Installer, and
State Update Dialog).

## Broad Descriptions

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

### cmd/{app_name}/

Individual "main" applications.

#### cmd/{app_name}/internal/

Packages used exclusively for the parent application.

### docs/

Developer-focused documentation.

### installers/

Shell scripts wrapping the installer application for user convenience.

### internal/

Packages that are made available for use by any application, but are restricted
from use by external code.

#### internal/runbits/

Packages that are made available for use by "runner" packages. In essence,
`internal/runbits/internal/runners`.

#### internal/runners/

Packages that provide command behavior for the State Tool application.

### locale/

Localization keys and associated values.

### pkg/

Packages that are made available for use by any application, including external
code. The packages that are currently located here are not intended for use by
external code, and will eventually be moved into the `internal/` directory.

### scripts/

Helper scripts.

### test/integration/

End-to-end tests.

### vendor/

Go language dependencies.
