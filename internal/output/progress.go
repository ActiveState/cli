package output

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const moveCaretBack = "\x1b[%dD" // %d is the number of characters to move back

type Spinner struct {
	frame    int
	frames   []string
	out      Outputer
	stop     chan struct{}
	interval time.Duration
}

var _ Marshaller = &Spinner{}

func (d *Spinner) MarshalOutput(f Format) interface{} {
	return d.frames[d.frame]
}

func StartSpinner(out Outputer, msg string, interval time.Duration) *Spinner {
	frames := []string{".", "..", "..."}
	if isInteractive(out) {
		frames = []string{`|`, `/`, `-`, `\`}
	}
	d := &Spinner{0, frames, out, make(chan struct{}, 1), interval}

	if msg != "" {
		d.out.Fprint(d.out.Config().ErrWriter, strings.TrimSuffix(msg, " ")+" ")
	}
	if out.Type() == PlainFormatName { // Nothing will be printer otherwise
		go func() {
			d.ticker()
		}()
	}

	return d
}

func (d *Spinner) moveCaretBack() int {
	if !isInteractive(d.out) {
		return 0
	}
	prevPos := d.frame - 1
	if prevPos < 0 {
		prevPos = len(d.frames) - 1
	}
	prevFrame := d.frames[prevPos]
	d.out.Fprint(d.out.Config().ErrWriter, fmt.Sprintf(moveCaretBack, len(prevFrame)))

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

	if msg != "" {
		if !d.out.Config().Interactive {
			d.out.Fprint(d.out.Config().ErrWriter, " ")
		}
		d.out.Fprint(d.out.Config().ErrWriter, strings.TrimPrefix(msg, " "))
	}

	d.out.Fprint(d.out.Config().ErrWriter, "\n")
}

func isInteractive(out Outputer) bool {
	return out.Config().Interactive && terminal.IsTerminal(int(os.Stdin.Fd())) && (runtime.GOOS != "windows" || os.Getenv("SHELL") != "")
}
