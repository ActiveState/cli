package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ActiveState/Zeridian-CLI/internal/structures"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var cCyan = color.New(color.FgCyan)
var cRed = color.New(color.FgRed)
var cBold = color.New(color.Bold)

var branch string

func init() {
	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install the given project in the current working directory",
		Run: func(cmd *cobra.Command, args []string) {
			cwd, _ := os.Getwd()

			if len(args) > 0 {
				cloneRepo(cmd, args)
			} else {
				doInstall(cmd, cwd)
			}
		},
	}
	installCmd.PersistentFlags().StringVarP(&branch, "branch", "b", "", "optional branch that you want to run your project at")

	rootCmd.AddCommand(installCmd)
}

func cloneRepo(cmd *cobra.Command, args []string) {
	url := args[0]

	cCyan.Printf("Invoking 'git clone %s ..' for you\n", url)
	cBold.Println("-----------------------------------------------")

	var path string
	var cloneCmd *exec.Cmd

	if len(args) > 1 {
		path = args[1]
		cloneCmd = exec.Command("git", "clone", url, path)
	} else {
		fmt.Println("Cloning without a target directory is not supported in the PoC")
		os.Exit(1)
	}

	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stdin = os.Stdin
	cloneCmd.Stderr = os.Stderr
	err := cloneCmd.Run()

	cBold.Println("-----------------------------------------------")

	if err != nil {
		cRed.Println("Error occurred while running 'git clone ..'")
		cRed.Println(err)
		os.Exit(1)
	}

	os.Chdir(path)

	if branch != "" {
		cCyan.Printf("Switching to branch: %s\n", branch)
		cBold.Println("-----------------------------------------------")

		coutCmd := exec.Command("git", "checkout", branch)

		coutCmd.Stdout = os.Stdout
		coutCmd.Stdin = os.Stdin
		coutCmd.Stderr = os.Stderr
		err := coutCmd.Run()

		cBold.Println("-----------------------------------------------")

		if err != nil {
			cRed.Println("Error occurred while running 'git checkout ..'")
			cRed.Println(err)
			os.Exit(1)
		}
	}

	doInstall(cmd, path)
}

func doInstall(cmd *cobra.Command, path string) {
	cCyan.Println("Pulling in your Zeridian Environment")

	os.Chdir(path)
	cwd, _ := os.Getwd()

	if _, err := os.Stat(cwd + "/.z"); err == nil {
		cRed.Println("A Zeridian Project has already been created at: " + cwd)
		os.Exit(1)
	}

	err := os.Mkdir(cwd+"/.z", 0755)
	if err != nil {
		cRed.Println("Failed creating .z directory")
		cRed.Println(err)
		os.Exit(1)
	}

	if _, err := os.Stat(cwd + "/zeridian.json"); err != nil {
		cRed.Println("No zeridian.json found at " + cwd)
		os.Exit(1)
	}

	dat, err := ioutil.ReadFile(cwd + "/zeridian.json")
	if err != nil {
		cRed.Println("Failed reading zeridian.json")
		cRed.Println(err)
		os.Exit(1)
	}

	var project structures.Project
	err = json.Unmarshal([]byte(dat), &project)

	if err != nil {
		cRed.Println("Failed parsing zeridian.json")
		cRed.Println(err)
		os.Exit(1)
	}

}
