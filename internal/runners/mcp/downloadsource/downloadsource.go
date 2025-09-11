package downloadsource

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type DownloadSourceRunner struct {
	output output.Outputer
}

func New(p *primer.Values) *DownloadSourceRunner {
	return &DownloadSourceRunner{
		output: p.Output(),
	}
}

type Params struct {
	sourceURI  string
	targetFile string
}

func NewParams(sourceURI string, targetFile string) *Params {
	return &Params{
		sourceURI:  sourceURI,
		targetFile: targetFile,
	}
}

// Download source code from a specified URI (S3 or HTTP/HTTPS) and unpacks it (.tar.gz).
// If a target file is specified, it will extract that specific file from the archive.
// Otherwise, it will list the files in the archive.
func (runner *DownloadSourceRunner) Run(params *Params) error {
	parsedURL, err := url.Parse(params.sourceURI)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	var reader io.ReadCloser
	var fileName string

	switch parsedURL.Scheme {
	case "s3":
		reader, fileName, err = DownloadFromS3(parsedURL)
		if err != nil {
			return err
		}
	case "http", "https":
		reader, fileName, err = DownloadFromHTTPS(parsedURL)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported URL scheme: %s (only s3://, http://, https:// are supported)", parsedURL.Scheme)
	}
	defer reader.Close()

	if !strings.HasSuffix(fileName, ".tar.gz") && !strings.HasSuffix(fileName, ".tgz") {
		return fmt.Errorf("file '%s' is not a .tar.gz file", fileName)
	}

	err = ProcessTarGz(reader, params.targetFile, runner.output)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFromS3(parsedURL *url.URL) (io.ReadCloser, string, error) {
	bucket := parsedURL.Host
	key := strings.TrimPrefix(parsedURL.Path, "/")

	// Load AWS config with profile and region
	var configOptions []func(*config.LoadOptions) error

	configOptions = append(configOptions, config.WithSharedConfigProfile("sso"))
	configOptions = append(configOptions, config.WithRegion("us-east-1"))

	awsCfg, err := config.LoadDefaultConfig(context.Background(), configOptions...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Download file from S3
	result, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, filepath.Base(key), nil
}

func DownloadFromHTTPS(parsedURL *url.URL) (io.ReadCloser, string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	// Make the request
	resp, err := client.Get(parsedURL.String())
	if err != nil {
		return nil, "", fmt.Errorf("failed to download from HTTPS: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Extract filename from URL
	fileName := filepath.Base(parsedURL.Path)
	if fileName == "" || fileName == "." {
		fileName = "downloaded-file"
	}

	return resp.Body, fileName, nil
}

func ProcessTarGz(reader io.Reader, targetFile string, output output.Outputer) error {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Typeflag == tar.TypeDir { // Skip directories
			continue
		}

		pathParts := strings.Split(strings.TrimPrefix(header.Name, "./"), "/")

		// Only process root files and immediate subdirectories
		if len(pathParts) <= 3 {
			fileName := ""
			if len(pathParts) == 3 {
				fileName = filepath.Join(pathParts[len(pathParts)-2], pathParts[len(pathParts)-1])
			} else {
				fileName = filepath.Base(header.Name)
			}

			if targetFile != "" { // Read file mode, match target file and print its content
				if fileName == targetFile {
					content, err := io.ReadAll(tarReader)
					if err != nil {
						return fmt.Errorf("failed to read file content: %w", err)
					}
					output.Print(string(content))
					return nil
				}
			} else {
				output.Print(fileName) // List mode, just print the file name
			}
		}
	}

	if targetFile != "" {
		return fmt.Errorf("file '%s' not found in archive", targetFile)
	}

	return nil
}
