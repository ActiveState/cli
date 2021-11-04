package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/gammazero/workerpool"
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

	if os.Args[1] == "results" {
		if len(os.Args) != 3 {
			return errs.New("Must provide job ID")
		}
		return readJob(os.Args[2])
	}

	jsonData := []byte(strings.Join(os.Args[1:], ""))
	var jobs []Job
	if err := json.Unmarshal(jsonData, &jobs); err != nil {
		return errs.Wrap(err, "Invalid JSON. Data: %s", jsonData)
	}

	wp := workerpool.New(3)
	for _, job := range jobs {
		func(job Job) {
			wp.Submit(func() {
				t := time.Now()
				fmt.Printf("Running: %s\n", job.ID)
				runJob(job)
				fmt.Printf("Finished %s after %s\n", job.ID, time.Since(t))
			})
		}(job)
	}
	wp.StopWait()

	return nil
}

func jobDir() string {
	path, err := storage.AppDataPath()
	if err != nil {
		panic(err)
	}

	path = filepath.Join(path, "jobs")
	if err := fileutils.MkdirUnlessExists(path); err != nil {
		panic(err)
	}

	return path
}

func runJob(job Job) {
	outname := filepath.Join(jobDir(), fmt.Sprintf("%s.out", job.ID))
	fmt.Printf("%s: saving to %s\n", job.ID, outname)

	outfile, err := os.Create(outname)
	if err != nil {
		panic(fmt.Sprintf("Could not create: %#v, error: %s\n", job, errs.JoinMessage(err)))
	}
	defer outfile.Close()

	failure := func(msg string, args ...interface{}) {
		fmt.Fprintf(outfile, msg + "\n1", args...)
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s: ", job.ID) + msg, args...)
	}

	if job.If != "" {
		cond := constraints.NewPrimeConditional(nil, "", "", "", "")
		run, err := cond.Eval(job.If)
		if err != nil {
			failure("Could not evaluate conditonal: %s, error: %s\n", job.If, errs.JoinMessage(err))
			return
		}
		if !run {
			fmt.Printf( "%s: Skipping as per conditional: %s\n", job.ID, job.If)
			return
		}
	}
	if len(job.Args) == 0 {
		failure("Job must have arguments: %#v\n", job)
		return
	}


	code, _, err := exeutils.Execute(job.Args[0] + osutils.ExeExt, job.Args[1:], func(cmd *exec.Cmd) error {
		cmd.Stdout = outfile
		cmd.Stderr = outfile
		cmd.Env = append(job.Env, os.Environ()...)
		return nil
	})
	if err != nil {
		failure("Executing job %s failed, error: %s", job.ID, errs.JoinMessage(err))
	}
	outfile.WriteString(fmt.Sprintf("\n%d", code)) // last entry is the exit code
	fmt.Printf("%s: Completed, exit code: %d\n", job.ID, code)
}

func readJob(id string) error {
	jobfile := filepath.Join(jobDir(), fmt.Sprintf("%s.out", id))
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
