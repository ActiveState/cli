package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
)

const checkFileText = "ACTIVESTATE"

func isInstallDirectory(installDirectory string) bool {
	licenseFilePath := filepath.Join(installDirectory, licenseFileName)

	// read the whole file at once
	b, err := ioutil.ReadFile(licenseFilePath)
	if err != nil {
		return false
	}

	//check whether s contains substring text
	return bytes.Contains(b, []byte(checkFileText))
}
