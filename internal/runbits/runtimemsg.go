package runbits

// Progress bar design
//
import (
	"context"
	"time"

	"github.com/vbauerster/mpb/v6"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

type RuntimeMessageHandler struct {
	out output.Outputer
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out}
}

func (rmh *RuntimeMessageHandler) HandleUpdateEvents(eventCh <-chan events.BaseEventer, shutdownCh chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	prg := mpb.NewWithContext(ctx, mpb.WithShutdownNotifier(shutdownCh))

	pb := newProgressBar(prg)

	eh := events.NewRuntimeEventConsumer(pb)
	go func() {
		defer close(shutdownCh)
		defer cancel()
		eh.Run(eventCh)

		// Note: all of the following can be removed if we do our own progress bar implementation:
		// It is currently necessary as the mpb.Progress accepts requests from multiple threads, and therefore needs to be waited for to shutdown correctly.
		// But we do not need that functionality as we run all requests from the the same go routine in the eventHandle.run() call

		// wait at most half a second for the mpb.Progress instance to finish up its processing
		select {
		case <-time.After(time.Millisecond * 500):
		case <-shutdownCh:
		}
	}()
}
