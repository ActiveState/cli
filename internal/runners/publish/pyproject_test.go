package publish

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
)

func TestParsePythonRequirement(t *testing.T) {
	valid := []struct {
		in, name, versionReqs string
	}{
		{"requests", "requests", ""},
		{"requests>=2.0", "requests", ">=2.0"},
		{"requests >= 2.0, < 3", "requests", ">=2.0,<3"},
		{"requests[security,socks]>=2", "requests", ">=2"},
		{"Typing_Extensions (>=4.0)", "Typing_Extensions", ">=4.0"},
		{"django==4.2.7", "django", "==4.2.7"},
		{"django==4.2.*", "django", ">=4.2,<4.3"},
		{"numpy~=1.24", "numpy", ">=1.24,<2"},
		{"numpy~=1.24.2", "numpy", ">=1.24.2,<1.25"},
	}
	for _, tt := range valid {
		req, err := parsePythonRequirement(tt.in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.in, err)
			continue
		}
		if req.Name != tt.name || req.VersionRequirements != tt.versionReqs {
			t.Errorf("%q: got (%q, %q), want (%q, %q)", tt.in, req.Name, req.VersionRequirements, tt.name, tt.versionReqs)
		}
	}

	invalid := []string{
		`colorama; sys_platform == "win32" or sys_platform == "cygwin"`,
		`colorama; (sys_platform == "win32")`,
		`pywin32; sys_platform != "win32"`,
		`uvloop; platform_machine == "x86_64"`,
		`pytest; extra == "test"`,
		`tomli; python_version != "3.10"`,
		"mypkg @ https://example.com/mypkg-1.0.tar.gz",
		"requests!=2.1",
		"requests===2.1",
		"numpy~=1",
		"django==4.x.*",
		">=1.0",
	}
	for _, in := range invalid {
		if req, err := parsePythonRequirement(in); err == nil {
			t.Errorf("%q: expected an error, got %+v", in, req)
		}
	}
}

func TestParsePythonRequirementMarkers(t *testing.T) {
	windows := request.Dependency{Name: "Windows", Namespace: "kernel", VersionRequirements: ">=0"}
	linux := request.Dependency{Name: "Linux", Namespace: "kernel", VersionRequirements: ">=0"}
	python := func(reqs string) request.Dependency {
		return request.Dependency{Name: "python", Namespace: "language", VersionRequirements: reqs}
	}

	tests := []struct {
		in   string
		want []request.Dependency
	}{
		{`pywin32>=300; sys_platform == "win32"`, []request.Dependency{windows}},
		{`colorama ; platform_system == 'Windows'`, []request.Dependency{windows}},
		{`pywin32; os_name == "nt"`, []request.Dependency{windows}},
		{`secretstorage; sys_platform == "linux"`, []request.Dependency{linux}},
		{`tomli; python_version < "3.11"`, []request.Dependency{python("<3.11")}},
		{`tomli; python_version >= "3.8"`, []request.Dependency{python(">=3.8")}},
		{`tomli; python_version > "3.9"`, []request.Dependency{python(">=3.10")}},
		{`tomli; python_version <= "3.9"`, []request.Dependency{python("<3.10")}},
		{`tomli; python_version == "3.9"`, []request.Dependency{python(">=3.9,<3.10")}},
		{`tomli; python_full_version > "3.9.1"`, []request.Dependency{python(">3.9.1")}},
		{`tomli; python_version >= "3.8" and python_version < "3.11"`, []request.Dependency{python(">=3.8,<3.11")}},
		{`pywin32; sys_platform == "win32" and python_version > "3.9"`, []request.Dependency{windows, python(">=3.10")}},
	}
	for _, tt := range tests {
		req, err := parsePythonRequirement(tt.in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(req.Conditions, tt.want) {
			t.Errorf("%q: conditions = %+v, want %+v", tt.in, req.Conditions, tt.want)
		}
	}

	if req, err := parsePythonRequirement("six>=1"); err != nil || req.Conditions != nil {
		t.Errorf("unconditional requirement: got %+v, %v", req, err)
	}
}
