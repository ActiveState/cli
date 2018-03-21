package distribution

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/download"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/fileutils"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"

	"github.com/ActiveState/ActiveState-CLI/internal/artefact"

	"github.com/ActiveState/sysinfo"
	"github.com/mholt/archiver"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
)

// FailArtefactMeta is a failure used when the artifact meta is invalid
var FailArtefactMeta = failures.Type("distribution.fail.artefactmeta", failures.FailVerify)

// Artefact reflects and entry from the distribution.json file
type Artefact struct {
	Hash     string
	Parent   string
	Download string
}

// Sanitized is a sanitized version of the distribution.json that can be more easily interpreted
type Sanitized struct {
	Languages []*artefact.Artefact
	Artefacts map[string][]*artefact.Artefact
}

// variables that tests will override
var dist []Artefact

// Obtain will obtain the latest distribution data and ensure all artifacts are downloaded
func Obtain() (*Sanitized, *failures.Failure) {
	var fail *failures.Failure
	if dist == nil {
		dist, fail = ObtainArtefacts()
		if fail != nil {
			return nil, fail
		}
	}

	var entries []*download.Entry
	for _, distArtf := range dist {
		if NeedToObtainArtefact(&distArtf) {
			target, fail := PrepareForArtefact(&distArtf)
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
			InstallArtefact(entry.Data.(Artefact), entry.Path, entry)
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

func sanitize(distArtefacts []Artefact) (*Sanitized, *failures.Failure) {
	sanitized := Sanitized{}
	sanitized.Artefacts = make(map[string][]*artefact.Artefact)

	for _, distArtf := range distArtefacts {
		artf, fail := artefact.Get(distArtf.Hash)
		if fail != nil {
			return nil, fail
		}

		switch artf.Meta.Type {
		case "language":
			sanitized.Languages = append(sanitized.Languages, artf)
		default:
			if distArtf.Parent == "" {
				return nil, FailArtefactMeta.New("err_artefact_no_parent", distArtf.Hash)
			}

			if _, ok := sanitized.Artefacts[distArtf.Parent]; !ok {
				sanitized.Artefacts[distArtf.Parent] = []*artefact.Artefact{}
			}
			sanitized.Artefacts[distArtf.Parent] = append(sanitized.Artefacts[distArtf.Parent], artf)
		}
	}

	return &sanitized, nil
}

// ObtainArtefacts will download the given artefacts
func ObtainArtefacts() ([]Artefact, *failures.Failure) {
	dist := []Artefact{}

	os := sysinfo.OS().String()
	arch := sysinfo.Architecture().String()
	platform := strings.ToLower(fmt.Sprintf("%s-%s", os, arch))
	url := fmt.Sprintf("%sdistro/%s/distribution.json", constants.APIArtefactURL, platform)

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

// NeedToObtainArtefact will check whether the given artefact will need to be obtained
func NeedToObtainArtefact(distArtf *Artefact) bool {
	if artefact.Exists(distArtf.Hash) {
		return false
	}

	return true
}

// PrepareForArtefact will ensure everything is in place for the given artefact to be obtained
func PrepareForArtefact(distArtf *Artefact) (string, *failures.Failure) {
	path := artefact.GetPath(distArtf.Hash)

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

	defer os.Remove(out.Name())

	return out.Name(), nil
}

// InstallArtefact will install the given artefact from a local source archive
func InstallArtefact(distArtf Artefact, source string, entry *download.Entry) *failures.Failure {
	path := artefact.GetPath(distArtf.Hash)

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

	_, fail = artefact.Get(distArtf.Hash)
	if fail != nil {
		return fail
	}

	return nil
}
