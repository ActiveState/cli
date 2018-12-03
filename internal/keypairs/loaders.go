package keypairs

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailLoad indicates a failure when loading something.
	FailLoad = failures.Type("keypairs.fail.load")

	// FailLoadUnknown represents a failure to successfully load Keypair for for some unknown reason.
	FailLoadUnknown = failures.Type("keypairs.fail.load.unknown", FailLoad)

	// FailLoadNotFound represents a failure to successfully find a Keypair file for loading.
	FailLoadNotFound = failures.Type("keypairs.fail.load.not_found", FailLoad)

	// FailLoadFileTooPermissive represents a failure wherein a Keypair file's permissions
	// (it's octet) are too permissive.
	FailLoadFileTooPermissive = failures.Type("keypairs.fail.load.too_permissive", FailLoad)
)

// Load will attempt to load a Keypair using private and public-key files from the
// user's file system; specifically from the config dir. It is assumed that this
// keypair file has no passphrase, even if it is encrypted.
func Load(keyName string) (Keypair, *failures.Failure) {
	var kp Keypair
	keyFilename := filepath.Join(config.GetDataDir(), keyName+".key")
	failure := validateKeyFile(keyFilename)
	if failure == nil {
		kp, failure = loadAndParseKeypair(keyFilename)
	}
	return kp, failure
}

func validateKeyFile(keyFilename string) *failures.Failure {
	keyFileStat, err := os.Stat(keyFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return FailLoadNotFound.New("keypairs_err_load_not_found")
		}
		return FailLoad.Wrap(err)
	}
	// allows u+rw only
	if keyFileStat.Mode()&(0177) > 0 {
		return FailLoadFileTooPermissive.New("keypairs_err_load_requires_mode", keyFilename, "0600")
	}
	return nil
}

func loadAndParseKeypair(keyFilename string) (Keypair, *failures.Failure) {
	keyFileBytes, err := ioutil.ReadFile(keyFilename)
	if err != nil {
		return nil, FailLoadUnknown.Wrap(err)
	}
	return ParseRSA(string(keyFileBytes))
}
