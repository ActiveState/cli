package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func mkSitePackages(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

func TestGlobSitePackages(t *testing.T) {
	tests := []struct {
		name   string
		layout string // relative dir to create under the root, "" for none
		want   string // expected relative site-packages path, "" for no match
	}{
		{"linux versioned", filepath.Join("lib", "python3.10", "site-packages"), filepath.Join("lib", "python3.10", "site-packages")},
		{"linux lib64", filepath.Join("lib64", "python3.11", "site-packages"), filepath.Join("lib64", "python3.11", "site-packages")},
		{"macos unversioned", filepath.Join("lib", "site-packages"), filepath.Join("lib", "site-packages")},
		{"usr prefixed", filepath.Join("usr", "lib", "python3.9", "site-packages"), filepath.Join("usr", "lib", "python3.9", "site-packages")},
		{"windows", filepath.Join("Lib", "site-packages"), filepath.Join("Lib", "site-packages")},
		{"no site-packages", filepath.Join("bin"), ""},
		{"unexpected layout not globbed", filepath.Join("opt", "python", "site-packages"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.layout != "" {
				mkSitePackages(t, root, tt.layout)
			}
			got := globSitePackages([]string{root})
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("expected no candidates, got %v", got)
				}
				return
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("expected [%s], got %v", tt.want, got)
			}
		})
	}
}

func TestWalkSitePackagesFallback(t *testing.T) {
	root := t.TempDir()
	// A layout the globs do not cover, so only the walk finds it.
	mkSitePackages(t, root, filepath.Join("opt", "python", "site-packages"))

	if got := globSitePackages([]string{root}); len(got) != 0 {
		t.Fatalf("globs should not match unexpected layout, got %v", got)
	}

	got, err := walkSitePackages([]string{root})
	if err != nil {
		t.Fatalf("walkSitePackages: %v", err)
	}
	want := filepath.Join("opt", "python", "site-packages")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("expected [%s], got %v", want, got)
	}
}

func TestGlobSitePackagesDeduplicatesAcrossRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	rel := filepath.Join("lib", "python3.10", "site-packages")
	mkSitePackages(t, rootA, rel)
	mkSitePackages(t, rootB, rel)

	got := globSitePackages([]string{rootA, rootB})
	if len(got) != 1 || got[0] != rel {
		t.Fatalf("expected a single deduplicated [%s], got %v", rel, got)
	}
}
