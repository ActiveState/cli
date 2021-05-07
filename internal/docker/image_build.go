package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func main() {
	url := "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/release/0.25.1-SHAd98b4c4/linux-amd64.tar.gz"
	image := "ubuntu:latest"
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	dockerFile, err := dockerFile(image, url)
	if err != nil {
		panic(err)
	}
	dir, err := dockerContext(dockerFile)
	fmt.Printf("temporary directory is: \n%s\n", dir)
	defer os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
	err = imageBuild(cli, dir)
	if err != nil {
		panic(err)
	}
}

func dockerFile(image string, url string) (string, error) {
	tarSlice := strings.Split(url, "/")
	tarFile := tarSlice[len(tarSlice)-1]
	if !strings.HasSuffix(tarFile, ".tar.gz") {
		return "", errs.New("File does not end with '.tar.gz'")
	}
	fileName := strings.Split(tarFile, ".")[0]
	return "FROM " + image +
		"\nADD " + url + " /tmp/" + tarFile +
		"\nRUN tar -zxf /tmp/" + tarFile + " && mv " + fileName + " /usr/local/bin/state && rm /tmp/" + tarFile +
		"\nCMD [\"/bin/bash\"]", nil
}

func dockerContext(dockerfile string) (string, error) {
	d, err := ioutil.TempDir("", "stateTool")
	if err != nil {
		return "", errs.Wrap(err)
	}
	file, err := os.Create(d + "/Dockerfile")
	if err != nil {
		return "", errs.Wrap(err)
	}
	defer file.Close()
	file.WriteString(dockerfile)
	return d, nil
}

func imageBuild(dockerClient *client.Client, contextDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	tar, err := archive.TarWithOptions(contextDir, &archive.TarOptions{})
	if err != nil {
		return errs.Wrap(err)
	}

	// if _, err := dockerClient.ImageBuild(ctx, tar, types.ImageBuildOptions{}); err != nil {
	// 	return errs.Wrap(err)
	// }
	res, err := dockerClient.ImageBuild(ctx, tar, types.ImageBuildOptions{})
	if err != nil {
		return errs.Wrap(err)
	}
	err = print(res.Body)
	if err != nil {
		return errs.Wrap(err)
	}
	return nil
}

func print(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errs.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return errs.Wrap(err)
	}

	return nil
}
