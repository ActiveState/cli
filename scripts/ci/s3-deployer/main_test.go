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
	assert.NoError(t, createSession(), "Creates session")
}

func TestGetFileList(t *testing.T) {
	_, err := getFileList()
	assert.NoError(t, err, "Gets file list")
}

func TestPrepareFile(t *testing.T) {
	params, err := prepareFile(os.Args[0])
	assert.NoError(t, err, "Prepares file")
	assert.NotNil(t, params, "Sets params")
}
