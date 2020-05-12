package preprocess

import (
	"testing"

	"github.com/google/go-github/v29/github"
)

func TestGetLabel(t *testing.T) {
	labelName := "version: minor"
	labels := []*github.Label{&github.Label{Name: &labelName}}

	if getLabel(labels) != labelName {
		t.Fatal("version string should be 'minor'")
	}
}
