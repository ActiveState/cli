package distribution

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/sysinfo"
	"github.com/mholt/archiver"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

// FailArtifactMeta is a failure used when the artifact meta is invalid
var FailArtifactMeta = failures.Type("distribution.fail.artifactmeta", failures.FailVerify)

// Artifact reflects and entry from the distribution.json file
type Artifact struct {
	Hash     string
	Parent   string
	Download string
}

// Sanitized is a sanitized version of the distribution.json that can be more easily interpreted
type Sanitized struct {
	Languages []*artifact.Artifact
	Artifacts map[string][]*artifact.Artifact
}

// variables that tests will override
var dist []Artifact

// Obtain will obtain the latest distribution data and ensure all artifacts are downloaded
func Obtain() (*Sanitized, *failures.Failure) {
	var fail *failures.Failure
	if dist == nil {
		dist, fail = ObtainArtifacts()
		if fail != nil {
			return nil, fail
		}
	}

	var entries []*download.Entry
	for _, distArtf := range dist {
		if NeedToObtainArtifact(&distArtf) {
			target, fail := PrepareForArtifact(&distArtf)
			if fail != nil {
				return nil, fail
			}

			entry := &download.Entry{Path: target, Download: distArtf.Download, Data: distArtf}
			entries = append(entries, entry)
		}
	}

	if len(entries) > 0 {
		print.Info(locale.T("distro_obtaining"))

		manager := download.New(entries, 5)
		fail = manager.Download()
		if fail != nil {
			return nil, fail
		}

		print.Info(locale.T("distro_installing"))
		progress := mpb.New()
		bar := progress.AddBar(int64(len(entries)),
			mpb.PrependDecorators(
				decor.CountersNoUnit("%d / %d", 10, 0),
			),
			mpb.AppendDecorators(
				decor.ETA(3, 0),
			))
		for _, entry := range entries {
			InstallArtifact(entry.Data.(Artifact), entry.Path, entry)
			bar.Increment()
		}

		progress.Wait()
	}

	sanitized, fail := sanitize(dist)
	if fail != nil {
		return nil, fail
	}

	return sanitized, nil
}

func sanitize(distArtifacts []Artifact) (*Sanitized, *failures.Failure) {
	sanitized := Sanitized{}
	sanitized.Languages = []*artifact.Artifact{}
	sanitized.Artifacts = make(map[string][]*artifact.Artifact)

	for _, distArtf := range distArtifacts {
		artf, fail := artifact.Get(distArtf.Hash)
		if fail != nil {
			return nil, fail
		}

		switch artf.Meta.Type {
		case "language":
			sanitized.Languages = append(sanitized.Languages, artf)
		default:
			if distArtf.Parent == "" {
				return nil, FailArtifactMeta.New("err_artifact_no_parent", distArtf.Hash)
			}

			if _, ok := sanitized.Artifacts[distArtf.Parent]; !ok {
				sanitized.Artifacts[distArtf.Parent] = []*artifact.Artifact{}
			}
			sanitized.Artifacts[distArtf.Parent] = append(sanitized.Artifacts[distArtf.Parent], artf)
		}
	}

	return &sanitized, nil
}

// ObtainArtifacts will download the given artifacts
func ObtainArtifacts() ([]Artifact, *failures.Failure) {
	dist := []Artifact{}

	os := sysinfo.OS().String()
	arch := sysinfo.Architecture().String()
	platform := strings.ToLower(fmt.Sprintf("%s-%s", os, arch))
	url := fmt.Sprintf("%sdistro/%s/distribution.json", constants.APIArtifactURL, platform)

	logging.Debug("Using distro URL: %s", url)

	body, fail := download.Get(url)
	if fail != nil {
		return nil, fail
	}

	err := json.Unmarshal(body, &dist)
	if err != nil {
		return nil, failures.FailMarshal.Wrap(err)
	}

	return dist, nil
}

// NeedToObtainArtifact will check whether the given artifact will need to be obtained
func NeedToObtainArtifact(distArtf *Artifact) bool {
	if artifact.Exists(distArtf.Hash) {
		return false
	}

	return true
}

// PrepareForArtifact will ensure everything is in place for the given artifact to be obtained
func PrepareForArtifact(distArtf *Artifact) (string, *failures.Failure) {
	path := artifact.GetPath(distArtf.Hash)

	if fileutils.DirExists(path) {
		err := os.Remove(path)
		if err != nil {
			return "", failures.FailIO.Wrap(err)
		}
	}

	os.MkdirAll(path, os.ModePerm)

	out, err := ioutil.TempFile(os.TempDir(), distArtf.Hash)
	out.Close()

	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}

	// We don't want the temp file to be created yet, we just need a unique path, so delete it
	os.Remove(out.Name())

	return out.Name(), nil
}

// InstallArtifact will install the given artifact from a local source archive
func InstallArtifact(distArtf Artifact, source string, entry *download.Entry) *failures.Failure {
	path := artifact.GetPath(distArtf.Hash)

	hash, fail := fileutils.Hash(source)
	if fail != nil {
		return fail
	}

	if hash != distArtf.Hash {
		return failures.FailVerify.New("err_hash_mismatch", source, hash, distArtf.Hash)
	}

	err := archiver.TarGz.Open(source, path)
	if err != nil {
		return failures.FailArchiving.Wrap(err)
	}

	_, fail = artifact.Get(distArtf.Hash)
	if fail != nil {
		return fail
	}

	return nil
}
