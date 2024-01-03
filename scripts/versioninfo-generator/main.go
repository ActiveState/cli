package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/blang/semver"
)

const (
	versionInfoAssetsFileName = "versioninfo.json"
	versionInfoLegalCopyright = "© ActiveState Software, Inc. %d"
)

type versionInfo struct {
	FixedFileInfo  FixedFileInfo  `json:"FixedFileInfo"`
	StringFileInfo StringFileInfo `json:"StringFileInfo"`

	// The remainder is data we don't care about but need
	// to preserve in the JSON file.
	VarFileInfo  VarFileInfo `json:"VarFileInfo"`
	IconPath     string      `json:"IconPath"`
	ManifestPath string      `json:"ManifestPath"`
}

type FixedFileInfo struct {
	FileVersion    FileVersion    `json:"FileVersion"`
	ProductVersion ProductVersion `json:"ProductVersion"`

	// The remainder is data we don't care about but need
	// to preserve in the JSON file.
	FileFlagsMask string `json:"FileFlagsMask"`
	FileFlags     string `json:"FileFlags"`
	FileOS        string `json:"FileOS"`
	FileType      string `json:"FileType"`
	FileSubType   string `json:"FileSubType"`
}

type FileVersion struct {
	Major int `json:"Major"`
	Minor int `json:"Minor"`
	Patch int `json:"Patch"`
	Build int `json:"Build"`
}

type ProductVersion struct {
	Major int `json:"Major"`
	Minor int `json:"Minor"`
	Patch int `json:"Patch"`
	Build int `json:"Build"`
}

type StringFileInfo struct {
	FileVersion    string `json:"FileVersion"`
	ProductVersion string `json:"ProductVersion"`

	// The remainder is data we don't care about but need
	// to preserve in the JSON file.
	Comments         string `json:"Comments"`
	CompanyName      string `json:"CompanyName"`
	FileDescription  string `json:"FileDescription"`
	InternalName     string `json:"InternalName"`
	LegalCopyright   string `json:"LegalCopyright"`
	LegalTrademarks  string `json:"LegalTrademarks"`
	OriginalFilename string `json:"OriginalFilename"`
	PrivateBuild     string `json:"PrivateBuild"`
	ProductName      string `json:"ProductName"`
	SpecialBuild     string `json:"SpecialBuild"`
}

type VarFileInfo struct {
	Translation Translation `json:"Translation"`
}

type Translation struct {
	LangID    string `json:"LangID"`
	CharsetID string `json:"CharsetID"`
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if len(os.Args) != 4 {
		return errs.New("Usage: versioninfo-generator <version.txt> <versioninfo.json>")
	}

	versionFile := os.Args[1]
	versionInfoFilePath := os.Args[2]
	productName := os.Args[3]

	// Read version.txt
	version, err := os.ReadFile(versionFile)
	if err != nil {
		return errs.Wrap(err, "failed to read version.txt")
	}

	// The goversioninfo library does not support prerelease versions
	if strings.Contains(string(version), "-") {
		parts := strings.SplitN(string(version), "-", 2)
		version = []byte(parts[0])
	}

	// Parse semver from version.txt
	semver, err := semver.Parse(string(version))
	if err != nil {
		return errs.Wrap(err, "failed to parse version.txt")
	}

	versionInfoFile, err := assets.ReadFileBytes(versionInfoAssetsFileName)
	if err != nil {
		return errs.Wrap(err, "failed to read versioninfo.json")
	}

	// Parse versioninfo.json to data map
	var data map[string]interface{}
	if err := json.Unmarshal(versionInfoFile, &data); err != nil {
		return errs.Wrap(err, "failed to parse versioninfo.json")
	}

	// Parse versioninfo.json
	var versionInfo versionInfo
	if err := json.Unmarshal(versionInfoFile, &versionInfo); err != nil {
		return errs.Wrap(err, "failed to parse versioninfo.json")
	}

	// Update versioninfo.json
	versionInfo.FixedFileInfo.FileVersion.Major = int(semver.Major)
	versionInfo.FixedFileInfo.FileVersion.Minor = int(semver.Minor)
	versionInfo.FixedFileInfo.FileVersion.Patch = int(semver.Patch)
	versionInfo.FixedFileInfo.ProductVersion.Major = int(semver.Major)
	versionInfo.FixedFileInfo.ProductVersion.Minor = int(semver.Minor)
	versionInfo.FixedFileInfo.ProductVersion.Patch = int(semver.Patch)
	versionInfo.StringFileInfo.FileVersion = semver.String()
	versionInfo.StringFileInfo.ProductVersion = semver.String()
	versionInfo.StringFileInfo.ProductName = productName
	versionInfo.StringFileInfo.LegalCopyright = fmt.Sprintf(versionInfoLegalCopyright, time.Now().Year())

	// Marshal the updated versioninfo.json
	finalJSON, err := json.MarshalIndent(versionInfo, "", "  ")
	if err != nil {
		return errs.Wrap(err, "failed to marshal versioninfo.json")
	}

	// Save the final JSON back to the file
	err = os.WriteFile(versionInfoFilePath, finalJSON, 0644)
	if err != nil {
		return errs.Wrap(err, "failed to write versioninfo.json")
	}
	fmt.Println("Successfully generated versioninfo.json at ", versionInfoFilePath)

	return nil

}
