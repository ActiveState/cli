package projectfile

import (
	"fmt"
	"strings"
)

var (
	pFileYAMLValid = pFileYAML{"CLI_BUILDFLAGS", "-ldflags=\"-s -w\""}
)

type pFileYAML struct {
	firstKey, firstVal string
}

func (y pFileYAML) asLongYAML() []byte {
	return applyToYAML(`
project: https://platform.activestate.com/ActiveState/cli?branch=main
constants:
  - name: %s
    value: %s
  - name: CLI_PKGS
    value: ./cmd/state
  - name: SET_ENV
    description: The environment settings used throughout our project
    value: |
      GOFLAGS='-mod=vendor'
      BUILD_TARGET_DIR=$constants.BUILD_TARGET_PREFIX_DIR/${GOARCH#amd64}
  - name: SCRIPT_EXT
    if: ne .OS.Name "Windows"
    value: .sh
scripts:
  - name: preprocess
    language: bash
    description: Generates assets required by the project that aren't just specific to the build
    value: |
      $constants.SET_ENV
      go run scripts/constants-generator/main.go -- internal/constants/generated.go
  - name: build
    language: bash
    description: Builds the project with the host OS as the target OS.
    value: |
      $constants.SET_ENV
      go build -tags "$GO_BUILD_TAGS" -o $BUILD_TARGET_DIR/$constants.BUILD_TARGET $constants.CLI_BUILDFLAGS $constants.CLI_PKGS
events:
  - name: activate
    value: |
      if ! type "go" &> /dev/null; then
        echo "go is not installed. Please install Go version 1.11 or above."
        exit 1
      fi
      git config core.hooksPath .githooks
  - name: file-changed
    scope: ["internal/locale/locales"]
    value: build
`, y)
}

func (y pFileYAML) asShortYAML() []byte {
	return applyToYAML(`
project: https://platform.activestate.com/ActiveState/cli?branch=main
constants:
  - %s: %s
  - CLI_PKGS: ./cmd/state
  - SET_ENV: |
      GOFLAGS='-mod=vendor'
      BUILD_TARGET_DIR=$constants.BUILD_TARGET_PREFIX_DIR/${GOARCH#amd64}
  - name: SCRIPT_EXT
    if: ne .OS.Name "Windows"
    value: .sh
scripts:
  - preprocess: |
      $constants.SET_ENV
      go run scripts/constants-generator/main.go -- internal/constants/generated.go
  - build: |
      $constants.SET_ENV
      go build -tags "$GO_BUILD_TAGS" -o $BUILD_TARGET_DIR/$constants.BUILD_TARGET $constants.CLI_BUILDFLAGS $constants.CLI_PKGS
events:
  - activate: |
      if ! type "go" &> /dev/null; then
        echo "go is not installed. Please install Go version 1.11 or above."
        exit 1
      fi
      git config core.hooksPath .githooks
  - name: file-changed
    scope: ["internal/locale/locales"]
    value: build
`, y)
}

func applyToYAML(d string, vs pFileYAML) []byte {
	return []byte(fmt.Sprintf(strings.TrimSpace(d)+"\n", vs.firstKey, vs.firstVal))
}
