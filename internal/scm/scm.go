package scm

import "github.com/ActiveState/ActiveState-CLI/internal/scm/git"

// SCMer is the interface all known SCMs should implement.
type SCMer interface {
	ConfigFileExists() bool // whether or not the ActiveState config file exists
	SetPath(string)         // set the repo's path (usually for cloning into)
	Path() string           // the repo's path (automatically set after cloning)
	SetBranch(string)       // set the repo's branch to use
	Branch() string         // the repo's branch
	CheckoutBranch() error  // checkout the configured branch
	TargetExists() bool     // Check if the repo directory has already been create
	Clone() error           // clone the repo into the current directory
}

// New returns an SCMer to use for the given URI, or nil if no SCMer was found.
func New(uri string) SCMer {
	if git.IsGitURI(uri) {
		return &git.Git{URI: uri}
	} // TODO: other supported SCMs
	return nil
}
