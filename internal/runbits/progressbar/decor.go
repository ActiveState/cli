package progressbar

import (
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// trimName ensures that the name in a progress bar is not too wide for a terminal to display
func (pb *RuntimeProgress) trimName(name string) string {
	if len(name) > pb.maxWidth {
		return name[0:pb.maxWidth]
	}
	return name
}

// addTotalBar adds a bar counting a number of sub-events adding up to total
func (pb *RuntimeProgress) addTotalBar(name string, total int64) *mpb.Bar {
	return pb.addBar(name, total, false, mpb.BarFillerClearOnComplete())
}

// addSpinnerBar adds a bar with a spinning progress indicator
func (pb *RuntimeProgress) addSpinnerBar(name string) *mpb.Bar {
	return pb.prg.Add(0,
		mpb.NewBarFiller(mpb.SpinnerStyle([]string{`|`, `/`, `-`, `\`}...)),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: pb.maxWidth, C: decor.DidentRight}),
			decor.CountersNoUnit("", decor.WC{W: 0}),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.NewPercentage("", decor.WC{W: 5}), ""),
		),
	)
}

// addArtifactStepBar adds a bar counting the progress in a specific artifact setup step
func (pb *RuntimeProgress) addArtifactStepBar(name string, total int64, countsBytes bool) *mpb.Bar {
	return pb.addBar(name, total, countsBytes, mpb.BarRemoveOnComplete())
}

func (pb *RuntimeProgress) addBar(name string, total int64, countsBytes bool, options ...mpb.BarOption) *mpb.Bar {
	name = pb.trimName(name)
	decorators := []decor.Decorator{
		decor.Name(name, decor.WC{W: pb.maxWidth, C: decor.DidentRight}),
		decor.OnComplete(
			decor.Spinner(nil, decor.WCSyncSpace), "",
		),
	}
	if countsBytes {
		decorators = append(decorators, decor.CountersKiloByte("%.1f/%.1f", decor.WC{W: 17}))
	} else {
		decorators = append(decorators, decor.CountersNoUnit("%d/%d", decor.WC{W: 17}))
	}
	options = append(options,
		mpb.PrependDecorators(decorators...),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WC{W: 5}), ""),
		),
	)

	return pb.prg.AddBar(total, options...)
}
