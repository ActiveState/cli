package runtime

var _ Assembler = &HybridRuntime{}

// HybridRuntime holds all the meta-data necessary to activate a runtime
// environment for a Hybrid build. It is currently leveraging the behavior of
// CamelRuntime.
type HybridRuntime struct {
	*CamelRuntime
}

// BuildEngine always returns Hybrid
func (hr *HybridRuntime) BuildEngine() BuildEngine {
	return Hybrid
}
