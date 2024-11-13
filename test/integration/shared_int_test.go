package integration

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

func init() {
	if os.Getenv("VERBOSE") == "true" || os.Getenv("VERBOSE_TESTS") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}
}

// AssertValidJSON asserts that the previous command emitted valid JSON and did not attempt to emit
// any non-JSON/structured output.
// This should only be called after a command has executed and all output is available.
func AssertValidJSON(t *testing.T, cp *e2e.SpawnedCmd) []byte {
	output := cp.StrippedSnapshot()
	if strings.Contains(output, `"errors":[`) {
		assert.NotContains(t, output, `output not supported`, "The command attempted to emit non-JSON/structured output:\n"+output)
	}
	if runtime.GOOS != "windows" {
		assert.True(t, json.Valid([]byte(output)), "The command produced invalid JSON/structured output:\n"+output)
		return []byte(output)
	} else {
		switch {
		case json.Valid([]byte(output)):
			return []byte(output)
		case json.Valid([]byte(output + "}")):
			return []byte(output + "}")
		case json.Valid([]byte(output + "]")):
			return []byte(output + "]")
		}
		t.Fatal("The command produced invalid JSON/structured output:\n" + output)
	}
	t.Fatal("Unreachable")
	return nil
}
