package client

import "testing"

// TestDefaultIntegration is an integration test, that checks that the fields
// received from the backend are of the correct format type.
// Note:  This is just a suggestion.  We may not want to do that...
func TestDefaultIntegration(t *testing.T) {
	d := NewDefault()
	d.Solve()
	// ...
	// d.Build()
	// d.BuildLog()
}
