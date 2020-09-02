package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/google/uuid"
)

type languagePreset int

// Language presets
const (
	Perl languagePreset = iota
	Python
	Unknown
)

func (lp languagePreset) String() string {
	if lp == Perl {
		return "Perl"
	}
	if lp == Python {
		return "Python"
	}
	return pad("PRESET")
}

// InFile is the template file for the Product.wxs file
var InFile string = "installers/msi-language/Product.p.wxs"

// OutFile is the path to the generated file
var OutFile string = "installers/msi-language/Product.wxs"

type config struct {
	Preset              string
	ID                  string
	ProjectName         string
	Version             string
	CommitID            string
	ReleaseNotes        string
	Icon                string
	ProjectOwnerAndName string
	Visibility          string
	MSIVersion          string
}

func seededUUID(seed string) string {
	bytes := []byte(seed)
	hash := sha256.New()
	hash.Write(bytes)

	uuid := uuid.NewHash(hash, uuid.UUID{}, bytes, 0)
	return uuid.String()
}

func parsePreset(p string) (languagePreset, error) {
	if strings.ToLower(p) == "perl" {
		return Perl, nil
	}
	if strings.ToLower(p) == "python" {
		return Python, nil
	}
	return Unknown, fmt.Errorf("Invalid language preset: %s", p)
}

func icon(p languagePreset) (string, error) {
	if p == Perl {
		return "assets/perl.ico", nil
	}
	return "", fmt.Errorf("No icon for language preset %v", p)
}

func releaseNotes(p languagePreset, version string) (string, error) {
	if p == Perl {
		vParts := strings.Split(version, ".")
		if len(vParts) < 2 {
			return "", fmt.Errorf("invalid version format")
		}
		majorMinor := strings.Join(vParts[0:2], ".")
		return fmt.Sprintf("http://docs.activestate.com/activeperl/%s/get/relnotes/", majorMinor), nil
	}
	return "", fmt.Errorf("No release notes for language preset %v", p)
}

// normalizes and validates the configuration
func normalize(preset languagePreset, c *config) (*config, error) {
	parts := strings.SplitN(c.ProjectOwnerAndName, "/", 2)
	if len(parts) != 2 {
		return c, fmt.Errorf("Second argument must be of type owner/project")
	}

	if c.Visibility != "Public" && c.Visibility != "Private" {
		return c, fmt.Errorf("Visibility needs to be set to 'Public' or 'Private'")
	}

	c.ProjectName = parts[1]
	c.ID = seededUUID(c.ProjectOwnerAndName)

	c.CommitID = constants.RevisionHash

	ic, err := icon(preset)
	if err != nil {
		return c, err
	}
	c.Icon = ic

	c.ReleaseNotes, err = releaseNotes(preset, c.Version)
	if err != nil {
		return c, err
	}

	if c.MSIVersion == "" {
		dateTime := time.Now().Format("2006-01-02T15:04:05-0700") // ISO 8601
		commitHash := constants.RevisionHashShort
		if len(commitHash) > 7 {
			commitHash = commitHash[:7]
		}
		c.MSIVersion = dateTime + "-" + commitHash
	}

	return c, nil
}

func pad(s string) string {
	return s + strings.Repeat("-", 246-4-len(s))
}

func baseConfig() *config {
	return &config{
		ID:                  "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
		Version:             "255.255.255.65535",
		Icon:                "./assets/as.ico",
		Preset:              Unknown.String(),
		Visibility:          "Public",
		CommitID:            constants.RevisionHash,
		ProjectOwnerAndName: pad("PROJECT_OWNER_AND_NAME"),
		ReleaseNotes:        pad("RELEASE_NOTES"),
		ProjectName:         pad("PROJECT_NAME"),
		MSIVersion:          msiVersionInfo(),
	}
}

func msiVersionInfo() string {
	dateTime := time.Now().Format("2006-01-02T15:04:05-0700") // ISO 8601
	commitHash := constants.RevisionHashShort
	if len(commitHash) > 7 {
		commitHash = commitHash[:7]
	}
	return dateTime + "-" + commitHash
}

func parseArgs(args []string) (*config, error) {
	if len(os.Args) == 5 {
		preset, err := parsePreset(os.Args[1])
		if err != nil {
			return nil, err
		}
		return normalize(preset, &config{
			Preset:              preset.String(),
			Visibility:          os.Args[2],
			ProjectOwnerAndName: os.Args[3],
			Version:             os.Args[4],
			MSIVersion:          msiVersionInfo(),
		})
	}

	if len(os.Args) == 2 && os.Args[1] == "base" {
		return baseConfig(), nil
	}

	return nil, fmt.Errorf("invalid arguments: Expected <preset> <visibility> <owner/name> <version> | \"base\"")
}

func run(args []string) error {
	c, err := parseArgs(args)
	if err != nil {
		return err
	}

	in, err := ioutil.ReadFile(filepath.FromSlash(InFile))
	if err != nil {
		return err
	}
	tmpl, err := template.New("Product.wxs").Parse(string(in))
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.FromSlash(OutFile))
	if err != nil {
		return err
	}
	defer f.Close()
	err = tmpl.Execute(f, c)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

}
