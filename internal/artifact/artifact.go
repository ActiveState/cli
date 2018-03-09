package artifact

// Artifact reflects the spec described here https://docs.google.com/document/d/1HprLsYXiKBeKfUvRrXpgyD_aodMnf6ZyuwgqpZu5ii4
type Artifact struct {
	Name     string
	Type     string
	Version  string
	Relocate string
	Binaries []string
}
