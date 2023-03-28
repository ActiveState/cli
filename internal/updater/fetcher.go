package updater

import (
	"crypto/sha256"
	"fmt"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httpreq"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

const CfgUpdateTag = "update_tag"

type Fetcher struct {
	httpreq *httpreq.Client
	an      analytics.Dispatcher
}

func NewFetcher(an analytics.Dispatcher) *Fetcher {
	return &Fetcher{httpreq.New(), an}
}

func (f *Fetcher) Fetch(update *AvailableUpdate, targetDir string) error {
	logging.Debug("Fetching update: %s", update.url)
	b, _, err := f.httpreq.Get(update.url)
	if err != nil {
		msg := fmt.Sprintf("Fetch %s failed", update.url)
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}

	if err := verifySha(b, update.Sha256); err != nil {
		msg := "Could not verify sha256"
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}

	logging.Debug("Preparing target dir: %s", targetDir)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		msg := "Could not create target dir"
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}

	isEmpty, err := fileutils.IsEmptyDir(targetDir)
	if err != nil {
		msg := "Could not verify if target dir is empty"
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}
	if !isEmpty {
		msg := "Target dir is not empty"
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}

	a := blobUnarchiver(b)
	if err := a.Unarchive(targetDir); err != nil {
		msg := "Unarchiving failed"
		f.analyticsEvent(update.Version, msg)
		return errs.Wrap(err, msg)
	}

	return nil
}

func (f *Fetcher) analyticsEvent(version, msg string) {
	f.an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateDownload, anaLabelFailed, &dimensions.Values{
		TargetVersion: p.StrP(version),
		Error:         p.StrP(msg),
	})
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
