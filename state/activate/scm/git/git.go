package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/dvirsky/go-pylog/logging"
)

// IsGitURI returns whether or not the given URI points to a Git repository.
func IsGitURI(uri string) bool {
	// Check for git-specific URI components.
	if strings.HasSuffix(uri, ".git") ||
		strings.Contains(uri, "git@") ||
		strings.Contains(uri, "github.com") {
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
		print.Error(locale.T("error_state_activate"))
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
