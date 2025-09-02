package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ActiveState/cli/internal/condition"
)

var awsRegionName, awsBucketName string
var client *s3.Client

func main() {
	if !condition.InUnitTest() {
		if len(os.Args) != 3 {
			log.Fatalf("Usage: %s <region-name> <bucket-name>", os.Args[0])
		}

		awsRegionName = os.Args[1]
		awsBucketName = os.Args[2]

		run()
	}
}

func run() {
	fmt.Printf("Promoting staging files to production in bucket: %s\n", awsBucketName)

	createClient()

	// List all objects with staging/ prefix
	stagingObjects, err := listObjectsWithPrefix("staging/")
	if err != nil {
		log.Fatalf("Failed to list staging objects: %v", err)
	}

	if len(stagingObjects) == 0 {
		fmt.Println("No staging files found to promote.")
		return
	}

	fmt.Printf("Found %d staging files to promote:\n", len(stagingObjects))
	for _, obj := range stagingObjects {
		fmt.Printf("  - %s\n", *obj.Key)
	}

	// Copy each staging object to production location and delete the staging version
	for _, obj := range stagingObjects {
		stagingKey := *obj.Key
		productionKey := strings.TrimPrefix(stagingKey, "staging/")

		fmt.Printf("Promoting %s -> %s\n", stagingKey, productionKey)

		err := copyObject(stagingKey, productionKey)
		if err != nil {
			log.Fatalf("Failed to copy %s to %s: %v", stagingKey, productionKey, err)
		}

		err = deleteObject(stagingKey)
		if err != nil {
			log.Fatalf("Failed to delete staging object %s: %v", stagingKey, err)
		}
	}

	fmt.Printf("Successfully promoted %d files from staging to production.\n", len(stagingObjects))
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

func listObjectsWithPrefix(prefix string) ([]types.Object, error) {
	var objects []types.Object

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(awsBucketName),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		objects = append(objects, page.Contents...)
	}

	return objects, nil
}

func copyObject(sourceKey, destinationKey string) error {
	copySource := fmt.Sprintf("%s/%s", awsBucketName, sourceKey)

	_, err := client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(awsBucketName),
		CopySource: aws.String(copySource),
		Key:        aws.String(destinationKey),
		ACL:        types.ObjectCannedACLPublicRead,
	})

	return err
}

func deleteObject(key string) error {
	_, err := client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(awsBucketName),
		Key:    aws.String(key),
	})

	return err
}
