package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/dvirsky/go-pylog/logging"
	"github.com/google/go-github/github"
)

// IsGitURI returns whether or not the given URI points to a Git repository.
func IsGitURI(uri string) bool {
	if strings.HasPrefix(uri, "git@github.com") ||
		strings.HasPrefix(uri, "http://github.com") ||
		strings.HasPrefix(uri, "https://github.com") {
		// Ensure this is a valid GitHub URL since the owner and repository name
		// ultimately need to be extracted properly.
		regex := regexp.MustCompile("^(git@github\\.com:|https?://github\\.com/)[^/]+/.*$")
		return regex.MatchString(uri)
	} else if strings.HasSuffix(uri, ".git") ||
		strings.Contains(uri, "git@") {
		return true
	}
	// Check for a '.git' directory or file in a local URI.
	if _, err := os.Stat(uri); err == nil {
		_, err = os.Stat(filepath.Join(uri, ".git"))
		return err == nil
	}
	return false
}

// Git represents a Git repository to clone locally.
type Git struct {
	URI  string // the URI of the repository to clone
	path string // the local path to clone into
}

// ConfigFileExists returns whether or not the ActiveState config file exists in
// the repository, PRIOR to cloning (not after).
func (g *Git) ConfigFileExists() bool {
	if strings.HasPrefix(g.URI, "git@github.com") ||
		strings.HasPrefix(g.URI, "http://github.com") ||
		strings.HasPrefix(g.URI, "https://github.com") {
		client := github.NewClient(nil)
		regex := regexp.MustCompile("^.+github\\.com[:/]([^/]+)/(.+)$")
		matches := regex.FindStringSubmatch(strings.TrimSuffix(g.URI, ".git"))[1:]
		reader, err := client.Repositories.DownloadContents(context.Background(), matches[0], matches[1], constants.ConfigFileName, nil)
		if err != nil {
			return false // assume does not exist
		}
		reader.Close()
	} /*else {
		return true // assume it exists for now
	}*/
	return true
}

// SetPath sets the Git repository's local path.
func (g *Git) SetPath(path string) {
	g.path = path
}

// Path returns the Git repository's local path.
func (g *Git) Path() string {
	return g.path
}

// Clone clones the Git repository into its given or computed directory.
func (g *Git) Clone() error {
	logging.Debug("Attempting to clone %+v", g)
	if g.path == "" {
		g.path = g.humanishPart()
		logging.Debug("Determined 'humanish' dir to clone into as '%s'", g.path)
	}
	print.Info(locale.T("info_state_activate_uri", map[string]interface{}{
		"URI": g.URI, "Dir": g.path,
	}))

	cmd := exec.Command("git", "clone", g.URI, g.path)
	fmt.Println(strings.Join(cmd.Args, " ")) // match command output style
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// Computes the 'humanish' part of the source repository in order to use it as
// the directory to clone into if it wasn't explicitly given.
// This computation is based on git clone's shell script.
func (g *Git) humanishPart() string {
	re := regexp.MustCompile(":*[/\\\\]*\\.git$")
	path := re.ReplaceAllString(strings.TrimRight(g.URI, "/"), "")
	re = regexp.MustCompile(".*[/\\\\:]")
	return re.ReplaceAllString(path, "")
}
