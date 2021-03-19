package main

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

func init() {
	sourcePath = "build/update"
	awsRegionName = "us-east-1"
	awsBucketName = "state-tool"
	awsBucketPrefix = "update/state"
}

func TestCreateSession(t *testing.T) {
	createSession()
	// succeeds if no panic/exit
}

func TestGetFileList(t *testing.T) {
	getFileList()
	// succeeds if no panic/exit
}

func TestPrepareFile(t *testing.T) {
	var params *s3.PutObjectInput
	params = prepareFile(os.Args[0])
	assert.NotNil(t, params, "Sets params")
}
