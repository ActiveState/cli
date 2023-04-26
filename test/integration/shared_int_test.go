package integration

import (
	"os"

	"github.com/ActiveState/cli/internal/logging"
)

func init() {
	if os.Getenv("VERBOSE") == "true" || os.Getenv("VERBOSE_TESTS") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}
}
