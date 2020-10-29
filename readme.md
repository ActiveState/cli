# State Tool
State Tool is the Command Line Interface for accessing [ActiveState Platform](https://www.activestate.com/products/platform/) functionality. It is a cross-platform tool written in Go supporting Windows, Linux and Mac. We're working on extending support to more  versions and distributions in tandem with the Platform (feel free to let us know if you have issues in your favorite OS). 

State Tool helps with mananging the package dependencies of a language runtime, for the languages supported by the ActiveState Platform (Dynamic Open Source Languages like Python, Perl and Tcl). It functions as a package manager (of sorts, more development is ongoing in this area) and a virtual environment for developers that isolates their project environment and allows easy switching between different projects on the same machine. Additional neat developer features of State Tool includes cross-platform scripting and support for secrets. State Tool also has functionality to support deployment of the language runtimes (supported by ActiveState like CEs or Custom Runtimes created by the users on the Platform) into environments like VMs and containers used in CI/CD systems. 

State Tool has a stated goal of "Replacing the Makefile". We're making progress, and always open for new ideas/suggestions/issue reports and code contributions. 
 
## Installation

### Linux & macOS
In your favourite terminal:

```
sh <(curl -q https://platform.activestate.com/dl/cli/install.sh)
```

### Windows
In Powershell with Administrator privileges:

```
IEX(New-Object Net.WebClient).downloadString('https://platform.activestate.com/dl/cli/install.ps1')
```

## Usage
For usage information please refer to the [State Tool Documentation](http://docs.activestate.com/platform/state/).

## Development

### Requirements
* Go 1.13 or above
* [packr](https://github.com/gobuffalo/packr): `go get -u github.com/gobuffalo/packr/...`

### Building & Testing
* **Building:** `state run build`
   * The built executable will be stored in the `build` directory
* **Testing:**
   * **Unit tests\*:** `state run test`
   * **Integration tests:** `state run integration-tests`

<sup>
* Our unit tests are in a state of slowly being converted to standalone
 integration tests, meaning that while we refer to them as unit tests
 they still contain a lot of tests that are better described as integration tests.
</sup>

### Refactoring
Our codebase has various refactorings underway that are too large to land
in a single PR, as such please keep the following guidelines in mind when
contributing.

* Error handling is slowly being refactored to retire our home brewed
 failures package in favour of conventional Go errors, please refer to
 [docs/errors.md](docs/errors.md) for more information.
* Commands registered under the [state/](state/) folder are using our legacy
  command architecture, all future commands should use the
  [cmd/state/internal/cmdtree](cmd/state/internal/cmdtree) architecture.
