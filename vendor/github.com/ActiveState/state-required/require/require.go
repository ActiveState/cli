package require

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	ps "github.com/mitchellh/go-ps"
)

func init() {
	exe, err := os.Executable()
	if err != nil {
		log.Println("Could not detect Executable, state-required will not be able to function")
		return
	}

	if !strings.Contains(exe, filepath.Join("_obj", "exe")) || flag.Lookup("test.v") != nil {
		return
	}

	if !hasStateParent(os.Getppid()) {
		log.Fatal("The go code you are trying to run requires that you use the ActiveState state tool, " +
			"for more information check out https://github.com/ActiveState/cli")
	}
}

func hasStateParent(pid int) bool {
	for true {
		p, err := ps.FindProcess(pid)
		if err != nil {
			panic("Could not detect process information: " + err.Error())
		}

		if p == nil {
			return false
		}

		if strings.HasPrefix(p.Executable(), "state") {
			return true
		}

		ppid := p.PPid()
		if p.PPid() == pid {
			break
		}

		pid = ppid
	}

	return false
}
