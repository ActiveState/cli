package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcctl"
)

func main() {
	for i := 0; i < 1000; i++ {
		if i%20 == 0 {
			fmt.Println("iter:", i)
		}
		spath := svcctl.NewIPCSockPathFromGlobals()
		ipcClient := ipc.NewClient(spath)

		addr, err := svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, "../../../../build/state-svc")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ensure and locate: %v\n", errs.JoinMessage(err))
			os.Exit(1)
		}

		if i == 0 {
			fmt.Println(addr)
		}

		if err := svcctl.StopServer(ipcClient); err != nil {
			fmt.Print("iter: ", i, " ")
			fmt.Println(errs.JoinMessage(err))
		}
	}
}
