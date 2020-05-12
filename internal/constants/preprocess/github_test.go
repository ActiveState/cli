package preprocess

import (
	"os"
	"testing"

	"github.com/google/go-github/v29/github"
)

func setupCircleEnv(t *testing.T) func() {
	t.Helper()
	return setupEnv(t, "CI_PULL_REQUEST", "https://github.com/ActiveState/cli/pull/123")
}

func setupAzureEnv(t *testing.T) func() {
	t.Helper()
	return setupEnv(t, "SYSTEM_PULLREQUEST_PULLREQUESTNUMBER", "123")
}

func setupEnv(t *testing.T, key string, value string) func() {
	t.Helper()
	prInfo := os.Getenv(key)
	if prInfo == "" {
		os.Setenv(key, value)
	}

	return func() {
		os.Unsetenv(key)
	}
}

func TestGetLabel(t *testing.T) {
	labelName := "version: minor"
	labels := []*github.Label{&github.Label{Name: &labelName}}

	if getLabel(labels) != labelName {
		t.Fatal("version string should be 'minor'")
	}
}
