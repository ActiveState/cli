// Package progress includes helper functions to report progress for a task
// The idea is that you always start with a TotalBar (`p.AddTotalBar`) counting eg.,
// the number of downloads or installations.
// For each actual task you can add a separate progress bar once it is running
// Currently, the following task based progrss bars are supported:
// - a progress bar usually used for downloads, counting the number of bytes processed
// - a special progress bar used for unpacking an archive, where only the number of bytes to be read are known.
package progress

import (
	"context"
	"io"

	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

// FileSizeCallback can be called by a task to report that a sub-task of length `fileSize` (in bytes) has been finished
type FileSizeCallback func(fileSize int)

// FileSizeTask is a function for a task that reports its progress in bytes processed
type FileSizeTask func(FileSizeCallback) error

// Progress is a small wrapper around the mpb.Progress struct
// Motivation: The multi-progress bars are used in several places, and can override each other.
// So all code that generates and manipulates progress bars is organized under this struct
// This simplifies testing and demo-ing of new progress bar functionality.
type Progress struct {
	progress *mpb.Progress
	cancel   context.CancelFunc // triggered on Close to ensure that the progress bar unblocks
	totalBar *mpb.Bar           // a bar at the top that can report the current total progress
}

// WithOutput changes the output of the progress bar
// This is a wrapper around `mpb.WithOutput`
// Provide `nil` if output should be discarded
func WithOutput(w io.Writer) mpb.ContainerOption {
	return mpb.WithOutput(w)
}

// New creates a new Progress struct
// mpb.ContainerOptions are forwarded
func New(options ...mpb.ContainerOption) *Progress {
	ctx, cancel := context.WithCancel(context.Background())
	return &Progress{
		progress: mpb.NewWithContext(ctx, options...),
		cancel:   cancel,
	}
}

// Close needs to be called after the Progress struct is not needed anymore
func (p *Progress) Close() {
	p.cancel()
	p.progress.Wait()
}

// TotalBar is an alias of mpb.Bar currently
type TotalBar = mpb.Bar

// ByteProgressBar is an alias of mpb.Bar currently
type ByteProgressBar = mpb.Bar

// AddTotalBar returns the top bar, that is supposed to report the total progress (of the current sub-task)
// The `name` is prepended, and for the last total bar, the `remove` flag should be set to `false` otherwise
// always `true`.
func (p *Progress) AddTotalBar(name string, numElements int) *TotalBar {
	options := []mpb.BarOption{
		mpb.BarClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(name, decor.WCSyncSpaceR),
			decor.CountersNoUnit("%d / %d", decor.WCSyncSpace),
			// decor.CountersNoUnit("%d / %d", 20, decor.DwidthSync),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	}

	return p.progress.AddBar(int64(numElements), options...)
}

// AddByteProgressBar adds a progressbar counting the progress in bytes
func (p *Progress) AddByteProgressBar(totalBytes int64) *ByteProgressBar {
	return p.progress.AddBar(totalBytes,
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			// synchronize the width with the two total bar decorations
			decor.Merge(
				decor.Counters(decor.UnitKiB, "%.1f / %.1f", decor.WCSyncSpace),
				decor.WCSyncSpace,
			),
		),
		mpb.AppendDecorators(decor.Percentage(decor.WC{W: 5})))
}

// AddUnpackBar adds a progressbar for unpacking an archiving.
func (p *Progress) AddUnpackBar(bytesToRead int64) *UnpackBar {
	return NewUnpackBar(bytesToRead, p)

}
