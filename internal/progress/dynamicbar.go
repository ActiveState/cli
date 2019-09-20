package progress

import (
	"github.com/vbauerster/mpb"
)

// DynamicBar is a helper struct for a progress bar with unknown total
type DynamicBar struct {
	bar          *mpb.Bar
	initialGuess int
	total        int
	offset       int
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// IncrBy increments the current count for the bar by `fileSize`
func (db *DynamicBar) IncrBy(fileSize int) {
	db.total += fileSize
	db.bar.SetTotal(int64(max(db.initialGuess, db.total+db.offset)), false)
	db.bar.IncrBy(int(fileSize))
}

// Complete needs to be called after the underlying task is finished
func (db *DynamicBar) Complete() {
	db.bar.SetTotal(int64(db.total), true)
	if !db.bar.Completed() {
		db.bar.IncrBy(int(db.bar.Total()))
	}
}
