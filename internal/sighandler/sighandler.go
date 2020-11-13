package sighandler

import (
	"os"
	"os/signal"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/logging"
)

var sigCh chan os.Signal
var ignoreCh chan bool

// Init initializes the global signal interrupt handler
func Init(subCommand string) {
	sigCh = make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	ignoreCh = make(chan bool, 0)

	go func(subCommand string) {
		defer close(sigCh)
		defer close(ignoreCh)
		ignore := false
		for {
			select {
			case <-sigCh:
				if !ignore {
					logging.Debug("captured ctrl-c event")
					analytics.EventWithLabel(analytics.CatCommandExit, subCommand, "interrupt")
					analytics.WaitForAllEvents(time.Second)
					os.Exit(1)
				}
			case val := <-ignoreCh:
				ignore = val
			}
		}
	}(subCommand)
}

// IgnoreInterrupts triggers whether the State Tool process should exit on an interrupt event
// It should be set to true if interrupt events are handled with error values as
// it is done eg., when launching subprocesses.
func IgnoreInterrupts(ignore bool) {
	ignoreCh <- ignore
}
