package spinner

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
)

type nonInteractive struct {
	isGroup        bool
	supportsColors bool
	stop           chan struct{}
}

func newNonInteractive(isGroup, supportColors bool) *nonInteractive {
	n := &nonInteractive{isGroup: isGroup, supportsColors: supportColors, stop: make(chan struct{}, 1)}
	go n.ticker()
	return n
}

func (n *nonInteractive) ticker() {
	ticker := time.NewTicker(constants.TerminalAnimationInterval)
	for {
		select {
		case <-ticker.C:
			os.Stderr.WriteString(".")
		case <-n.stop:
			return
		}
	}
}

func (n *nonInteractive) Add(prefix string) Spinnable {
	os.Stderr.WriteString("\n" + color(prefix, !n.supportsColors) + " ")
	return n
}

func (n *nonInteractive) Wait() {
	n.stop <- struct{}{}
}

func (n *nonInteractive) Stop(msg string) {
	os.Stderr.WriteString(color(msg, !n.supportsColors) + "\n")
	if !n.isGroup {
		// If this isn't a group Wait will never be called
		n.stop <- struct{}{}
	}
}
