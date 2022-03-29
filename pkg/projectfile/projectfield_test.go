package projectfile

import (
	"reflect"
	"testing"
)

func TestProjectField(t *testing.T) {
	tests := []struct {
		name       string
		projectRaw string
		run        func(p *projectField)
		want       string
	}{
		{
			"Add Commit",
			`https://platform.activestate.com/org/project`,
			func(p *projectField) { p.SetCommit("906D66B1-8D89-483C-8E44-6A613B49BADD", false) },
			"https://platform.activestate.com/org/project?commitID=906D66B1-8D89-483C-8E44-6A613B49BADD",
		},
		{
			"Add Headless Commit",
			`https://platform.activestate.com/org/project`,
			func(p *projectField) { p.SetCommit("906D66B1-8D89-483C-8E44-6A613B49BADD", true) },
			"https://platform.activestate.com/commit/906D66B1-8D89-483C-8E44-6A613B49BADD",
		},
		{
			"Add Headless Commit, already has Branch",
			`https://platform.activestate.com/org/project?branch=main`,
			func(p *projectField) { p.SetCommit("906D66B1-8D89-483C-8E44-6A613B49BADD", true) },
			"https://platform.activestate.com/commit/906D66B1-8D89-483C-8E44-6A613B49BADD",
		},
		{
			"Add Headless Commit, already has Commit and Branch",
			`https://platform.activestate.com/org/project?commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246&branch=main`,
			func(p *projectField) { p.SetCommit("906D66B1-8D89-483C-8E44-6A613B49BADD", true) },
			"https://platform.activestate.com/commit/906D66B1-8D89-483C-8E44-6A613B49BADD",
		},
		{
			"Add Branch",
			`https://platform.activestate.com/org/project`,
			func(p *projectField) { p.SetBranch("main") },
			"https://platform.activestate.com/org/project?branch=main",
		},
		{
			"Add Branch, already has commit",
			`https://platform.activestate.com/org/project?commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246`,
			func(p *projectField) { p.SetBranch("main") },
			"https://platform.activestate.com/org/project?branch=main&commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246",
		},
		{
			"Set Namespace",
			`https://platform.activestate.com/org1/project1?commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246`,
			func(p *projectField) { p.SetNamespace("org2", "project2") },
			"https://platform.activestate.com/org2/project2?commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246",
		},
		{
			"Set Namespace, currently headless",
			`https://platform.activestate.com/commit/25B3DEDF-98E5-400B-8B78-CE12F990B246`,
			func(p *projectField) { p.SetNamespace("org", "project") },
			"https://platform.activestate.com/org/project?commitID=25B3DEDF-98E5-400B-8B78-CE12F990B246",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewProjectField()
			if err := pf.LoadProject(tt.projectRaw); err != nil {
				t.Fatalf("Loading data failed")
			}

			tt.run(pf)

			if got := pf.Marshal(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProjectField = %v, want %v", got, tt.want)
			}
		})
	}
}
