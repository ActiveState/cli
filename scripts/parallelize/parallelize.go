package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
)

type Job struct {
	ID   string
	Args []string
	Env  []string
	If   string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, errs.JoinMessage(err))
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) <= 1{
		return errs.New("Must provide single argument with JSON blob, or [job <ID>] to check the results of a job.")
	}

	if os.Args[1] == "job" {
		if len(os.Args) != 3 {
			return errs.New("Must provide job ID")
		}
		return readJob(os.Args[2])
	}

	var jobs []Job
	if err := json.Unmarshal([]byte(strings.Join(os.Args[1:], "")), &jobs); err != nil {
		return errs.Wrap(err, "Invalid JSON. Args: %#v", os.Args[1:])
	}

	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(job Job) {
			defer wg.Done()
			runJob(job)
		}(job)
	}

	wg.Wait()

	return nil
}

func runJob(job Job) {
	fmt.Printf("Running: %s\n", job.ID)

	if job.If != "" {
		cond := constraints.NewPrimeConditional(nil, "", "", "", "")
		run, err := cond.Eval(job.If)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not evaluate conditonal: %#v", job)
		}
		if !run {
			fmt.Printf( "Skipping '%s' as per conditional: %s", job.ID, job.If)
			return
		}
	}
	if len(job.Args) == 0 {
		fmt.Fprintf(os.Stderr, "Job must have arguments: %#v", job)
		return
	}

	outname := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "parallelize", "jobs", fmt.Sprintf("%s.out", job.ID))
	outfile, err := os.Create(outname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create : %#v", job)
	}
	defer outfile.Close()

	code, _, err := exeutils.Execute(job.Args[0], job.Args[1:], func(cmd *exec.Cmd) error {
		cmd.Stdout = outfile
		cmd.Stderr = outfile
		cmd.Env = append(job.Env, os.Environ()...)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Executing job %s failed", job.ID)
		return
	}
	outfile.WriteString(fmt.Sprintf("\n%d", code)) // last entry is the exit code
	fmt.Printf("%s Completed, exit code: %d\n", job.ID, code)
}

func readJob(id string) error {
	jobfile := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "parallelize", "jobs", fmt.Sprintf("%s.out", id))
	if ! fileutils.FileExists(jobfile) {
		return errs.New("Job does not exist: %s", jobfile)
	}

	contents := strings.Split(string(fileutils.ReadFileUnsafe(jobfile)), "\n")
	code, err := strconv.Atoi(contents[len(contents)-1])
	if err != nil {
		return errs.Wrap(err,"Expected last line to be the exit code, instead found: %s", contents[len(contents)-1])
	}

	fmt.Println(strings.Join(contents[0:(len(contents)-2)], "\n"))
	os.Exit(code)

	return nil
}
