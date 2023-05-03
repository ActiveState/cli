package integration

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/assert"
)

func init() {
	if os.Getenv("VERBOSE") == "true" || os.Getenv("VERBOSE_TESTS") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}
}

// AssertValidJSON asserts that the previous command emitted valid JSON and did not attempt to emit
// any non-JSON/structured output.
// This should only be called after a command has executed and all output is available.
func AssertValidJSON(t *testing.T, cp *termtest.ConsoleProcess) {
	snapshot := cp.TrimmedSnapshot()
	if runtime.GOOS != "windows" {
		assert.True(t, json.Valid([]byte(snapshot)), "The command produced invalid JSON/structured output:\n"+snapshot)
	} else {
		// Windows can trim the last byte for some reason.
		assert.True(
			t,
			json.Valid([]byte(snapshot)) || json.Valid([]byte(snapshot+"}")) || json.Valid([]byte(snapshot+"]")),
			"The command produced invalid JSON/structured output:\n"+snapshot,
		)
	}
	if strings.Contains(snapshot, `"errors":[`) {
		assert.NotContains(t, snapshot, `output not supported`, "The command attempted to emit non-JSON/structured output:\n"+snapshot)
	}
}
