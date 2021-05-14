package updater

import (
	"crypto/sha256"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httpreq"
)

type Fetcher struct {
	httpreq *httpreq.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{httpreq.New()}
}

func (f *Fetcher) Fetch(update *AvailableUpdate, targetDir string) error {
	b, err := f.httpreq.Get(update.url)
	if err != nil {
		return errs.Wrap(err, "Fetch %s failed", update.url)
	}

	if err := verifySha(b, update.Sha256); err != nil {
		return errs.Wrap(err, "Could not verify sha256")
	}

	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		return errs.Wrap(err, "Could not create target dir: %s", targetDir)
	}

	isEmpty, err := fileutils.IsEmptyDir(targetDir)
	if err != nil {
		return errs.Wrap(err, "Could not verify if target dir is empty")
	}
	if !isEmpty {
		return errs.Wrap(err, "Target dir is not empty: %s", targetDir)
	}

	a := blobUnarchiver(b)
	if err := a.Unarchive(targetDir); err != nil {
		return errs.Wrap(err, "Unarchiving failed")
	}

	return nil
}

func verifySha(b []byte, sha string) error {
	h := sha256.New()
	h.Write(b)

	var computed = h.Sum(nil)
	var computedSha = fmt.Sprintf("%x", computed)
	var bytesEqual = computedSha == sha
	if !bytesEqual {
		return errs.New("sha256 did not match")
	}

	return nil
}
