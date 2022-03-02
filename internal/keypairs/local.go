package keypairs

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/ci/gcloud"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Configurable interface {
	ConfigPath() string
	Close() error
	Set(s string, i interface{}) error
	GetString(s string) string
}

// Load will attempt to load a Keypair using private and public-key files from
// the user's file system; specifically from the config dir. It is assumed that
// this keypair file has no passphrase, even if it is encrypted.
func Load(cfg Configurable, keyName string) (Keypair, error) {
	keyFilename := LocalKeyFilename(cfg.ConfigPath(), keyName)
	if err := validateKeyFile(keyFilename); err != nil {
		return nil, err
	}
	return loadAndParseKeypair(keyFilename)
}

// Save will save the unencrypted and encoded private key to a local config
// file. The filename will be the value of `keyName` and suffixed with `.key`.
func Save(cfg Configurable, kp Keypair, keyName string) error {
	keyFileName := LocalKeyFilename(cfg.ConfigPath(), keyName)
	err := ioutil.WriteFile(keyFileName, []byte(kp.EncodePrivateKey()), 0600)
	if err != nil {
		return errs.Wrap(err, "WriteFile failed")
	}
	return nil
}

// Delete will delete an unencrypted and encoded private key from the local
// config directory. The base filename (sans suffix) must be provided.
func Delete(cfg Configurable, keyName string) error {
	filename := LocalKeyFilename(cfg.ConfigPath(), keyName)
	if fileutils.FileExists(filename) {
		if err := os.Remove(filename); err != nil {
			return errs.Wrap(err, "os.Remove %s failed", filename)
		}
	}
	return nil
}

// LoadWithDefaults will call Load with the default key name (i.e.
// constants.KeypairLocalFileName). If the key override is set
// (constants.PrivateKeyEnvVarName), that value will be parsed directly.
func LoadWithDefaults(cfg Configurable) (Keypair, error) {
	key, err := gcloud.GetSecret(constants.PrivateKeyEnvVarName)
	if err != nil && !errors.Is(err, gcloud.ErrNotAvailable{}) {
		return nil, errs.Wrap(err, "gcloud.GetSecret failed")
	}
	if err == nil && key != "" {
		logging.Debug("Using private key sourced from gcloud")
		return ParseRSA(key)
	}

	if key := os.Getenv(constants.PrivateKeyEnvVarName); key != "" {
		logging.Debug("Using private key sourced from environment")
		return ParseRSA(key)
	}

	return Load(cfg, constants.KeypairLocalFileName)
}

// SaveWithDefaults will call Save with the provided keypair and the default
// key name (i.e. constants.KeypairLocalFileName). The operation will fail when
// the key override is set (constants.PrivateKeyEnvVarName).
func SaveWithDefaults(cfg Configurable, kp Keypair) error {
	if hasKeyOverride() {
		return locale.NewInputError("keypairs_err_override_with_save")
	}

	return Save(cfg, kp, constants.KeypairLocalFileName)
}

// DeleteWithDefaults will call Delete with the default key name (i.e.
// constants.KeypairLocalFileName). The operation will fail when the key
// override is set (constants.PrivateKeyEnvVarName).
func DeleteWithDefaults(cfg Configurable) error {
	if hasKeyOverride() {
		return locale.NewInputError("keypairs_err_override_with_delete")
	}

	return Delete(cfg, constants.KeypairLocalFileName)
}

// LocalKeyFilename returns the full filepath for the given key name
func LocalKeyFilename(configPath, keyName string) string {
	return filepath.Join(configPath, keyName+".key")
}

func loadAndParseKeypair(keyFilename string) (Keypair, error) {
	keyFileBytes, err := ioutil.ReadFile(keyFilename)
	if err != nil {
		return nil, errs.Wrap(err, "ReadFile %s failed", keyFilename)
	}
	return ParseRSA(string(keyFileBytes))
}

func hasKeyOverride() bool {
	if os.Getenv(constants.PrivateKeyEnvVarName) != "" {
		logging.Debug("Has key override from env")
		return true
	}

	tkn, err := gcloud.GetSecret(constants.PrivateKeyEnvVarName)
	if err != nil && !errors.Is(err, gcloud.ErrNotAvailable{}) {
		logging.Error("Could not retrieve gcloud secret: %v", err)
	}
	if err == nil && tkn != "" {
		logging.Debug("Has key override from gcloud")
		return true
	}

	return false
}
