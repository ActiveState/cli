package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
)

// MatchesRemote returns whether or not the given URI points to a Git repository.
func MatchesRemote(uri string) bool {
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
	// Check for a '.git' file in a local URI (directories are checkouts, ie not a remote)
	return fileutils.FileExists(filepath.Join(uri, ".git"))
}

// MatchesPath returns whether the given path is a git repository
func MatchesPath(path string) bool {
	return fileutils.DirExists(filepath.Join(path, ".git"))
}

// NewFromURI creates a new Git struct using the given uri
func NewFromURI(uri string) *Git {
	return &Git{uri: uri}
}

// NewFromPath creates a new Git struct using the given path
func NewFromPath(path string) *Git {
	return &Git{path: path}
}

// Git represents a Git repository to clone locally.
type Git struct {
	uri    string // the uri of the repository to clone
	path   string // the local path to clone into
	branch string // the branch to use
}

// SetPath sets the Git repository's local path.
func (g *Git) SetPath(path string) {
	g.path = path
}

// Path returns the Git repository's local path.
func (g *Git) Path() string {
	if g.path == "" {
		cwd, _ := os.Getwd()
		reponame := g.humanishPart()
		g.path = filepath.Join(cwd, reponame)
		logging.Debug("Determined 'humanish' dir to clone into as '%s'", reponame)
	}
	return g.path
}

// SetURI sets the Git repository's remote uri.
func (g *Git) SetURI(URI string) {
	g.uri = URI
}

// URI returns the Git repository's remote uri.
func (g *Git) URI() string {
	if g.uri == "" {
		out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
		if err == nil {
			g.uri = strings.Trim(string(out), "\n")
		} else {
			logging.Warning("Could not retrieve git remote from local repository")
		}
	}
	return g.uri
}

// SetBranch sets the Git repository's branch to use
func (g *Git) SetBranch(branch string) {
	g.branch = branch
}

// Branch returns the Git repository's branch
func (g *Git) Branch() string {
	return g.branch
}

// CheckoutBranch checks out the configured branch
func (g *Git) CheckoutBranch() error {
	cmd := exec.Command("git", "checkout", g.Branch())

	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// TargetExists used to check if the repo has already been created or not
func (g *Git) TargetExists() bool {
	if _, err := os.Stat(g.Path()); err == nil {
		return true
	}
	return false
}

// Clone clones the Git repository into its given or computed directory.
func (g *Git) Clone() error {
	logging.Debug("Attempting to clone %+v", g)
	path := g.Path()
	print.Info(locale.T("info_state_activate_uri", map[string]interface{}{
		"URI": g.URI(), "Dir": path,
	}))

	cmd := exec.Command("git", "clone", g.URI(), path)
	fmt.Println(strings.Join(cmd.Args, " ")) // match command output style
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// Computes the 'humanish' part of the source repository in order to use it as
// the directory to clone into if it wasn't explicitly given.
// This computation is based on git clone's shell script.
func (g *Git) humanishPart() string {
	re := regexp.MustCompile(":*[/\\\\]*\\.git$")
	path := re.ReplaceAllString(strings.TrimRight(g.URI(), "/"), "")
	re = regexp.MustCompile(".*[/\\\\:]")
	return re.ReplaceAllString(path, "")
}
