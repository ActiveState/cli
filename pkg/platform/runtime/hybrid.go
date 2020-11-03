package runtime

var _ EnvGetter = &HybridRuntime{}
var _ Assembler = &HybridInstall{}

// HybridRuntime holds all the meta-data necessary to activate a runtime
// environment for a Hybrid build. It is currently leveraging the behavior of
// CamelRuntime.
type HybridRuntime struct {
	*CamelEnv
}

type HybridInstall struct {
	*CamelInstall
}

// BuildEngine always returns Hybrid
func (hr *HybridInstall) BuildEngine() BuildEngine {
	return Hybrid
}
