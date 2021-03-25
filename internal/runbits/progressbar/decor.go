package progressbar

import (
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
)

// trimName ensures that the name in a progress bar is not too wide for a terminal to display
func (pb *RuntimeProgress) trimName(name string) string {
	if len(name) > pb.maxWidth {
		return name[0:pb.maxWidth]
	}
	return name
}

// addTotalBar adds a bar counting a number of discrete events
func (pb *RuntimeProgress) addTotalBar(name string, total int64) *mpb.Bar {
	name = pb.trimName(name)
	options := []mpb.BarOption{
		mpb.BarFillerClearOnComplete(),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: pb.maxWidth, C: decor.DidentRight}),
			decor.CountersNoUnit("%d / %d", decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	}

	return pb.prg.AddBar(total, options...)
}

// addByteBar adds a bar counting a number of bytes that have been processed eg., for a file download
func (pb *RuntimeProgress) addByteBar(name string, total int64, options ...mpb.BarOption) *mpb.Bar {
	name = pb.trimName(name)
	options = append([]mpb.BarOption{
		mpb.BarRemoveOnComplete(),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: pb.maxWidth, C: decor.DidentRight}),
			decor.Counters(decor.UnitKB, "%.1f / %.1f", decor.WCSyncSpace),
		),
		mpb.AppendDecorators(decor.Percentage(decor.WC{W: 5})),
	}, options...)

	return pb.prg.AddBar(total, options...)
}
