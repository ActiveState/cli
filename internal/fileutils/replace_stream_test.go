package fileutils

import (
	"bytes"
	"fmt"
	"testing"
)

func TestStream(t *testing.T) {
	byts := []byte("123abc/def/ghi/bin/python\x00456abc/def/ghi/bin/perl\x00other")
	oldPath := "abc/def/ghi"
	newPath := "def/ghi"
	_, res, err := ReplaceNulTerminatedPathStream(bytes.NewReader(byts), oldPath, newPath)
	if err != nil {
		t.Errorf("Received error: %v", err)
	}
	if len(res) != len(byts) {
		t.Errorf("Expected len = %d, got = %d", len(byts), len(res))
	}
	fmt.Printf("%d, %d, %q\n", len(res), len(byts), res)
}
