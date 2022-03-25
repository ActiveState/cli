package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/test/pseudo/cmd/internal/serve"
)

func main() {
	ns := svcctl.NewIPCNamespaceFromGlobals()
	ipcClient := ipc.NewClient(ns)

	addr, err := svcctl.EnsureAndLocateHTTP(ipcClient)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	httpClient := serve.NewClient(addr)
	data, err := httpClient.GetInfo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(data)

	time.Sleep(time.Second)
	fmt.Println(svcctl.StopServer(ipcClient))
}
