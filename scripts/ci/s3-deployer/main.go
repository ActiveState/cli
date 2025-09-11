package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ActiveState/cli/internal/condition"
)

var sourcePath, awsRegionName, awsBucketName, awsBucketPrefix string

var client *s3.Client

func main() {
	if !condition.InUnitTest() {
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

	createClient()
	fileList := getFileList()

	// Upload the files
	fmt.Printf("Uploading %d files\n", len(fileList))
	for _, path := range fileList {
		params := prepareFile(path)
		uploadFile(params)
	}
}

func createClient() {
	var err error

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsRegionName),
	)
	if err != nil {
		log.Fatalf("failed to load config, %s", err.Error())
	}

	// For Windows workstations, you might need to handle profile selection differently
	if runtime.GOOS == "windows" && !condition.OnCI() {
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(awsRegionName),
			config.WithSharedConfigProfile("mfa"),
		)
		if err != nil {
			log.Fatalf("failed to load config with profile, %s", err.Error())
		}
	}

	client = s3.NewFromConfig(cfg)
}

func getFileList() []string {
	// Get list of files to upload
	fmt.Printf("Getting list of files\n")
	fileList := []string{}

	err := os.MkdirAll(sourcePath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create directory", sourcePath, err)
		os.Exit(1)
	}

	if err = filepath.Walk(sourcePath, func(p string, f os.FileInfo, err error) error {
		if isDirectory(p) {
			return nil
		}
		fileList = append(fileList, p)
		return nil
	}); err != nil {
		fmt.Println("Failed to walk directory", sourcePath, err)
		os.Exit(1)
	}

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

	_, err = file.Read(buffer)
	if err != nil {
		fmt.Println("Failed to read file", file, err)
		os.Exit(1)
	}

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
		ACL:                types.ObjectCannedACLPublicRead,
	}

	return params
}

func uploadFile(params *s3.PutObjectInput) {
	_, err := client.PutObject(context.Background(), params)
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
