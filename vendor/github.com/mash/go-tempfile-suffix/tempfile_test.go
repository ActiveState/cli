// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tempfile

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestTempFile(t *testing.T) {
	f, err := TempFileWithSuffix("/_not_exists_", "foo", ".txt")
	if f != nil || err == nil {
		t.Errorf("TempFile(`/_not_exists_`, `foo`) = %v, %v", f, err)
	}

	dir := os.TempDir()
	f, err = TempFileWithSuffix(dir, "ioutil_test", ".txt")
	if f == nil || err != nil {
		t.Errorf("TempFile(dir, `ioutil_test`) = %v, %v", f, err)
	}
	if f != nil {
		f.Close()
		os.Remove(f.Name())
		re := regexp.MustCompile("^" + regexp.QuoteMeta(filepath.Join(dir, "ioutil_test")) + "[0-9]+\\.txt$")
		if !re.MatchString(f.Name()) {
			t.Errorf("TempFile(`"+dir+"`, `ioutil_test`) created bad name %s", f.Name())
		}
	}
}
