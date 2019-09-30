package progress

import (
	"io"
	"time"

	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

// UnpackBar wraps a regular progress bar that is specifically for unpacking
// Note this peculiarities about unpacking:
// - the number of bytes to be read is known.
// - the number of bytes to be written is unknown.
// This struct therefore stores an artificial total (2% about the total bytes to be read)
// Run Complete when you have written all files to disc, and the bar will fill the remaining 2%.
type UnpackBar struct {
	bar   *mpb.Bar
	total int64
}

// NewUnpackBar creates a progressbar for unpacking an archiving.
func NewUnpackBar(bytesToRead int64, p *Progress) *UnpackBar {
	// add a 2% buffer for finishing the last writes
	total := int64(float64(bytesToRead) * 1.02)
	return &UnpackBar{bar: p.progress.AddBar(total,
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			// synchronize the width with the two total bar decorations
			decor.Merge(
				decor.OnComplete(decor.Spinner(nil, decor.WCSyncSpace), ""),
				decor.WCSyncSpace),
			// decor.Counters(decor.UnitKiB, "%6.1f / %6.1f", decor.WC{W: 20, C: decor.DidentRight}),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
		)),
		total: total,
	}
}

// Complete completes the progress to 100% and should be called after all files are written to disc
func (upb *UnpackBar) Complete() {
	upb.bar.SetCurrent(upb.total)
}

// NewProxyReader wraps a Reader with functionality that automatically updates
// the bar with progress about how many bytes have been read from the underlying
// reader so far.
func (upb *UnpackBar) NewProxyReader(r io.ReadCloser) *proxyReader {
	return &proxyReader{ReadCloser: r, bar: upb.bar, iT: time.Now()}
}
