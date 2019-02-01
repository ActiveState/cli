[![CircleCI](https://circleci.com/gh/ActiveState/cli.svg?style=shield&circle-token=e439410d217d72704e82808bdc3bbe78b6ecbf21)](https://circleci.com/gh/ActiveState/cli)

# Installation

bogus

 1. Install the following dependencies:
   * Go 1.9 or later
   * dep - https://github.com/golang/dep#installation
   * upx - https://upx.github.io/
 2. Clone this repository under `$GOPATH/src/github.com/ActiveState/cli`
 3. Run `git submodule init` and `git submodule update` (for tests)
 4. Run `state activate` to get into an activated state
 5. Run `build`

# Development Workflow

 * Currently it is recommended that you use vscode for development
 * You likely have to specify your GOPATH in vscode with `"go.gopath": "~/.go"`
 * Ensure gocode is using the latest version: ```go get -u gopkg.in/nsf/gocode.v0```
 * If on Linux, you may have to run the following for godef to work properly: ```sudo ln -s /usr/lib/go /usr/lib/go-1.9```
 * ALWAYS `go fmt` before commit
 * Do not commit untested code (meaning tests exist for the code, and the tests pass)
 * To run code without building run `go run state/state.go <your command>`
 * To run all tests use `test`

# Deploying an Update

Running `build` from an activated state will generate the update bits.

When update bits exist you can deploy them using `deploy-updates`.

You will need to set the following env vars:
 * AWS_ACCESS_KEY_ID
 * AWS_SECRET_ACCESS_KEY

The rest of the configuration is hard-coded in our activestate.yaml and should generally not be changed.

# Deploying Artifacts

To deploy artifacts you first have to generate them, to do so run `generate-artifacts`. For this to work you must
provide source files inside the scripts/artifact-generator/source directory. Follow the folder structure and instructions
provided within them.

Once generated you can deploy the artifacts using 'deploy-artifacts'. This requires AWS credentials the same as for
deploying an update.

