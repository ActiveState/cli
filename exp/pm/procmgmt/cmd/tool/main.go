package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/proccomm"
	"github.com/ActiveState/cli/exp/pm/internal/socket"
	"github.com/ActiveState/cli/exp/pm/procmgmt/internal/serve"
	"github.com/ActiveState/cli/internal/exeutils"
)

func main() {
	start := time.Now()

	var (
		rootDir = "/tmp/proccomm"
		name    = "state"
		version = "default"
		hash    = "DEADBEEF"
	)

	flag.StringVar(&version, "v", version, "version id")
	flag.Parse()
	fmt.Println("parsed flags", time.Since(start))

	n := &socket.Namespace{
		RootDir:    rootDir,
		AppName:    name,
		AppVersion: version,
		AppHash:    hash,
	}
	sc := socket.NewClient(n)
	pc := proccomm.NewClient(sc)
	fmt.Println("setup proccomm client", time.Since(start))
	addr, err := pc.GetHTTPAddr()
	fmt.Println("got http addr", time.Since(start))

	if err != nil {
		args := []string{"-v", version}

		if _, err = exeutils.ExecuteAndForget("../svc/build/svc", args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("starting service")
		time.Sleep(time.Second)

		addr, err = pc.GetHTTPAddr()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	c := serve.NewClient(addr)
	info, err := c.GetInfo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("got info via http", time.Since(start))
	fmt.Print(info)
}
