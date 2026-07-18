package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/sysinfo"
)

func mkSitePackages(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

func TestGlobSitePackages(t *testing.T) {
	// The expected layout is OS-specific (see sitePackagesGlob).
	layout := filepath.Join("usr", "lib", "python3.10", "site-packages")
	if sysinfo.OS() == sysinfo.Windows {
		layout = filepath.Join("Lib", "site-packages")
	}

	tests := []struct {
		name   string
		layout string // relative dir to create under the root, "" for none
		want   string // expected relative site-packages path, "" for no match
	}{
		{"matching layout", layout, layout},
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

func TestGlobSitePackagesDeduplicatesAcrossRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	rel := filepath.Join("usr", "lib", "python3.10", "site-packages")
	if sysinfo.OS() == sysinfo.Windows {
		rel = filepath.Join("Lib", "site-packages")
	}
	mkSitePackages(t, rootA, rel)
	mkSitePackages(t, rootB, rel)

	got := globSitePackages([]string{rootA, rootB})
	if len(got) != 1 || got[0] != rel {
		t.Fatalf("expected a single deduplicated [%s], got %v", rel, got)
	}
}
