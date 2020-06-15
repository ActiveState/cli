package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Must have a path argument")
	}

	removePath := strings.TrimSuffix(os.Args[1], `\`)
	fmt.Printf("Attempting to remove path: %s\n", removePath)

	oldPath := os.Getenv("PATH")
	oldPathElements := strings.Split(oldPath, string(os.PathListSeparator))

	var newPathElements []string
	for _, element := range oldPathElements {
		element = strings.TrimSuffix(element, `\`)
		if element == removePath {
			continue
		}
		newPathElements = append(newPathElements, element)
	}

	newPath := strings.Join(newPathElements, string(os.PathListSeparator))

	cmd := exec.Command("cmd.exe", "/V", "/C", "setx", "PATH", newPath)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
