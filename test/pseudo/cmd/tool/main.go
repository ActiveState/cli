package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/test/pseudo/cmd/internal/serve"
	intsvcctl "github.com/ActiveState/cli/test/pseudo/cmd/internal/svcctl"
)

func main() {
	ns := intsvcctl.NewIPCNamespace()
	ipcClient := ipc.NewClient(ns)

	addr, err := svcctl.EnsureStartedAndLocateHTTP(ipcClient, "../svc/build/svc")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ensure and locate: %v\n", errs.JoinMessage(err))
		os.Exit(1)
	}

	httpClient := serve.NewClient(addr)
	data, err := httpClient.GetInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get info: %v\n", errs.JoinMessage(err))
		os.Exit(1)
	}

	fmt.Print(data)

	//time.Sleep(time.Second)
	fmt.Println(svcctl.StopServer(ipcClient))
}
