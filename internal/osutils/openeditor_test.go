package osutils

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyEditor(t *testing.T) {
	expected := "debug"
	if runtime.GOOS == "windows" {
		expected = "debug.exe"
	}

	f, err := os.OpenFile(expected, os.O_CREATE|os.O_EXCL, 0700)
	require.NoError(t, err, "should be able to create executable file")
	defer os.Remove(f.Name())

	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	wd, err := Getwd()
	require.NoError(t, err, "could not get current working directory")

	err = os.Setenv("PATH", wd)
	require.NoError(t, err, "could not set PATH")

	err = verifyEditor(expected)
	require.NoError(t, err)
}

func TestVerifyPathEditor(t *testing.T) {
	expected := "debug"
	if runtime.GOOS == "windows" {
		expected = "debug.exe"
	}

	f, err := os.CreateTemp("", expected)
	require.NoError(t, err, "should be able to create executable file")
	defer os.Remove(f.Name())

	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	wd, err := Getwd()
	require.NoError(t, err, "could not get current working directory")

	err = os.Setenv("PATH", wd)
	require.NoError(t, err, "could not set PATH")

	err = verifyEditor(f.Name())
	require.NoError(t, err)
}

func TestVerifyEditor_NotInPath(t *testing.T) {
	executeable := "someExecutable"
	if runtime.GOOS == "windows" {
		executeable = "someExecutable.exe"
	}

	err := verifyEditor(executeable)
	require.Error(t, err, "should get failure when editor in path does not exist")
}

func TestEditor_NoExtension(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the test for file extensions is only relevant for Windows")
	}

	err := verifyEditor("someExecutable")
	require.Error(t, err, "should get failure when editor in path does not have extension")
}
