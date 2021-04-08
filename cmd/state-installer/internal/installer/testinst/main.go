package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
)

func main() {
	if len(os.Args) != 5 {
		log.Println("Need to run with argument <from-dir> <to-dir> <log-file> <timeout>")
		os.Exit(1)
	}
	fromDir := os.Args[1]
	toDir := os.Args[2]
	logFile := os.Args[3]
	timeout, err := strconv.Atoi(os.Args[4])
	if err != nil {
		log.Printf("Failed to parse timeout %s: %v", os.Args[4], err)
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("error initializing log file: %v", err)
	}
	defer f.Close()

	// pausing before installation (to give time to stop running executables)
	time.Sleep(time.Duration(timeout) * time.Second)

	logger := log.New(f, "installer", log.LstdFlags)
	logger.Printf("Installing %s -> %s", fromDir, toDir)
	err = installer.Install(fromDir, toDir, logger)
	if err != nil {
		logger.Printf("Installation failed: %v", err)
	}
	logger.Printf("Installation from %s -> %s was successful.", fromDir, toDir)
}
