package main

import (
	"log"
	"os"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
)

func main() {
	if len(os.Args) != 4 {
		log.Println("Need to run with argument <from-dir> <to-dir> <log-file>")
		os.Exit(1)
	}
	fromDir := os.Args[1]
	toDir := os.Args[2]

	logFile := os.Args[3]
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("error initializing log file: %v", err)
	}
	defer f.Close()
	logger := log.New(f, "installer", log.LstdFlags)
	logger.Printf("Installing %s -> %s", fromDir, toDir)
	err = installer.Install(fromDir, toDir, logger)
	if err != nil {
		logger.Printf("Installation failed: %v", err)
	}
	logger.Printf("Installation from %s -> %s was successful.", fromDir, toDir)
}
