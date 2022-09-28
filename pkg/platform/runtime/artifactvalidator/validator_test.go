package artifactvalidator

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	attestationFile := filepath.Join(osutil.GetTestDataDir(), "bzip2_attestation.json")
	err := ValidateAttestation(attestationFile)
	assert.NoError(t, err)
}
