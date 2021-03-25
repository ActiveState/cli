package runbits

// Progress bar design
//
import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/thoas/go-funk"
	"github.com/vbauerster/mpb/v6"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/progressbar"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
)

type RuntimeMessageHandler struct {
	out output.Outputer
}

func NewRuntimeMessageHandler(out output.Outputer) *RuntimeMessageHandler {
	return &RuntimeMessageHandler{out}
}

func (rmh *RuntimeMessageHandler) ChangeSummary(artifacts artifact.ArtifactRecipeMap, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) error {
	// currently we only print a change summary if we are adding exactly ONE package
	if len(requested.Added) != 1 {
		return nil
	}

	ar, ok := artifacts[requested.Added[0]]
	if !ok {
		return errs.New("Did not find requested artifact in ArtifactRecipeMap")
	}

	// the added (direct dependencies) of this artifact that are actually new in this project
	addedDependencies := funk.Join(ar.Dependencies, changed.Added, funk.InnerJoin).([]artifact.ArtifactID)

	rmh.out.Notice("")
	rmh.out.Notice(locale.Tl(
		"changesummary_title",
		"[NOTICE]{{.V0}}[/RESET] includes {{.V1}} dependencies, for a combined total of {{.V2}} new dependencies.",
		ar.Name, strconv.Itoa(len(addedDependencies)), strconv.Itoa(len(changed.Added)),
	))
	for i, dep := range addedDependencies {
		depMapping, ok := artifacts[dep]
		if !ok {
			logging.Error("Could not find dependency %s in artifactsMap", dep)
			continue
		}
		var depCount string
		recDeps := artifact.RecursiveDependenciesFor(dep, artifacts)
		filteredRecDeps := funk.Join(recDeps, changed.Added, funk.InnerJoin).([]artifact.ArtifactID)
		if len(filteredRecDeps) > 0 {
			depCount = locale.Tl("ingredient_dependency_count", " ({{.V0}} dependencies)", strconv.Itoa(len(filteredRecDeps)))
		}
		prefix := "├─"
		if i == len(addedDependencies)-1 {
			prefix = "└─"
		}
		rmh.out.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s%s", prefix, depMapping.Name, depCount))
	}
	return nil
}

// HandleUpdateEvents prints output based on runtime events received on the eventCh
func (rmh *RuntimeMessageHandler) HandleUpdateEvents(eventCh <-chan events.BaseEventer) {
	ctx, cancel := context.WithCancel(context.Background())
	prgShutdownCh := make(chan struct{})
	options := []mpb.ContainerOption{mpb.WithShutdownNotifier(prgShutdownCh), mpb.WithWidth(40)}
	if rmh.out.Type() != output.PlainFormatName {
		options = append(options, mpb.WithOutput(nil))
	}
	prg := mpb.NewWithContext(ctx, options...)

	defer prg.Wait() // Note: This closes the prgShutdownCh
	defer cancel()

	pb := progressbar.New(prg)

	eh := events.NewRuntimeEventConsumer(pb, rmh)
	eh.Run(eventCh)

	// Note: all of the following can be removed if we do our own progress bar implementation:
	// It is currently necessary as the mpb.Progress accepts requests from multiple threads, and therefore needs to be waited for to shutdown correctly.
	// But we do not need that functionality as we run all requests from the the same go routine in the eventHandle.run() call

	// wait at most half a second for the mpb.Progress instance to finish up its processing
	select {
	case <-time.After(time.Millisecond * 500):
	case <-prgShutdownCh:
	}
}
