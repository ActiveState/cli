[![CircleCI](https://circleci.com/gh/ActiveState/cli.svg?style=shield&circle-token=e439410d217d72704e82808bdc3bbe78b6ecbf21)](https://circleci.com/gh/ActiveState/cli)

# Installation

 1. Install the state tool: https://www.activestate.com/products/platform/state-tool/
 2. Run `state activate` to get into an activated state

# Development Workflow

 * ALWAYS `go fmt` before commit
 * Do not commit untested code (meaning tests exist for the code, and the tests pass)
 * To build use `state run build`
 * To run all tests use `state run test`
 * Run `state scripts` to see all available commands
