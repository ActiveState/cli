package preprocess

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const bucketPrefix = "update/state/versions/"

type versionFile struct {
	Increment string
}

func startSession() (*session.Session, error) {
	// Enable loading shared config file
	os.Setenv("aws_SDK_LOAD_CONFIG", "1")

	// Specify profile to load for the session's config
	return session.NewSessionWithOptions(session.Options{
		Profile: "default",
		Config:  aws.Config{Region: aws.String("ca-central-1")},
	})
}

func getVersionString(branchName string) (string, error) {
	session, err := startSession()
	if err != nil {
		return "", err
	}

	downloader := s3manager.NewDownloader(session)

	var buffer []byte
	atBuffer := aws.NewWriteAtBuffer(buffer)

	params := &s3.GetObjectInput{
		Bucket: aws.String("cli-update"),
		Key:    aws.String(fmt.Sprintf("%s%s/version.json", bucketPrefix, branchName)),
	}

	_, err = downloader.Download(atBuffer, params)
	if err != nil {
		return "", err
	}

	file := &versionFile{}
	err = json.Unmarshal(atBuffer.Bytes(), file)
	if err != nil {
		return "", err
	}

	return file.Increment, nil
}
