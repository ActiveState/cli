package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ActiveState/cli/exp/pm/cmd/internal/serve"
	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/exp/pm/internal/svcctl"
)

func main() {
	var (
		rootDir = "/tmp/svccomm"
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
	)

	flag.StringVar(&version, "v", version, "version id")
	flag.Parse()

	n := &ipc.Namespace{
		RootDir:    rootDir,
		AppName:    name,
		AppVersion: version,
		AppHash:    hash,
	}
	addr, err := svcctl.EnsureAndLocateHTTP(n)
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
}
