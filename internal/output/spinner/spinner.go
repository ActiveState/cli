package spinner

import (
	"bytes"
	"io"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type Groupable interface {
	Add(prefix string) Spinnable
	Wait()
}

type Spinnable interface {
	Stop(msg string)
}

// Group collects multiple spinners
type Group struct {
	mpbGroup      *mpb.Progress
	supportColors bool
	interactive   bool
}

// Spinner represents a single spinner
type Spinner struct {
	mpbBar        *mpb.Bar
	completionMsg string
}

// StandaloneSpinner represents a single spinner that is used in standalone, meaning it doesn't have a group.
type StandaloneSpinner struct {
	*Spinner
	mpbGroup *mpb.Progress
}

func NewGroup(supportColors, interactive bool) Groupable {
	if !interactive {
		return newNonInteractive(true, supportColors)
	}
	return &Group{mpb.New(
		mpb.WithWidth(1)),
		supportColors,
		interactive,
	}
}

func NewSpinner(prefix string, supportColors, interactive bool) Spinnable {
	if !interactive {
		n := newNonInteractive(false, supportColors)
		n.Add(prefix)
		return n
	}
	mpbGroup := mpb.New(mpb.WithWidth(1))
	return &StandaloneSpinner{newSpinner(mpbGroup, prefix, supportColors, interactive), mpbGroup}
}

func newSpinner(mpbGroup *mpb.Progress, prefix string, supportColors, interactive bool) *Spinner {
	s := &Spinner{}
	p := mpbGroup.Add(
		1,
		mpb.NewBarFiller(mpb.SpinnerStyle([]string{`|`, `/`, `-`, `\`}...)),
		mpb.PrependDecorators(decor.Any(func(s decor.Statistics) string {
			return color(prefix, !supportColors)
		})),
		barFillerOnComplete(func() string { return color(strings.TrimPrefix(s.completionMsg, " "), !supportColors) }),
	)
	s.mpbBar = p
	return s
}

func (g *Group) Add(prefix string) Spinnable {
	s := newSpinner(g.mpbGroup, prefix, g.supportColors, g.interactive)
	return s
}

func (s *Spinner) Stop(msg string) {
	s.completionMsg = msg
	s.mpbBar.Increment() // Our "bar" has a total of 1, so a single increment will complete it
}

func (s *StandaloneSpinner) Stop(msg string) {
	s.Spinner.Stop(msg)
	s.mpbGroup.Wait()
}

func (g *Group) Wait() {
	g.mpbGroup.Wait()
}

func color(v string, strip bool) string {
	if strip {
		return colorize.StripColorCodes(v)
	}

	b := &bytes.Buffer{}
	_, err := colorize.Colorize(v, b, false)
	if err != nil {
		logging.Warning("colorize failed, stripping colors - error: %s", errs.JoinMessage(err))
		v = colorize.StripColorCodes(v)
	} else {
		v = b.String()
	}

	return v
}

func barFillerOnComplete(value func() string) mpb.BarOption {
	return mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
		return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
			if st.Completed {
				io.WriteString(w, value())
			} else {
				base.Fill(w, reqWidth, st)
			}
		})
	})
}
