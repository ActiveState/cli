package keypairs

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
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

	// FailSave indicates a failure when saving something.
	FailSave = failures.Type("keypairs.fail.save")

	// FailSaveFile indicates a failure when saving a keypair file.
	FailSaveFile = failures.Type("keypairs.fail.save.file")

	// FailDeleteFile indicates a failure when deleting a keypair file.
	FailDeleteFile = failures.Type("keypairs.fail.delete.file")
)

// Load will attempt to load a Keypair using private and public-key files from the
// user's file system; specifically from the config dir. It is assumed that this
// keypair file has no passphrase, even if it is encrypted.
func Load(keyName string) (Keypair, *failures.Failure) {
	var kp Keypair
	keyFilename := localKeyFilename(keyName)
	failure := validateKeyFile(keyFilename)
	if failure == nil {
		kp, failure = loadAndParseKeypair(keyFilename)
	}
	return kp, failure
}

// Save will save the unencrypted and encoded private key to a local config file. The filename will be
// the value of `keyName` and suffixed with `.key`.
func Save(kp Keypair, keyName string) *failures.Failure {
	err := ioutil.WriteFile(localKeyFilename(keyName), []byte(kp.EncodePrivateKey()), 0600)
	if err != nil {
		return FailSaveFile.Wrap(err)
	}
	return nil
}

// Delete will delete an unencrypted and encoded private key from the local config directory. The base
// filename (sans suffix) must be provided.
func Delete(keyName string) *failures.Failure {
	filename := localKeyFilename(keyName)
	if fileutils.FileExists(filename) {
		if err := os.Remove(filename); err != nil {
			return FailDeleteFile.Wrap(err)
		}
	}
	return nil
}

// LoadWithDefaults will call Load with the default key name (i.e. constants.KeypairLocalFileName).
func LoadWithDefaults() (Keypair, *failures.Failure) {
	return Load(constants.KeypairLocalFileName)
}

// SaveWithDefaults will call Save with the provided keypair and the default key name
// (i.e. constants.KeypairLocalFileName).
func SaveWithDefaults(kp Keypair) *failures.Failure {
	return Save(kp, constants.KeypairLocalFileName)
}

// DeleteWithDefaults will call Delete with the default key name (i.e. constants.KeypairLocalFileName).
func DeleteWithDefaults() *failures.Failure {
	return Delete(constants.KeypairLocalFileName)
}

func localKeyFilename(keyName string) string {
	return filepath.Join(config.ConfigPath(), keyName+".key")
}

func validateKeyFile(keyFilename string) *failures.Failure {
	if !fileutils.FileExists(keyFilename) {
		return FailLoadNotFound.New("keypairs_err_load_not_found")
	}

	keyFileStat, err := os.Stat(keyFilename)
	if err != nil {
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
