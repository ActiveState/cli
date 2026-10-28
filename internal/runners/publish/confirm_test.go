package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"gopkg.in/yaml.v3"
)

// editorContentEnv makes the test binary act as $EDITOR: it writes the variable's value to the file it is given.
const editorContentEnv = "PUBLISH_TEST_EDITOR_CONTENT"

func TestMain(m *testing.M) {
	if content, ok := os.LookupEnv(editorContentEnv); ok && len(os.Args) == 2 {
		if err := os.WriteFile(os.Args[1], []byte(content), 0644); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// useTestEditor points $EDITOR at the test binary, which replaces the edited file with content.
func useTestEditor(t *testing.T, content string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("EDITOR", exe)
	t.Setenv(editorContentEnv, content)
}

// fakePrompter answers Confirm and Select from queues; any other prompt panics through the nil embedded Prompter.
type fakePrompter struct {
	prompt.Prompter
	confirms []bool
	selects  []int // index into the offered choices
	messages []string
}

func (f *fakePrompter) Confirm(_, message string, _, _ *bool) (bool, error) {
	f.messages = append(f.messages, message)
	if len(f.confirms) == 0 {
		return false, errs.New("unexpected confirmation: %s", message)
	}
	answer := f.confirms[0]
	f.confirms = f.confirms[1:]
	return answer, nil
}

func (f *fakePrompter) Select(_, message string, choices []string, _, _ *string) (string, error) {
	if len(f.selects) == 0 {
		return "", errs.New("unexpected selection: %s", message)
	}
	choice := choices[f.selects[0]]
	f.selects = f.selects[1:]
	return choice, nil
}

func (f *fakePrompter) Input(_, _ string, _, _ *string, _ ...prompt.ValidatorFlag) (string, error) {
	return "", nil // "Press enter when done editing"
}

const (
	choiceEdit = iota
	choiceSave
	choiceCancel
)

func newRequest() *request.PublishVariables {
	return &request.PublishVariables{
		Name:      "mypkg",
		Namespace: "private/org/python",
		Version:   "1.0.0",
		Dependencies: []request.PublishVariableDep{{
			Dependency: request.Dependency{Name: "requests", Namespace: "language/python", Type: request.DependencyTypeRuntime},
			Conditions: []request.Dependency{{Name: "Windows", Namespace: "kernel"}},
		}},
	}
}

func TestConfirmRequestFillsVersionRequirements(t *testing.T) {
	p := &fakePrompter{confirms: []bool{true}}
	r := &Runner{out: outputhelper.NewCatcher(), prompt: p}
	reqVars := newRequest()

	publish, err := r.confirmRequest(reqVars, &Params{}, false)
	if err != nil || !publish {
		t.Fatalf("publish = %v, err = %v", publish, err)
	}
	dep := reqVars.Dependencies[0]
	if dep.VersionRequirements != ">=0" || dep.Conditions[0].VersionRequirements != ">=0" {
		t.Errorf("version requirements not filled in: %+v", dep)
	}
	if !strings.Contains(p.messages[0], "versionRequirements: '>=0'") {
		t.Errorf("confirmation does not show the filled in requirements:\n%s", p.messages[0])
	}
}

func TestConfirmRequestEditor(t *testing.T) {
	t.Run("removed fields are removed", func(t *testing.T) {
		useTestEditor(t, "name: mypkg\nnamespace: private/org/python\nversion: 1.0.0\n")
		p := &fakePrompter{confirms: []bool{false, true}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p}
		reqVars := newRequest()

		publish, err := r.confirmRequest(reqVars, &Params{}, false)
		if err != nil || !publish {
			t.Fatalf("publish = %v, err = %v", publish, err)
		}
		if len(reqVars.Dependencies) != 0 {
			t.Errorf("dependencies removed in the editor were kept: %+v", reqVars.Dependencies)
		}
	})

	t.Run("namespace change is kept", func(t *testing.T) {
		useTestEditor(t, "name: mypkg\nnamespace: private/org/other\nversion: 1.0.0\n")
		p := &fakePrompter{confirms: []bool{false, true}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p, versionExists: noVersions}
		reqVars := newRequest()

		publish, err := r.confirmRequest(reqVars, &Params{}, false)
		if err != nil || !publish {
			t.Fatalf("publish = %v, err = %v", publish, err)
		}
		if reqVars.Namespace != "private/org/other" {
			t.Errorf("namespace = %q, want the edited one", reqVars.Namespace)
		}
	})

	for name, tt := range map[string]struct {
		namespace string
		params    Params
		exists    bool
		reason    string
	}{
		"cleared namespace is put back":          {"", Params{}, false, "namespace is required"},
		"namespace with the version is put back": {"private/org/other", Params{}, true, "already exists"},
		"namespace outside the org is put back":  {"private/other/python", Params{Build: "src"}, false, "requires a namespace under"},
	} {
		t.Run(name, func(t *testing.T) {
			useTestEditor(t, "name: mypkg\nnamespace: "+tt.namespace+"\nversion: 1.0.0\n")
			out := outputhelper.NewCatcher()
			p := &fakePrompter{confirms: []bool{false, true}, selects: []int{choiceEdit}}
			r := &Runner{out: out, prompt: p, versionExists: func(_, _, _ string) (bool, error) { return tt.exists, nil }}
			reqVars := newRequest()

			publish, err := r.confirmRequest(reqVars, &tt.params, false)
			if err != nil || !publish {
				t.Fatalf("publish = %v, err = %v", publish, err)
			}
			if reqVars.Namespace != "private/org/python" {
				t.Errorf("namespace = %q, want the original one put back", reqVars.Namespace)
			}
			if !strings.Contains(out.CombinedOutput(), tt.reason) {
				t.Errorf("output does not explain why (%q):\n%s", tt.reason, out.CombinedOutput())
			}
		})
	}

	t.Run("name change with --build is saved instead", func(t *testing.T) {
		t.Chdir(t.TempDir())
		useTestEditor(t, "name: otherpkg\nnamespace: private/org/python\nversion: 1.0.0\n")
		p := &fakePrompter{confirms: []bool{false}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p}

		_, err := r.confirmRequest(newRequest(), &Params{Build: "src"}, false)
		if err == nil || !strings.Contains(err.Error(), "The name can't be changed") {
			t.Fatalf("err = %v, want the name named as fixed", err)
		}
		if !strings.Contains(p.messages[0], "name and version are built into the package") {
			t.Errorf("confirmation does not say up front what is fixed:\n%s", p.messages[0])
		}
		if _, err := os.Stat("otherpkg-1.0.0.meta.yaml"); err != nil {
			t.Errorf("edits were not saved: %v", err)
		}
	})

	t.Run("namespace change with --edit is saved instead", func(t *testing.T) {
		t.Chdir(t.TempDir())
		useTestEditor(t, "name: mypkg\nnamespace: private/org/other\nversion: 1.0.0\n")
		p := &fakePrompter{confirms: []bool{false}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p}

		_, err := r.confirmRequest(newRequest(), &Params{Edit: true}, true)
		if err == nil || !strings.Contains(err.Error(), "The namespace can't be changed") {
			t.Errorf("err = %v, want the namespace named as fixed", err)
		}
	})

	t.Run("description added on a revision is rejected", func(t *testing.T) {
		useTestEditor(t, "name: mypkg\nnamespace: private/org/python\nversion: 1.0.0\ndescription: changed\n")
		p := &fakePrompter{confirms: []bool{false}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p}

		_, err := r.confirmRequest(newRequest(), &Params{Edit: true}, true)
		if err == nil || !strings.Contains(err.Error(), "cannot be changed") {
			t.Errorf("err = %v, want the identity error", err)
		}
	})

	t.Run("private-builder is restored with --build", func(t *testing.T) {
		useTestEditor(t, "name: mypkg\nnamespace: private/org/python\nversion: 1.0.0\n")
		p := &fakePrompter{confirms: []bool{false, true}, selects: []int{choiceEdit}}
		r := &Runner{out: outputhelper.NewCatcher(), prompt: p}
		reqVars := newRequest()

		if _, err := r.confirmRequest(reqVars, &Params{Build: "src"}, false); err != nil {
			t.Fatal(err)
		}
		if len(reqVars.Dependencies) != 1 || reqVars.Dependencies[0].Name != privateBuilderName {
			t.Errorf("dependencies = %+v, want only the private-builder", reqVars.Dependencies)
		}
	})
}

// noVersions is a versionExists for a platform where nothing is published yet.
func noVersions(_, _, _ string) (bool, error) { return false, nil }

func TestDefaultNamespace(t *testing.T) {
	withPyproject := t.TempDir()
	if err := os.WriteFile(filepath.Join(withPyproject, "pyproject.toml"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ buildDir, want string }{
		{"", "private/org"},
		{t.TempDir(), "private/org"},
		{withPyproject, "private/org/python"},
	} {
		if got := defaultNamespace("org", tt.buildDir); got != tt.want {
			t.Errorf("defaultNamespace(%q) = %q, want %q", tt.buildDir, got, tt.want)
		}
	}
}

func TestConfirmRequestSave(t *testing.T) {
	t.Chdir(t.TempDir())
	p := &fakePrompter{confirms: []bool{false}, selects: []int{choiceSave}}
	r := &Runner{out: outputhelper.NewCatcher(), prompt: p}

	publish, err := r.confirmRequest(newRequest(), &Params{Build: "src"}, false)
	if err != nil || publish {
		t.Fatalf("publish = %v, err = %v, want saved", publish, err)
	}

	b, err := os.ReadFile(filepath.Join(".", "mypkg-1.0.0.meta.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "#") {
		t.Errorf("saved meta file contains comments:\n%s", b)
	}
	var saved request.PublishVariables
	if err := yaml.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Name != "mypkg" || saved.Version != "1.0.0" || len(saved.Dependencies) != 2 ||
		saved.Dependencies[0].VersionRequirements != ">=0" {
		t.Errorf("saved meta file is incomplete: %+v", saved)
	}
}

func TestConfirmRequestCancel(t *testing.T) {
	p := &fakePrompter{confirms: []bool{false}, selects: []int{choiceCancel}}
	r := &Runner{out: outputhelper.NewCatcher(), prompt: p}

	if _, err := r.confirmRequest(newRequest(), &Params{}, false); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("err = %v, want cancelled", err)
	}
}

func TestPublishFailedError(t *testing.T) {
	msg := publishFailedError(errs.New("API responded with error: File to upload was not provided")).Error()
	if !strings.Contains(msg, "File to upload was not provided") || strings.Contains(msg, "<no value>") {
		t.Errorf("message does not include the API error:\n%s", msg)
	}
}

func TestMetaFromIngredient(t *testing.T) {
	ingredient := &ParentIngredient{Version: "1.0.0", Dependencies: windowsOnlyDependency()}

	meta, err := metaFromIngredient("private/org/python", "mypkg", ingredient)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Dependencies) != 1 || len(meta.Dependencies[0].Conditions) != 1 {
		t.Fatalf("dependencies = %+v", meta.Dependencies)
	}
	if got := meta.Dependencies[0].Conditions[0].VersionRequirements; got != ">=0" {
		t.Errorf("condition without requirements exported as %q, want >=0", got)
	}
}

// windowsOnlyDependency is a revision's dependency on requests, conditioned on Windows without a version requirement.
func windowsOnlyDependency() []inventory_models.Dependency {
	return []inventory_models.Dependency{{
		Feature:   ptr.To("requests"),
		Namespace: ptr.To("language/python"),
		Type:      inventory_models.DependencyTypeRuntime,
		Conditions: []*inventory_models.DependencyCondition{{Condition: inventory_models.Condition{
			Feature:   ptr.To("Windows"),
			Namespace: ptr.To("kernel"),
		}}},
	}}
}
