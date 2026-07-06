package artifactcrypto

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// allowedNonStdlibImports lists the non-stdlib packages this package may import
// directly.
var allowedNonStdlibImports = map[string]bool{
	"github.com/ActiveState/cli/internal/errs": true,
}

// TestImportsStdlibOnly checks that the non-test source imports only the standard
// library and the packages in allowedNonStdlibImports. A standard library import
// path's first segment contains no dot; a module path's does.
func TestImportsStdlibOnly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			first, _, _ := strings.Cut(path, "/")
			isStdlib := !strings.Contains(first, ".")
			if isStdlib || allowedNonStdlibImports[path] {
				continue
			}
			t.Errorf("%s imports %q, which is neither stdlib nor on the allowlist "+
				"(allowed non-stdlib: internal/errs only)", name, path)
		}
	}
	if checked == 0 {
		t.Fatal("no non-test source files were checked")
	}
}
