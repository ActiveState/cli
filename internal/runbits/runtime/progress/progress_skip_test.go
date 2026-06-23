package progress

import (
	"io"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
)

// A skipped artifact fires only ArtifactInstallSkipped (no Started/Success), yet
// it is counted in the install bar's total. The bar must still reach its total,
// otherwise Close() stalls waiting for it and reports a spurious error.
func TestInstallSkippedCompletesBar(t *testing.T) {
	installed := strfmt.UUID("11111111-1111-1111-1111-111111111111")
	skipped := strfmt.UUID("22222222-2222-2222-2222-222222222222")

	t.Run("alongside an installed artifact", func(t *testing.T) {
		p := newProgressIndicator(io.Discard, outputhelper.NewCatcher())
		defer p.cancelMpb()

		expected := buildplan.ArtifactIDMap{installed: nil, skipped: nil}
		handle(t, p, events.Start{ArtifactsToInstall: expected})
		handle(t, p, events.ArtifactInstallStarted{installed})
		handle(t, p, events.ArtifactInstallSuccess{installed})
		handle(t, p, events.ArtifactInstallSkipped{skipped, "pkg"})

		if got, want := p.installBar.Current(), int64(len(expected)); got != want {
			t.Errorf("install bar at %d/%d; want %d (a skipped artifact must still advance the bar)", got, p.installBar.total, want)
		}
	})

	t.Run("when every artifact is skipped", func(t *testing.T) {
		p := newProgressIndicator(io.Discard, outputhelper.NewCatcher())
		defer p.cancelMpb()

		// No Started event fires, so the skip event itself must create the bar.
		expected := buildplan.ArtifactIDMap{skipped: nil}
		handle(t, p, events.Start{ArtifactsToInstall: expected})
		handle(t, p, events.ArtifactInstallSkipped{skipped, "pkg"})

		if p.installBar == nil {
			t.Fatal("install bar was never created for an all-skipped install")
		}
		if got, want := p.installBar.Current(), int64(len(expected)); got != want {
			t.Errorf("install bar at %d/%d; want %d", got, p.installBar.total, want)
		}
	})
}

func handle(t *testing.T, p *ProgressDigester, ev events.Event) {
	t.Helper()
	if err := p.Handle(ev); err != nil {
		t.Fatalf("Handle(%T): %v", ev, err)
	}
}
