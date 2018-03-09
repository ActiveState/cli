package artifact

// Artifact is used to describe the contents of an artifact, this information can then be used to set up distributions
// An artifact can be a package, a language, a clib, etc. It is a component that makes up a distribution.
// It reflects the spec described here https://docs.google.com/document/d/1HprLsYXiKBeKfUvRrXpgyD_aodMnf6ZyuwgqpZu5ii4
type Artifact struct {
	Name     string
	Type     string
	Version  string
	Relocate string
	Binaries []string
}
