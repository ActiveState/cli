// Package common comprises interfaces and shared code used by the
// implementations in the alternative and camel packages.
package common

// Runtimer is an interface for a locally installed runtime
//
// If the runtime is not installed on the machine yet, create a new
// `runtime.Setup` to set it up.
type Runtimer interface {
	Environ() (map[string]string, error)
	// ...
}
