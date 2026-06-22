package orgkey

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

// cacheFileName is the on-disk cache file (the validated contract JSON) under the config dir.
const cacheFileName = "private_ingredient_orgkey.json"

func (p *provider) diskCacheEnabled() bool {
	return p.cfg.GetBool(constants.PrivateIngredientCacheKeyConfig)
}

func (p *provider) cachePath() string {
	return filepath.Join(p.cfg.ConfigPath(), cacheFileName)
}

// readDiskCache returns the cached contract bytes if a usable cache file exists.
// A missing file is normal (first run); an unsafe or unreadable file is ignored
// with a warning so the run falls back to a fresh fetch.
func (p *provider) readDiskCache() ([]byte, bool) {
	path := p.cachePath()
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if err := checkCacheMode(info); err != nil {
		logging.Warning("Ignoring on-disk org key cache: %v", errs.JoinMessage(err))
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		logging.Warning("Could not read on-disk org key cache: %v", errs.JoinMessage(err))
		return nil, false
	}
	return b, true
}

// writeDiskCache persists the validated contract for reuse by later runs,
// owner-readable only.
func (p *provider) writeDiskCache(raw []byte) error {
	if err := os.WriteFile(p.cachePath(), raw, 0600); err != nil {
		return errs.Wrap(err, "unable to write org key cache")
	}
	return nil
}
