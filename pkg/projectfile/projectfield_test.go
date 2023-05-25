package projectfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectField(t *testing.T) {
	tests := []struct {
		name       string
		projectRaw string
		run        func(p *projectField)
		want       string
	}{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := NewProjectField()
			err := pf.LoadProject(tt.projectRaw)
			assert.NoError(t, err, "Loading data failed")

			tt.run(pf)

			assert.Equal(t, tt.want, pf.Marshal())
		})
	}
}
