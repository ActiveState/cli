[![CircleCI](https://circleci.com/gh/ActiveState/ActiveState-CLI.svg?style=shield&circle-token=e439410d217d72704e82808bdc3bbe78b6ecbf21)](https://circleci.com/gh/ActiveState/ActiveState-CLI)

# Installation

 1. Make sure you have Go installed (version 1.9 ideally)
 2. Clone this repository under `$GOPATH/src/ActiveState/ActiveState-CLI`
 3. Run `dep ensure`
 4. Run `make build`

# Development Workflow

 * Currently it is recommended that you use vscode for development
 * You likely have to specify your GOPATH in vscode with `"go.gopath": "~/.go"`
 * Ensure gocode is using the latest version: ```go get -u gopkg.in/nsf/gocode.v0```
 * If on Linux, you may have to run the following for godef to work properly: ```sudo ln -s /usr/lib/go /usr/lib/go-1.9```
 * ALWAYS `go fmt` before commit
 * Do not commit untested code (meaning tests exist for the code, and the tests pass)
 * To run code without building run `go run state/state.go <your command>`
 * To run all tests use `make test`
