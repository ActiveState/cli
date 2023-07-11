package main

import (
	"bytes"
	"fmt"
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

var sourcePath, awsRegionName, awsBucketName, awsBucketPrefix string

var sess *session.Session

func main() {
	if condition.InUnitTest() {
		return
	}

	app := filepath.Base(os.Args[0])

	if len(os.Args) != 5 {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s <source> <region-name> <bucket-name> <bucket-prefix>", app,
		)
		os.Exit(1)
	}

	sourcePath = os.Args[1]
	awsRegionName = os.Args[2]
	awsBucketName = os.Args[3]
	awsBucketPrefix = os.Args[4]

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", app, err.Error())
		os.Exit(1)
	}
}

func run() error {
	fmt.Printf("Uploading files from %s\n", sourcePath)

	if err := createSession(); err != nil {
		return err
	}

	fmt.Printf("Getting list of files\n")
	fileList, err := getFileList()
	if err != nil {
		return err
	}

	// Upload the files
	fmt.Printf("Uploading %d files\n", len(fileList))
	for _, path := range fileList {
		fmt.Printf("Preparing %s\n", path)
		params, err := prepareFile(path)
		if err != nil {
			return err
		}
		if params.Key != nil {
			fmt.Printf(" \\- Destination: %s\n", *params.Key)
		}

		fmt.Printf("Uploading %s\n", path)
		uploadFile(params)
	}

	return nil
}

type logger struct{}

func (l *logger) Log(v ...interface{}) {
	fmt.Printf("AWS Log: %v", v)
}

func createSession() error {
	// Specify profile to load for the session's config
	var err error
	verboseErr := true
	logLevel := aws.LogDebug
	_ = logLevel
	opts := session.Options{
		Config: aws.Config{
			CredentialsChainVerboseErrors: &verboseErr,
			Region:                        aws.String(awsRegionName),
			/*Logger:                        &logger{},*/
			/*LogLevel:                      &logLevel,*/
		},
	}
	if runtime.GOOS == "windows" && !condition.OnCI() {
		opts.Profile = "mfa" // For some reason on windows workstations this is necessary
	}
	sess, err = session.NewSessionWithOptions(opts)
	if err != nil {
		return fmt.Errorf("Failed to create session, %w", err)
	}

	return nil
}

func getFileList() ([]string, error) {
	// Get list of files to upload
	emsg := "Cannot get file list: %w"

	if err := os.MkdirAll(sourcePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf(emsg, err)
	}

	fileList := []string{}
	err := filepath.Walk(sourcePath, func(p string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		isDir, err := isDirectory(p)
		if err != nil {
			return fmt.Errorf("Cannot walk %q: %w", sourcePath, err)
		}
		if isDir {
			return nil
		}
		fileList = append(fileList, p)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(emsg, err)
	}

	return fileList, nil
}

func prepareFile(p string) (*s3.PutObjectInput, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("Failed to prepare file %q: %w", file.Name(), err)
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

	params := &s3.PutObjectInput{
		Bucket:             aws.String(awsBucketName),
		Key:                aws.String(key),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(size),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		ACL:                aws.String("public-read"),
	}

	return params, nil
}

func uploadFile(params *s3.PutObjectInput) error {
	s3Svc := s3.New(sess)
	_, err := s3Svc.PutObject(params)
	if err != nil {
		return fmt.Errorf(
			"Failed to upload data to %s/%s: %w", awsBucketName, *params.Key, err,
		)
	}

	return nil
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

	// When tests are ran with coverage the location of this file is
	// changed to a temp file, and we have to adjust accordingly
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

func isDirectory(p string) (bool, error) {
	fd, err := os.Stat(p)
	if err != nil {
		return false, fmt.Errorf("Cannot determine if %q is a directory: %w", p, err)
	}
	switch mode := fd.Mode(); {
	case mode.IsDir():
		return true, nil
	case mode.IsRegular():
		return false, nil
	}
	return false, nil
}
