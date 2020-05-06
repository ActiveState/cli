[![CircleCI](https://circleci.com/gh/ActiveState/cli.svg?style=shield&circle-token=e439410d217d72704e82808bdc3bbe78b6ecbf21)](https://circleci.com/gh/ActiveState/cli)

-- Introduction to follow --

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
* Our unit tests are in an awkward state of slowly being converted to
standalone integration tests, meaning that while we refer to them as unit
tests they still contain a lot of tests that are better described as
integration tests.
</sup>

### Refactoring

Our codebase has various refactorings underway that are too large to land
in a single PR, as such please keep the following guidelines in mind when
contributing.

* Error handling is slowly being refactored to retire our home brewed
 failures package in favour of conventional Go errors, please refer to
 [docs/errors.md] for more information.
* Commands registered under the [state/] folder are using our legacy
  command architecture, all future commands should use the [cmd/state/internal/cmdtree]
  architecture.
