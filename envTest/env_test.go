package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/termtest"
)

func TestEnv(t *testing.T) {
	t.Skip("For debugging")
	ts := e2e.New(t, false)
	defer ts.Close()

	cp := ts.SpawnCmd("bash")
	cp.ExpectInput()

	cp.SendLine("/Users/mikedrakos/work/cli/build/envTest")
	cp.Expect("not exist", termtest.OptExpectTimeout(5*time.Second))
	fmt.Println(cp.Snapshot())
}
