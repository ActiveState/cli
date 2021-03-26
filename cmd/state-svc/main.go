package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type command string

const (
	CmdService    command = "service"
	CmdForeground         = "background"
	CmdUninstall          = "uninstall"
)

var commands = []command{
	CmdService,
	CmdForeground,
	CmdUninstall,
}

var elevatedCommands = []command{
	CmdService,
	CmdUninstall,
}

func main() {
	err := run()
	if err != nil {
		errMsg := errs.Join(err, ": ").Error()
		logging.Errorf("state-svc errored out: %s", errMsg)
		fmt.Fprintln(os.Stderr, errMsg)
		os.Exit(1)
	}
}

func run() error {
	cmd := command(os.Args[1])

	if !amAdmin() {
		for _, ec := range elevatedCommands {
			if ec == cmd {
				fmt.Println("Re-running as admin")
				return rerunElevated()
			}
		}
	}

	switch cmd {
	case CmdService:
		logging.Debug("Running CmdService")
		return runService()
	case CmdForeground:
		logging.Debug("Running CmdForeground")
		return runForeground()
	case CmdUninstall:
		logging.Debug("Running CmdUninstall")
		return runUninstall()
	}

	return errs.New("Missing command, expecting one of: %v", commands)
}

func runForeground() error {
	logging.Debug("Running standalone")

	p := NewProgram()
	if err := p.Start(); err != nil {
		return errs.Wrap(err, "Could not start program")
	}

	return nil
}

func runService() error {
	logging.Debug("Running service")

	p := NewProgram()
	svcHandler := NewServiceHandler(p)
	svc, err := NewService(svcHandler)
	if err != nil {
		return errs.Wrap(err, "Could not construct service")
	}

	isInstalled, err := svc.IsInstalled()
	if err != nil {
		return errs.Wrap(err, "Could not detect if installed")
	}
	if !isInstalled {
		logging.Debug("Installing service")
		if err := svc.Install(); err != nil {
			return errs.Wrap(err, "Installation failed")
		}
	}

	logging.Debug("Starting service")
	if err := svc.Start(); err != nil {
		return errs.Wrap(err, "Could not start service")
	}

	return nil
}

func runUninstall() error {
	logging.Debug("Running uninstall")

	p := NewProgram()
	svcHandler := NewServiceHandler(p)
	svc, err := NewService(svcHandler)
	if err != nil {
		return errs.Wrap(err, "Could not construct service")
	}

	if err := svc.Uninstall(); err != nil {
		return errs.Wrap(err, "Uninstall failed")
	}

	return nil
}

func rerunElevated() error {
	logging.Debug("Rerun as admin")

	verb := "runas"
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	args := strings.Join(os.Args[1:], " ")

	verbPtr, err := syscall.UTF16PtrFromString(verb)
	if err != nil {
		return err
	}
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	cwdPtr, err := syscall.UTF16PtrFromString(cwd)
	if err != nil {
		return err
	}
	argPtr, err := syscall.UTF16PtrFromString(args)
	if err != nil {
		return err
	}

	var showCmd int32 = 0 // SW_NORMAL

	return windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
}

func amAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}