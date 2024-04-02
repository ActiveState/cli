package updater

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"

	"github.com/ActiveState/cli/internal/analytics"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/retryhttp"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

const CfgUpdateTag = "update_tag"

type Fetcher struct {
	retryhttp *retryhttp.Client
	an        analytics.Dispatcher
}

func NewFetcher(an analytics.Dispatcher) *Fetcher {
	return &Fetcher{retryhttp.DefaultClient, an}
}

func (f *Fetcher) Fetch(update *UpdateInstaller, targetDir string) error {
	logging.Debug("Fetching update: %s", update.url)
	resp, err := f.retryhttp.Get(update.url)
	if err != nil {
		msg := fmt.Sprintf("Fetch %s failed", update.url)
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		if err2, ok := err.(net.Error); ok && err2.Timeout() {
			return locale.WrapInputError(err, "err_user_network_timeout", "", locale.Tr("err_user_network_solution", constants.ForumsURL))
		}
		msg := "Could not read response body"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	if err := verifySha(b, update.AvailableUpdate.Sha256); err != nil {
		msg := "Could not verify sha256"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	logging.Debug("Preparing target dir: %s", targetDir)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		msg := "Could not create target dir"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	isEmpty, err := fileutils.IsEmptyDir(targetDir)
	if err != nil {
		msg := "Could not verify if target dir is empty"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}
	if !isEmpty {
		msg := "Target dir is not empty"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	a := blobUnarchiver(b)
	if err := a.Unarchive(targetDir); err != nil {
		msg := "Unarchiving failed"
		f.analyticsEvent(update.AvailableUpdate.Version, msg)
		return errs.Wrap(err, msg)
	}

	return nil
}

func (f *Fetcher) analyticsEvent(version, msg string) {
	f.an.EventWithLabel(anaConst.CatUpdates, anaConst.ActUpdateDownload, anaConst.UpdateLabelFailed, &dimensions.Values{
		TargetVersion: ptr.To(version),
		Error:         ptr.To(msg),
	})
}

func verifySha(b []byte, sha string) error {
	h := sha256.New()
	h.Write(b)

	computed := h.Sum(nil)
	computedSha := fmt.Sprintf("%x", computed)
	bytesEqual := computedSha == sha
	if !bytesEqual {
		return errs.New("sha256 did not match")
	}

	return nil
}
