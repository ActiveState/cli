package activate

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/ActiveState/ActiveState-CLI/internal/structures"
	"github.com/ActiveState/cobra"
	"github.com/dvirsky/go-pylog/logging"
)

// Command holds our main command definition
var Command = &structures.Command{
	Name:        "activate",
	Description: "activate_project",
	Run:         Execute,
}

type fn func(string, ...interface{}) (int, error)

// Execute the activate command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Activating")

	if len(args) < 1 {
		fmt.Println("requires at least one arg")
		return
	}
	project := args[0]

	out(1, "Getting ready to develop %s", args[0])
	out(1, "Installing python runtime for OSX")
	out(1, "Cloning %s", args[0])
	out(1, "Installing dependencies")
	out(1, "Let's get coding!")

	shellArgs := []string{"bash"}
	binary := os.Getenv("SHELL")
	env := append(
		os.Environ(),
		fmt.Sprintf("PS1=\\e[1;94m($ %s) \\e[m", project))

	// TODO: install shims for git and pip
	syscall.Exec(binary, shellArgs, env)

}

func out(delay time.Duration, s string, a ...interface{}) {
	fmt.Printf(s+"\n", a...)
	time.Sleep(delay * time.Second)
}
