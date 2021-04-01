package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/go-openapi/strfmt"
)

// Demo for the runbits.RuntimeMessageHandler
// It demonstrates the installation (and optionally the remote build) of five artificial packages
// This script accepts two arguments
// <withBuildEvents> <failedSteps>
//
// Examples:
// 1. All five packages install successfully (no build step)
//    go run -race ./scripts/progress-demo/demo
// 2. All five packages install successfully (with build step)
//    go run -race ./scripts/progress-demo/demo true
// 3. The fourth package fails during the build step
//    go run -race ./scripts/progress-demo/demo true nnnbn
// 4. The fourth package fails during the build step, and the second during the unpacking step, and the first one during the install step
//    go run -race ./scripts/progress-demo/demo true iunbn

func wait(times ...int) {
	var factor time.Duration = 2
	if len(times) > 0 {
		factor = time.Duration(times[0])
	}
	time.Sleep(factor * 200 * time.Millisecond)
}

func main() {
	err := run()
	if err != nil {
		fmt.Println(err.Error())
	}
}

type mockProducer struct {
	IDs      []strfmt.UUID
	Names    []string
	Prod     *events.RuntimeEventProducer
	Failures string
}

func (mp *mockProducer) NumArtifacts() int {
	return len(mp.IDs)
}

func newMockProducer(prod *events.RuntimeEventProducer, failures string) *mockProducer {
	return &mockProducer{
		IDs:      []strfmt.UUID{"1", "2", "3", "4", "5"},
		Names:    []string{"pkg 1", "pkg 2", "pkg 3", "pkg 4", "a very long pkg name (5)"},
		Prod:     prod,
		Failures: failures,
	}
}

func (mp *mockProducer) mockStepProgress(index int, step events.SetupStep) bool {
	mp.Prod.ArtifactStepStarting(step, mp.IDs[index], mp.Names[index], 100)
	wait()
	for i := 0; i < 10; i++ {
		mp.Prod.ArtifactStepProgress(step, mp.IDs[index], 10)
		wait()
	}
	if strings.ToLower(step.String())[0] == mp.Failures[index] {
		mp.Prod.ArtifactStepFailed(step, mp.IDs[index], "error")
		return false
	}
	mp.Prod.ArtifactStepCompleted(step, mp.IDs[index])
	return true
}

func (mp *mockProducer) mockArtifactProgress(withBuild bool, index int) bool {
	steps := []events.SetupStep{events.Download, events.Unpack, events.Install}
	if withBuild {
		steps = append([]events.SetupStep{events.Build}, steps...)
	}
	for _, s := range steps {
		if !mp.mockStepProgress(index, s) {
			return false
		}
	}
	return true
}

func run() error {
	withBuildEvents := false
	if len(os.Args) > 1 {
		withBuildEvents = (os.Args[1] == "true")
	}
	failedSteps := "nnnnn"
	if len(os.Args) > 2 {
		failedSteps = os.Args[2]
		if len(failedSteps) != 5 {
			return fmt.Errorf("failure string needs to have length 5")
		}
	}

	prod := events.NewRuntimeEventProducer()
	handler := runbits.DefaultRuntimeEventHandler(outputhelper.NewCatcher())

	mock := newMockProducer(prod, failedSteps)

	go func() {
		defer prod.Close()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			prod.TotalArtifacts(mock.NumArtifacts())
			if withBuildEvents {
				prod.BuildStarting(mock.NumArtifacts())
			}
			wait()
			for i := 0; i < mock.NumArtifacts(); i++ {
				wg.Add(1)
				go func(withBuildEvents bool, index int) {
					defer wg.Done()
					mock.mockArtifactProgress(withBuildEvents, index)
				}(withBuildEvents, i)
				wait(8)
			}
		}()
		wg.Wait()
	}()

	handler.WaitForAllEvents(prod.Events())
	return nil
}
