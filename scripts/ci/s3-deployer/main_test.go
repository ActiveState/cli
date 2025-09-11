package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	sourcePath = "build/update"
	awsRegionName = "us-east-1"
	awsBucketName = "state-tool"
	awsBucketPrefix = "update/state"
}

func TestCreateSession(t *testing.T) {
	createClient()
	// succeeds if no panic/exit
}

func TestGetFileList(t *testing.T) {
	getFileList()
	// succeeds if no panic/exit
}

func TestPrepareFile(t *testing.T) {
	params := prepareFile(os.Args[0])
	assert.NotNil(t, params, "Sets params")
}
