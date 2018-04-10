package artefact

// Artefact is used to describe the contents of an artefact, this information can then be used to set up distributions
// An artefact can be a package, a language, a clib, etc. It is a component that makes up a distribution.
// It reflects the spec described here https://docs.google.com/document/d/1HprLsYXiKBeKfUvRrXpgyD_aodMnf6ZyuwgqpZu5ii4
type Artefact struct {
	Name     string
	Type     string
	Version  string
	Relocate string
	Binaries []string
}
