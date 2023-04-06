package validate

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Disabled on macOS due to non-standards compliant signing certificate") // DX-1451
	}
	attestationFile := filepath.Join(osutil.GetTestDataDir(), "bzip2_attestation.json")
	err := Attestation(attestationFile)
	assert.NoError(t, err, "joined message: %s", errs.JoinMessage(err))

	attestationFile = filepath.Join(osutil.GetTestDataDir(), "bzip2_attestation_bad_cert.json")
	err = Attestation(attestationFile)
	assert.NoError(t, err, "joined message: %s", errs.JoinMessage(err))

	attestationFile = filepath.Join(osutil.GetTestDataDir(), "bzip2_attestation_bad_sig.json")
	err = Attestation(attestationFile)
	assert.NoError(t, err, "joined message: %s", errs.JoinMessage(err))
}
