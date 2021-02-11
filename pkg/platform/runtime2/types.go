package runtime

// BuildEngine describes the build engine that was used to build the runtime
type BuildEngine int

const (
	// UnknownEngine represents an engine unknown to the runtime.
	UnknownEngine BuildEngine = iota

	// Camel is the legacy build engine, that builds Active{Python,Perl,Tcl}
	// distributions
	Camel

	// Alternative is the new alternative build orchestration framework
	Alternative

	// Hybrid wraps Camel.
	Hybrid
)

// ArtifactID represents an artifact ID
type ArtifactID string
