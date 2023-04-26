package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/assert"
)

func init() {
	if os.Getenv("VERBOSE") == "true" || os.Getenv("VERBOSE_TESTS") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}
}

// ExpectJSONKeys looks for JSON output, asserts that each key is present, and then asserts that no
// non-JSON/structured output is present.
// If you want to test for JSON output other than just keys, do not call this function, as it
// consumes console output.
func ExpectJSONKeys(t *testing.T, cp *termtest.ConsoleProcess, keys ...string) {
	cp.Expect("{", 60*time.Second)
	for _, key := range keys {
		cp.Expect(fmt.Sprintf(`"%s":`, key))
	}
	cp.Expect("}")
	AssertNoPlainOutput(t, cp)
}

// AssertNoPlainOutput asserts that the previous command did not attempt to emit any
// non-JSON/structured output.
// This is called automatically by ExpectJSONKeys, so you only need to call this if you have
// manually expected JSON output.
func AssertNoPlainOutput(t *testing.T, cp *termtest.ConsoleProcess) {
	snapshot := cp.TrimmedSnapshot()
	if strings.Contains(snapshot, `"errors":[`) {
		assert.NotContains(t, snapshot, `output not supported`, "The command attempted to emit non-JSON/structured output")
	}
}
