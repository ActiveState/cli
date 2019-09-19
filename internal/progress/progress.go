package progress

import (
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

// FileSizeCallback can be called by a task to report that a sub-task of length `fileSize` (in bytes) has been finished
type FileSizeCallback func(fileSize int64)

// FileSizeTask is a function for a task that reports its progress in bytes processed
type FileSizeTask func(FileSizeCallback) error

// progress includes helper functions to report progress for a task

// ReportProgressDynamically adds a progress for a task with an unknown length.
//
// An initial guess for the total size can be specified.
//
// Note: Currently the task is supposed to report a file size in bytes.
func ReportProgressDynamically(taskFunc FileSizeTask, progress *mpb.Progress, initialGuess int64) error {

	var total int64
	var bar *mpb.Bar
	if progress != nil {
		bar = progress.AddBar(initialGuess,
			mpb.BarRemoveOnComplete(),
			mpb.BarDynamicTotal(),
			mpb.BarAutoIncrTotal(18, 2048),
			mpb.PrependDecorators(
				decor.CountersKibiByte("%6.1f / %6.1f", 20, 0),
			),
			mpb.AppendDecorators(
				decor.Percentage(5, 0),
			))
	}

	max := func(x, y int64) int64 {
		if x < y {
			return y
		}
		return x
	}

	updateCallback := func(fileSize int64) {
		total += fileSize
		if bar != nil {
			bar.SetTotal(max(100*1024, total+2048), false)
			bar.IncrBy(int(fileSize))
		}
	}

	err := taskFunc(updateCallback)

	if bar != nil {
		// after the archiving is finished, update the total
		bar.SetTotal(total, true)

		// Failsafe, so we do not get blocked by a progressbar
		if !bar.Completed() {
			bar.IncrBy(int(bar.Total()))
		}
	}
	return err
}
