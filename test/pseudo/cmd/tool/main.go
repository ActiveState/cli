package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/test/pseudo/cmd/internal/serve"
)

func main() {
	var (
		rootDir = filepath.Join(os.TempDir(), "svccomm")
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

	time.Sleep(time.Second)
	fmt.Println(svcctl.StopServer(n))
}
