package secrets

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
)

func loadKeypairFromConfigDir() (keypairs.Keypair, *failures.Failure) {
	kp, failure := keypairs.LoadWithDefaults()
	if failure != nil {
		if failure.Type.Matches(keypairs.FailLoadNotFound) || failure.Type.Matches(keypairs.FailKeypairParse) {
			return nil, failure.Type.New("keypair_err_require_auth")
		}
		return nil, failure
	}
	return kp, nil
}
