package keypairs

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/ci/gcloud"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// FailLoad indicates a failure when loading something.
	FailLoad = failures.Type("keypairs.fail.load", failures.FailUser)

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

	// FailHasOverride indicates a failure when key override prevents
	// standard behavior.
	FailHasOverride = failures.Type("keypairs.fail.has_override")
)

// Load will attempt to load a Keypair using private and public-key files from
// the user's file system; specifically from the config dir. It is assumed that
// this keypair file has no passphrase, even if it is encrypted.
func Load(keyName string) (Keypair, error) {
	keyFilename := LocalKeyFilename(keyName)
	if fail := validateKeyFile(keyFilename); fail != nil {
		return nil, fail
	}
	return loadAndParseKeypair(keyFilename)
}

// Save will save the unencrypted and encoded private key to a local config
// file. The filename will be the value of `keyName` and suffixed with `.key`.
func Save(kp Keypair, keyName string) error {
	err := ioutil.WriteFile(LocalKeyFilename(keyName), []byte(kp.EncodePrivateKey()), 0600)
	if err != nil {
		return FailSaveFile.Wrap(err)
	}
	return nil
}

// Delete will delete an unencrypted and encoded private key from the local
// config directory. The base filename (sans suffix) must be provided.
func Delete(keyName string) error {
	filename := LocalKeyFilename(keyName)
	if fileutils.FileExists(filename) {
		if err := os.Remove(filename); err != nil {
			return FailDeleteFile.Wrap(err)
		}
	}
	return nil
}

// LoadWithDefaults will call Load with the default key name (i.e.
// constants.KeypairLocalFileName). If the key override is set
// (constants.PrivateKeyEnvVarName), that value will be parsed directly.
func LoadWithDefaults() (Keypair, error) {
	key, err := gcloud.GetSecret(constants.PrivateKeyEnvVarName)
	if err != nil && ! errors.Is(err, gcloud.ErrNotAvailable{}) {
		return nil, failures.FailNetwork.Wrap(err)
	}
	if err == nil && key != "" {
		logging.Debug("Using private key sourced from gcloud")
		return ParseRSA(key)
	}

	if key := os.Getenv(constants.PrivateKeyEnvVarName); key != "" {
		logging.Debug("Using private key sourced from environment")
		return ParseRSA(key)
	}

	return Load(constants.KeypairLocalFileName)
}

// SaveWithDefaults will call Save with the provided keypair and the default
// key name (i.e. constants.KeypairLocalFileName). The operation will fail when
// the key override is set (constants.PrivateKeyEnvVarName).
func SaveWithDefaults(kp Keypair) error {
	if hasKeyOverride() {
		return FailHasOverride.New("keypairs_err_override_with_save")
	}

	return Save(kp, constants.KeypairLocalFileName)
}

// DeleteWithDefaults will call Delete with the default key name (i.e.
// constants.KeypairLocalFileName). The operation will fail when the key
// override is set (constants.PrivateKeyEnvVarName).
func DeleteWithDefaults() error {
	if hasKeyOverride() {
		return FailHasOverride.New("keypairs_err_override_with_delete")
	}

	return Delete(constants.KeypairLocalFileName)
}

// LocalKeyFilename returns the full filepath for the given key name
func LocalKeyFilename(keyName string) string {
	return filepath.Join(config.ConfigPath(), keyName+".key")
}

func loadAndParseKeypair(keyFilename string) (Keypair, error) {
	keyFileBytes, err := ioutil.ReadFile(keyFilename)
	if err != nil {
		return nil, FailLoadUnknown.Wrap(err)
	}
	return ParseRSA(string(keyFileBytes))
}

func hasKeyOverride() bool {
	if os.Getenv(constants.PrivateKeyEnvVarName) != "" {
		return true
	}

	tkn, err := gcloud.GetSecret(constants.PrivateKeyEnvVarName)
	if err != nil && ! errors.Is(err, gcloud.ErrNotAvailable{}) {
		logging.Error("Could not retrieve gcloud secret: %v", err)
	}
	if err == nil && tkn != "" {
		return true
	}

	return false
}
