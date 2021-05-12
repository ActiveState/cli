package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/ActiveState/cli/internal/condition"
)

const awsProfileName = "default"

var sourcePath, awsRegionName, awsBucketName, awsBucketPrefix string

var sess *session.Session

func main() {
	if !condition.InTest() {
		if len(os.Args) != 5 {
			log.Fatalf("Usage: %s <source> <region-name> <bucket-name> <bucket-prefix>", os.Args[0])
		}

		sourcePath = os.Args[1]
		awsRegionName = os.Args[2]
		awsBucketName = os.Args[3]
		awsBucketPrefix = os.Args[4]

		run()
	}
}

func run() {
	fmt.Printf("Uploading files from %s\n", sourcePath)

	createSession()
	fileList := getFileList()

	// Upload the files
	fmt.Printf("Uploading %d files\n", len(fileList))
	for _, path := range fileList {
		params := prepareFile(path)
		uploadFile(params)
	}
}

type logger struct{}

func (l *logger) Log(v ...interface{}) {
	fmt.Printf("AWS Log: %v", v)
}

func createSession() {
	// Specify profile to load for the session's config
	var err error
	var verboseErr = true
	var logLevel = aws.LogDebug
	_ = logLevel
	sess, err = session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			CredentialsChainVerboseErrors: &verboseErr,
			Region:                        aws.String(awsRegionName),
			/*Logger:                        &logger{},*/
			/*LogLevel:                      &logLevel,*/
		},
	})
	if err != nil {
		log.Fatalf("failed to create session, %s", err.Error())
		os.Exit(1)
	}
}

func getFileList() []string {
	// Get list of files to upload
	fmt.Printf("Getting list of files\n")
	fileList := []string{}
	os.MkdirAll(sourcePath, os.ModePerm)
	filepath.Walk(sourcePath, func(p string, f os.FileInfo, err error) error {
		if isDirectory(p) {
			return nil
		}
		fileList = append(fileList, p)
		return nil
	})
	return fileList
}

func prepareFile(p string) *s3.PutObjectInput {
	fmt.Printf("Uploading %s\n", p)

	file, err := os.Open(p)
	if err != nil {
		fmt.Println("Failed to open file", file, err)
		os.Exit(1)
	}

	// We just created our file, so no need to err check .Stat()
	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	defer file.Close()
	var key string
	key = normalizePath(awsBucketPrefix + p)
	key = strings.Replace(key, normalizePath(sourcePath), "", 1)
	key = strings.Replace(key, normalizePath(path.Join(getRootPath(), "public")), "", 1)
	fmt.Printf(" \\- Destination: %s\n", key)

	params := &s3.PutObjectInput{
		Bucket:             aws.String(awsBucketName),
		Key:                aws.String(key),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(size),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		ACL:                aws.String("public-read"),
	}

	return params
}

func uploadFile(params *s3.PutObjectInput) {
	s3Svc := s3.New(sess)
	_, err := s3Svc.PutObject(params)
	if err != nil {
		fmt.Printf("Failed to upload data to %s/%s, %s\n",
			awsBucketName, *params.Key, err.Error())
		os.Exit(1)
	}
}

func normalizePath(p string) string {
	return path.Join(strings.Split(p, "\\")...)
}

func getRootPath() string {
	pathsep := string(os.PathSeparator)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("Could not call Caller(0)")
	}

	abs := path.Dir(file)

	// When tests are ran with coverage the location of this file is changed to a temp file, and we have to
	// adjust accordingly
	if strings.HasSuffix(abs, "_obj_test") {
		abs = ""
	}

	var err error
	abs, err = filepath.Abs(path.Join(abs, "..", ".."))

	if err != nil {
		return ""
	}

	return abs + pathsep
}

func isDirectory(p string) bool {
	fd, err := os.Stat(p)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	switch mode := fd.Mode(); {
	case mode.IsDir():
		return true
	case mode.IsRegular():
		return false
	}
	return false
}
