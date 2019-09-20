// Package progress includes helper functions to report progress for a task
package progress

import (
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
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
	cancel   chan struct{} // triggered on Close to ensure that the progress bar unblocks
	totalBar *mpb.Bar      // a bar at the top that can report the current total progress
}

// New creates a new Progress struct
// mpb.BarOptions are forwarded
func New(options ...mpb.ProgressOption) *Progress {
	cancel := make(chan struct{})
	options = append(options, mpb.WithCancel(cancel))
	return &Progress{
		progress: mpb.New(options...),
		cancel:   cancel,
	}
}

// Close needs to be called after the Progress struct is not needed anymore
func (p *Progress) Close() {
	close(p.cancel)
	p.progress.Wait()
}

// GetNewTotalbar returns the top bar, that is supposed to report the total progress (of the current sub-task)
// The `name` is prepended, and for the last total bar, the `remove` flag should be set to `false` otherwise
// always `true`.
func (p *Progress) GetNewTotalbar(name string, numElements int, remove bool) *mpb.Bar {
	options := []mpb.BarOption{
		mpb.PrependDecorators(
			decor.StaticName(name, 20, 0),
			// decor.CountersNoUnit("%d / %d", 20, decor.DwidthSync),
		),
		mpb.AppendDecorators(
			decor.Percentage(5, 0),
		),
	}

	if p.totalBar != nil {
		options = append(options, mpb.BarReplaceOnComplete(p.totalBar))
	}

	if remove {
		options = append(options, mpb.BarRemoveOnComplete())
	}

	p.totalBar = p.progress.AddBar(int64(numElements), options...)
	return p.totalBar
}

// AddByteProgressBar adds a progressbar counting the progress in bytes
func (p *Progress) AddByteProgressBar(totalBytes int64) *mpb.Bar {
	return p.progress.AddBar(totalBytes,
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			decor.CountersKibiByte("%6.1f / %6.1f", 20, 0),
		),
		mpb.AppendDecorators(decor.Percentage(5, 0)))
}

// AddDynamicByteProgressbar adds a progressbar with unknown total
// `initialGuess` is the initial guess of a total
// `offset` is the offset in bytes by which the total will be updated any time
// we reach the old total, but were not done yet.
func (p *Progress) AddDynamicByteProgressbar(initialGuess, offset int) *DynamicBar {
	return &DynamicBar{bar: p.progress.AddBar(int64(initialGuess),
		mpb.BarRemoveOnComplete(),
		mpb.BarDynamicTotal(),
		mpb.BarAutoIncrTotal(18, int64(offset)),
		mpb.PrependDecorators(
			decor.CountersKibiByte("%6.1f / %6.1f", 20, 0),
		),
		mpb.AppendDecorators(
			decor.Percentage(5, 0),
		)),
		initialGuess: initialGuess,
		total:        0,
		offset:       offset,
	}

}
