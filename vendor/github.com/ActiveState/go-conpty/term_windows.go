// +build windows

package conpty

import (
	"fmt"
	"log"
	"syscall"

	"github.com/Azure/go-ansiterm/winterm"
)

func InitTerminal() (func(), error) {
	stdinFd := int(syscall.Stdin)
	stdoutFd := int(syscall.Stdout)

	fmt.Printf("file descriptors <%d >%d\n", stdinFd, stdoutFd)

	oldInMode, err := winterm.GetConsoleMode(uintptr(stdinFd))
	if err != nil {
		return func() {}, fmt.Errorf("failed to retrieve stdin mode: %w", err)
	}

	oldOutMode, err := winterm.GetConsoleMode(uintptr(stdoutFd))
	if err != nil {
		return func() {}, fmt.Errorf("failed to retrieve stdout mode: %w", err)
	}

	fmt.Printf("old modes: <%d >%d\n", oldInMode, oldOutMode)

	newInMode := oldInMode                                                // | winterm.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	newOutMode := oldOutMode | winterm.ENABLE_VIRTUAL_TERMINAL_PROCESSING // | winterm.DISABLE_NEWLINE_AUTO_RETURN

	err = winterm.SetConsoleMode(uintptr(stdinFd), newInMode)
	if err != nil {
		return func() {}, fmt.Errorf("failed to set stdin mode: %w", err)
	}

	dump(uintptr(stdinFd))

	err = winterm.SetConsoleMode(uintptr(stdoutFd), newOutMode)
	if err != nil {
		return func() {}, fmt.Errorf("failed to set stdout mode: %w", err)
	}

	dump(uintptr(stdoutFd))

	return func() {
		err = winterm.SetConsoleMode(uintptr(stdinFd), oldInMode)
		if err != nil {
			log.Fatalf("Failed to reset input terminal mode to %d: %v\n", oldInMode, err)
		}

		err = winterm.SetConsoleMode(uintptr(stdoutFd), oldOutMode)
		if err != nil {
			log.Fatalf("Failed to reset output terminal mode to %d: %v\n", oldOutMode, err)
		}
	}, nil
}

func dump(fd uintptr) {
	fmt.Printf("FD=%d\n", fd)
	modes, err := winterm.GetConsoleMode(fd)
	if err != nil {
		panic(err)
	}

	fmt.Printf("ENABLE_ECHO_INPUT=%d, ENABLE_PROCESSED_INPUT=%d ENABLE_LINE_INPUT=%d\n",
		modes&winterm.ENABLE_ECHO_INPUT,
		modes&winterm.ENABLE_PROCESSED_INPUT,
		modes&winterm.ENABLE_LINE_INPUT)
	fmt.Printf("ENABLE_WINDOW_INPUT=%d, ENABLE_MOUSE_INPUT=%d\n",
		modes&winterm.ENABLE_WINDOW_INPUT,
		modes&winterm.ENABLE_MOUSE_INPUT)
	fmt.Printf("enableVirtualTerminalInput=%d, enableVirtualTerminalProcessing=%d, disableNewlineAutoReturn=%d\n",
		modes&winterm.ENABLE_VIRTUAL_TERMINAL_INPUT,
		modes&winterm.ENABLE_VIRTUAL_TERMINAL_PROCESSING,
		modes&winterm.DISABLE_NEWLINE_AUTO_RETURN)
}
