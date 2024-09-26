package output

import (
	"fmt"
	"strings"
	"time"
)

const moveCaretBackEscapeSequence = "\x1b[%dD" // %d is the number of characters to move back

var SpinnerFrames = []string{`|`, `/`, `-`, `\`}

type Spinner struct {
	frame         int
	frames        []string
	out           Outputer
	stop          chan struct{}
	interval      time.Duration
	reportedError bool
}

var _ Marshaller = &Spinner{}

func (d *Spinner) MarshalOutput(f Format) interface{} {
	return d.frames[d.frame]
}

func StartSpinner(out Outputer, msg string, interval time.Duration) *Spinner {
	frames := []string{"."}
	if out.Config().Interactive {
		frames = SpinnerFrames
	}
	d := &Spinner{0, frames, out, make(chan struct{}, 1), interval, false}

	if msg != "" && d.out.Type() == PlainFormatName {
		d.out.Fprint(d.out.Config().ErrWriter, strings.TrimSuffix(msg, " ")+" ")
	}
	if d.out.Type() == PlainFormatName { // Nothing will be printer otherwise
		go func() {
			d.ticker()
		}()
	}

	return d
}

func (d *Spinner) moveCaretBack() int {
	if !d.out.Config().Interactive {
		return 0
	}
	prevPos := d.frame - 1
	if prevPos < 0 {
		prevPos = len(d.frames) - 1
	}
	prevFrame := d.frames[prevPos]
	if d.out.Config().ShellName != "cmd" { // cannot use subshell/cmd.Name due to import cycle
		d.moveCaretBackInTerminal(len(prevFrame))
	} else {
		d.moveCaretBackInCommandPrompt(len(prevFrame))
	}

	return len(prevFrame)
}

func (d *Spinner) tick(isFirstFrame bool) {
	clear := ""
	if !isFirstFrame {
		nMoved := d.moveCaretBack()
		if nMoved > 0 {
			clear = strings.Repeat(" ", nMoved-1)
		}
	}
	d.out.Fprint(d.out.Config().ErrWriter, d.frames[d.frame]+clear)
	d.frame++
	if d.frame == len(d.frames) {
		d.frame = 0
	}
}

func (d *Spinner) ticker() {
	d.tick(true) // print the first frame immediately
	ticker := time.NewTicker(d.interval)
	for {
		select {
		case <-ticker.C:
			d.tick(false)
		case <-d.stop:
			return
		}
	}
}

func (d *Spinner) Stop(msg string) {
	d.stop <- struct{}{}
	close(d.stop)

	// We're done, so remove the last spinner frame
	if d.out.Config().Interactive {
		nMoved := d.moveCaretBack()
		if nMoved > len(msg) {
			msg += strings.Repeat(" ", nMoved-len(msg)-1)
		}
	}

	if msg != "" && d.out.Type() == PlainFormatName {
		if !d.out.Config().Interactive {
			d.out.Fprint(d.out.Config().ErrWriter, " ")
		}
		d.out.Fprint(d.out.Config().ErrWriter, strings.TrimPrefix(msg, " "))
	}

	if d.out.Type() == PlainFormatName {
		d.out.Fprint(d.out.Config().ErrWriter, "\n")
	}
}

func (d *Spinner) moveCaretBackInTerminal(n int) {
	d.out.Fprint(d.out.Config().ErrWriter, fmt.Sprintf(moveCaretBackEscapeSequence, n))
}
