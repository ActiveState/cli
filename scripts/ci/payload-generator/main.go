package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/scripts/ci/payload-generator/paygen"
)

var logErr = func(msg string, vals ...any) {
	fmt.Fprintf(os.Stderr, msg, vals...)
	fmt.Fprintf(os.Stderr, "\n")
}

// The payload-generator is an intentionally very dumb runner that just copies some files around.
// This could just be a bash script if not for the fact that bash can't be linked with our type system.
func main() {
	if err := run(); err != nil {
		logErr("%s", err)
		os.Exit(1)
	}
}

func run() error {
	root := environment.GetRootPathUnsafe()
	buildDir := filepath.Join(root, "build")
	payloadDir := filepath.Join(buildDir, "payload")

	return paygen.GeneratePayload(buildDir, payloadDir)
}
