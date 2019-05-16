package shim

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/executor"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
)

var Command *commands.Command

func init() {
	Command = &commands.Command{
		Name:        "shim",
		Description: "shim_description",
		Run:         Execute,
		Exiter:      os.Exit,

		DisableFlagParsing: true,
	}
}

func Execute(cmd *cobra.Command, args []string) {
	exe := executor.New(args[0], args[1:]...)
	exe.OnStdin(func(input []byte) {
		logging.Debug("STDIN: %s", string(input))
	})
	exe.OnStdout(func(output []byte) {
		logging.Debug("STDOUT: %s", string(output))
	})
	exe.OnStderr(func(output []byte) {
		logging.Debug("STDERR: %s", string(output))
	})
	fail := exe.Run()
	if fail != nil {
		failures.Handle(fail, "Command failed")
	}
}

func old(args []string) {
	execCmd := exec.Command(args[0], args[1:]...)

	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	stdinWriter, err := execCmd.StdinPipe()
	if err != nil {
		failures.Handle(err, "Could not get stdin pipe")
		Command.Exiter(1)
	}

	err = execCmd.Start()
	if err != nil {
		failures.Handle(err, "Command started with error")
		Command.Exiter(1)
	}

	scanner(stdinWriter, execCmd)

	if err := execCmd.Wait(); err != nil {
		failures.Handle(err, "Command exited with erro")
		Command.Exiter(1)
	}

	fmt.Println("Done")
	Command.Exiter(osutils.CmdExitCode(execCmd))
}

func scanner(stdinWriter io.WriteCloser, execCmd *exec.Cmd) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanBytes)

	for scanner.Scan() {
		input := scanner.Bytes()

		logging.Debug("STDIN: %s", string(input))
		stdinWriter.Write(input)
	}

	if err := scanner.Err(); err != nil {
		failures.Handle(err, "Scanner failed with error")
		Command.Exiter(1)
	}
}

func reader(stdinWriter io.WriteCloser, execCmd *exec.Cmd) {
	rdr := bufio.NewReader(os.Stdin)

	for {
		input, err := rdr.ReadByte()
		if err != nil {
			if err != io.EOF {
				failures.Handle(err, "Reader failed")
			}
			break
		}

		logging.Debug("STDIN: %s", string(input))
		stdinWriter.Write([]byte{input})
	}

	if err := execCmd.Wait(); err != nil {
		failures.Handle(err, "Command exited with erro")
		Command.Exiter(1)
	}
}
