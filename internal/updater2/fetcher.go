package updater2

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/unarchiver"
)

type Fetcher struct {
	update    *Update
	targetDir string
	httpreq   *httpreq.Client
}

func NewFetcher(update *Update, targetDir string) *Fetcher {
	return &Fetcher{update, targetDir, httpreq.New()}
}

func (f *Fetcher) Run() error {
	b, err := f.httpreq.Get(f.update.url)
	if err != nil {
		return errs.Wrap(err, "Fetch %s failed", f.update.url)
	}

	if err := fileutils.MkdirUnlessExists(f.targetDir); err != nil {
		return errs.Wrap(err, "Could not create target dir: %s", f.targetDir)
	}

	isEmpty, err := fileutils.IsEmptyDir(f.targetDir)
	if err != nil {
		return errs.Wrap(err, "Could not verify if target dir is empty")
	}
	if !isEmpty {
		return errs.Wrap(err, "Target dir is not empty: %s", f.targetDir)
	}

	zip := unarchiver.NewZipBlob(b)
	if err := zip.Unarchive(f.targetDir); err != nil {
		return errs.Wrap(err, "Unarchiving failed")
	}

	return nil
}
