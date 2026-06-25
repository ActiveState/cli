package integration

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
)

var editorFileRx = regexp.MustCompile(`file:\s*?(.*?)\.\s`)

type PublishIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PublishIntegrationTestSuite) TestPublish() {
	suite.OnlyRunForTags(tagsuite.Publish)

	// For development convenience, should not be committed without commenting out..
	// os.Setenv(constants.APIHostEnvVarName, "pr13375.activestate.build")

	type input struct {
		args          []string
		metafile      *string
		editorValue   *string
		confirmUpload bool
	}

	type expect struct {
		confirmPrompt    []string
		immediateOutput  string
		exitBeforePrompt bool
		exitCode         int
		parseMeta        bool
	}

	type invocation struct {
		input  input
		expect expect
	}

	tempFile := fileutils.TempFilePath("", ".zip")
	suite.Require().NoError(fileutils.Touch(tempFile))
	defer os.Remove(tempFile)

	tempFileInvalid := fileutils.TempFilePath("", ".notzip")
	suite.Require().NoError(fileutils.Touch(tempFileInvalid))
	defer os.Remove(tempFileInvalid)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	if apiHost := os.Getenv(constants.APIHostEnvVarName); apiHost != "" {
		ts.Env = append(ts.Env, constants.APIHostEnvVarName+"="+apiHost)
	}

	ts.LoginAsPersistentUser()

	namespaceUUID, err := uuid.NewRandom()
	suite.Require().NoError(err, "unable generate new random UUID")
	namespace := "private/ActiveState-CLI-Testing/" + namespaceUUID.String()

	tests := []struct {
		name                string
		ingredientName      string
		ingredientNamespace string
		ingredientVersion   string
		invocations         []invocation
	}{
		{
			"New ingredient with file arg and flags",
			"im-a-name-test1",
			namespace,
			"2.3.4",
			[]invocation{
				{
					input{
						[]string{
							tempFile,
							"--name", "{{.Name}}",
							"--namespace", "{{.Namespace}}",
							"--version", "2.3.4",
							"--description", "im-a-description",
							"--author", "author-name <author-email@domain.tld>",
							"--depend", "language/python@>=3",
							"--depend", "builder/python-module-builder@>=0",
							"--depend-test", "language/python@>=3",
							"--depend-build", "language/python@>=3",
							"--depend-runtime", "language/python@>=3",
						},
						nil,
						nil,
						true,
					},
					expect{
						[]string{
							`name: {{.Name}}`,
							`namespace: {{.Namespace}}`,
							`version: 2.3.4`,
							`description: im-a-description`,
							`name: author-name`,
							`email: author-email@domain.tld`,
							`publish this ingredient?`,
						},
						"",
						false,
						0,
						false,
					},
				},
			},
		},
		{
			"New ingredient with invalid filename",
			"",
			"",
			"",
			[]invocation{{input{
				[]string{tempFileInvalid},
				nil,
				nil,
				true,
			},
				expect{
					[]string{},
					"Expected file extension:",
					true,
					1,
					true,
				},
			},
			},
		},
		{
			"New ingredient with meta file",
			"im-a-name-test2",
			namespace,
			"2.3.4",
			[]invocation{{
				input{
					[]string{"--meta", "{{.MetaFile}}", tempFile},
					ptr.To(`
name: {{.Name}}
namespace: {{.Namespace}}
version: 2.3.4
description: im-a-description
authors:
  - name: author-name
    email: author-email@domain.tld
`),
					nil,
					true,
				},
				expect{
					[]string{
						`name: {{.Name}}`,
						`namespace: {{.Namespace}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name`,
						`email: author-email@domain.tld`,
						`publish this ingredient?`,
					},
					"",
					false,
					0,
					true,
				},
			}},
		},
		{
			"New ingredient with meta file and flags",
			"im-a-name-from-flag",
			namespace,
			"2.3.4",
			[]invocation{{
				input{
					[]string{"--meta", "{{.MetaFile}}", tempFile, "--name", "{{.Name}}", "--author", "author-name-from-flag <author-email-from-flag@domain.tld>"},
					ptr.To(`
name: {{.Name}}
namespace: {{.Namespace}}
version: 2.3.4
description: im-a-description
authors:
  - name: author-name
    email: author-email@domain.tld
`),
					nil,
					true,
				},
				expect{
					[]string{
						`name: {{.Name}}`,
						`namespace: {{.Namespace}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name-from-flag`,
						`email: author-email-from-flag@domain.tld`,
						`publish this ingredient?`,
					},
					"",
					false,
					0,
					true,
				},
			}},
		},
		{
			"New ingredient with editor flag",
			"im-a-name-test3",
			namespace,
			"2.3.4",
			[]invocation{{
				input{
					[]string{tempFile, "--editor"},
					nil,
					ptr.To(`
name: {{.Name}}
namespace: {{.Namespace}}
version: 2.3.4
description: im-a-description
authors:
  - name: author-name
    email: author-email@domain.tld
`),
					true,
				},
				expect{
					[]string{
						`name: {{.Name}}`,
						`namespace: {{.Namespace}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name`,
						`email: author-email@domain.tld`,
						`publish this ingredient?`,
					},
					"",
					false,
					0,
					true,
				},
			}},
		},
		{
			"Cancel Publish",
			"bogus",
			namespace,
			"2.3.4",
			[]invocation{{
				input{
					[]string{tempFile, "--name", "{{.Name}}", "--namespace", "{{.Namespace}}"},
					nil,
					nil,
					false,
				},
				expect{
					[]string{`name: {{.Name}}`},
					"",
					false,
					0,
					true,
				},
			}},
		},
		{
			"Edit ingredient without file arg and with flags",
			"editable",
			namespace,
			"1.0.1",
			[]invocation{
				{ // Create ingredient
					input{
						[]string{tempFile,
							"--name", "{{.Name}}",
							"--namespace", "{{.Namespace}}",
							"--version", "1.0.0",
						},
						nil,
						nil,
						true,
					},
					expect{
						[]string{
							`name: {{.Name}}`,
							`publish this ingredient?`,
						},
						"",
						false,
						0,
						true,
					},
				},
				{ // Edit ingredient
					input{
						[]string{
							tempFile,
							"--edit",
							"--name", "{{.Name}}",
							"--namespace", "{{.Namespace}}",
							"--version", "1.0.1",
							"--author", "author-name-edited <author-email-edited@domain.tld>",
						},
						nil,
						nil,
						true,
					},
					expect{
						[]string{
							`name: {{.Name}}`,
							`namespace: {{.Namespace}}`,
							`version: 1.0.1`,
							`name: author-name-edited`,
							`email: author-email-edited@domain.tld`,
							`publish this ingredient?`,
						},
						"",
						false,
						0,
						true,
					},
				},
				{ // description editing not supported
					input{
						[]string{
							"--edit",
							"--name", "{{.Name}}",
							"--description", "foo",
						},
						nil,
						nil,
						false,
					},
					expect{
						[]string{
							`Editing an ingredient description is not yet supported`,
						},
						"",
						true,
						1,
						true,
					},
				},
			},
		},
	}
	for n, tt := range tests {
		suite.Run(tt.name, func() {
			templateVars := map[string]interface{}{
				"Name":      tt.ingredientName,
				"Namespace": tt.ingredientNamespace,
			}

			for _, inv := range tt.invocations {
				suite.Run(fmt.Sprintf("%s invocation %d", tt.name, n), func() {
					ts.T = suite.T() // This differs per subtest
					if inv.input.metafile != nil {
						inputMetaParsed, err := strutils.ParseTemplate(*inv.input.metafile, templateVars, nil)
						suite.Require().NoError(err)
						metafile, err := fileutils.WriteTempFile("metafile.yaml", []byte(inputMetaParsed))
						suite.Require().NoError(err)
						templateVars["MetaFile"] = metafile
					}

					args := make([]string, len(inv.input.args))
					copy(args, inv.input.args)

					for k, v := range args {
						vp, err := strutils.ParseTemplate(v, templateVars, nil)
						suite.Require().NoError(err)
						args[k] = vp
					}

					cp := ts.SpawnWithOpts(
						e2e.OptArgs(append([]string{"publish"}, args...)...),
					)

					if inv.expect.immediateOutput != "" {
						cp.Expect(inv.expect.immediateOutput)
					}

					// Send custom input via --editor
					if inv.input.editorValue != nil {
						cp.Expect("Press enter when done editing")
						snapshot := cp.Snapshot()
						match := editorFileRx.FindSubmatch([]byte(snapshot))
						if len(match) != 2 {
							suite.Fail("Could not match rx in snapshot: %s", editorFileRx.String())
						}
						fpath := match[1]
						inputEditorValue, err := strutils.ParseTemplate(*inv.input.editorValue, templateVars, nil)
						suite.Require().NoError(err)
						suite.Require().NoError(fileutils.WriteFile(string(fpath), []byte(inputEditorValue)))
						time.Sleep(100 * time.Millisecond) // wait for disk write to happen
						cp.SendLine("")
					}

					if inv.expect.exitBeforePrompt {
						cp.ExpectExitCode(inv.expect.exitCode)
						return
					}

					for _, value := range inv.expect.confirmPrompt {
						v, err := strutils.ParseTemplate(value, templateVars, nil)
						suite.Require().NoError(err)
						cp.Expect(v)
					}

					cp.Expect("Y/n")

					var (
						name      = tt.ingredientName
						namespace = tt.ingredientNamespace
						version   = tt.ingredientVersion
					)

					if inv.expect.parseMeta {
						snapshot := cp.Snapshot()
						rx := regexp.MustCompile(`(?s)Prepared the following ingredient:(.*)Do you want to publish this ingredient\?`)
						match := rx.FindSubmatch([]byte(snapshot))
						suite.Require().NotNil(match, fmt.Sprintf("Could not match '%s' against: %s", rx.String(), snapshot))

						meta := request.PublishVariables{}
						err := yaml.Unmarshal(match[1], &meta)
						if err == nil {
							name = meta.Name
							namespace = meta.Namespace
							version = meta.Version
						}
					}

					if inv.input.confirmUpload {
						cp.SendLine("Y")
					} else {
						cp.SendLine("n")
						cp.Expect("Publish cancelled")
						return
					}

					cp.Expect("Successfully published")
					cp.Expect("Name:")
					cp.Expect(name)
					cp.Expect("Namespace:")
					cp.Expect(namespace)
					cp.Expect("Version:")
					cp.Expect(version)
					cp.ExpectExitCode(inv.expect.exitCode)

					cp = ts.Spawn("search", namespace+"/"+name, "--ts=now")
					cp.Expect(version)
					time.Sleep(time.Second)
					cp.Send("q")
					cp.ExpectExitCode(0)
				})
			}
		})
	}

	ts.IgnoreLogErrors() // ignore intentional failures like omitted filename, cannot edit description, etc.
}

// TestPublishBuildEncrypted exercises the full encrypted private-ingredient round
// trip: `state publish --build` packs a pure-Python source tree into a wheel,
// encrypts it under the org key, and publishes it; then `state install` resolves,
// downloads, and decrypts it. We prove decryption succeeded by reading the
// decrypted wheel back out of the depot and finding a sentinel string that only
// our plaintext source contains.
//
// The org key is supplied to both the publish (encrypt) and install (decrypt)
// sides through the environment, so the test needs no HTTPS key service.
//
// The ingredient is published under a unique, random name. Published private
// ingredients cannot be deleted, so the name must never collide across runs.
func (suite *PublishIntegrationTestSuite) TestPublishBuildEncrypted() {
	suite.OnlyRunForTags(tagsuite.Publish)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	if apiHost := os.Getenv(constants.APIHostEnvVarName); apiHost != "" {
		ts.Env = append(ts.Env, constants.APIHostEnvVarName+"="+apiHost)
	}

	ts.LoginAsPersistentUser()

	// Supply the org key to publish (encrypt) and install (decrypt) via the
	// environment, avoiding an HTTPS key service. The contract is validated just
	// like one fetched from a real service, including its binding to this org.
	key := make([]byte, artifactcrypto.KeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	ts.Env = append(ts.Env,
		constants.PrivateIngredientKeyContractEnvVarName+"="+orgKeyContract(suite, key, e2e.PersistentUsername))

	// A pure-Python source tree carrying a unique sentinel. After install we read
	// the decrypted wheel back out of the depot and look for the sentinel —
	// ciphertext could never yield a valid wheel containing it.
	sentinel := "private-ingredient-sentinel-" + strutils.UUID().String()
	srcDir := filepath.Join(ts.Dirs.Work, "ingredient-src")
	suite.Require().NoError(os.MkdirAll(filepath.Join(srcDir, "greeting"), 0755))
	suite.Require().NoError(fileutils.WriteFile(
		filepath.Join(srcDir, "greeting", "__init__.py"),
		[]byte(fmt.Sprintf("print(%q)\n", sentinel)),
	))

	// Create a fresh project under the testing org. `state publish --build`
	// requires a project (to determine the org its key encrypts under), and the
	// publish namespace must live under that same org.
	projectName := strutils.UUID()
	projectNamespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, projectName)
	cp := ts.SpawnWithOpts(e2e.OptArgs("init", "--language", "python", projectNamespace, ts.Dirs.Work))
	cp.Expect("Initializing Project")
	cp.Expect("has been successfully initialized", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(e2e.PersistentUsername, projectName.String())

	// Build, encrypt, and publish the private ingredient under a unique name.
	ingredientName := strutils.UUID().String()
	ingredientNamespace := "private/" + e2e.PersistentUsername + "/language/python"
	cp = ts.SpawnWithOpts(e2e.OptArgs(
		"publish", "--non-interactive",
		"--build", srcDir,
		"--namespace", ingredientNamespace,
		"--name", ingredientName,
		"--version", "0.0.1",
	))
	cp.Expect("Successfully published")
	cp.ExpectExitCode(0)

	// Install the freshly published ingredient, forcing resolution at the current
	// timestamp so the new revision is picked up rather than a cached solve.
	cp = ts.SpawnWithOpts(e2e.OptArgs(
		"install", ingredientNamespace+":"+ingredientName, "--ts=now",
	))
	cp.Expect("All dependencies have been installed and verified", e2e.RuntimeBuildSourcingTimeoutOpt)
	cp.ExpectExitCode(1)

	// Decryption proof: the decrypted content must be present in the depot and
	// contain our sentinel. A failed decrypt would skip the artifact, leaving the
	// sentinel absent. This is inlined (rather than in a helper) so a failure
	// surfaces the spawned-command output and state logs the e2e harness dumps.
	// The search is deliberately broad — any wheel or file anywhere under the
	// depot — because the exact on-disk path depends on how the artifact is
	// packaged on install.
	depot := filepath.Join(ts.Dirs.Cache, "depot")

	// Log the depot's top two levels so a failure shows how the artifact landed
	// (e.g. an "install" dir with a wheel vs. a leftover "payload.enc").
	if entries, err := os.ReadDir(depot); err == nil {
		for _, e := range entries {
			suite.T().Logf("depot/%s", e.Name())
			if e.IsDir() {
				if sub, err := os.ReadDir(filepath.Join(depot, e.Name())); err == nil {
					for _, s := range sub {
						suite.T().Logf("depot/%s/%s", e.Name(), s.Name())
					}
				}
			}
		}
	} else {
		suite.T().Logf("could not read depot %s: %v", depot, err)
	}

	var wheels []string
	fileCount := 0
	found := false
	walkErr := filepath.WalkDir(depot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileCount++

		// A decrypted wheel is a zip; scan its entries for the sentinel.
		if strings.HasSuffix(d.Name(), ".whl") {
			wheels = append(wheels, path)
			zr, err := zip.OpenReader(path)
			if err != nil {
				suite.T().Logf("could not open wheel %s as zip: %v", path, err)
				return nil
			}
			defer zr.Close()
			for _, f := range zr.File {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				content, _ := io.ReadAll(rc)
				rc.Close()
				if strings.Contains(string(content), sentinel) {
					found = true
				}
			}
			return nil
		}

		// Otherwise scan the raw file, in case the payload was delivered unpacked
		// rather than as a wheel. Skip large files (the sentinel lives in a tiny
		// Python source file).
		if info, err := d.Info(); err != nil || info.Size() > 5<<20 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(content), sentinel) {
			found = true
		}
		return nil
	})
	suite.Require().NoError(walkErr, "could not walk depot %s", depot)
	suite.T().Logf("searched %d files under the depot; wheels found: %v", fileCount, wheels)
	suite.Require().True(found, "sentinel %q not found in the depot; the artifact was likely not decrypted", sentinel)
}

// orgKeyContract builds the org-key contract JSON the key service would serve for
// the given key and organization, for injection via the environment.
func orgKeyContract(suite *PublishIntegrationTestSuite, key []byte, org string) string {
	contract := map[string]string{
		"schema":      "activestate.pim.orgkey/v1",
		"org":         org,
		"key_id":      "integration-test-key",
		"algorithm":   "AES-256-GCM",
		"encoding":    "base64",
		"key":         base64.StdEncoding.EncodeToString(key),
		"fingerprint": artifactcrypto.Fingerprint(key),
	}
	b, err := json.Marshal(contract)
	suite.Require().NoError(err)
	return string(b)
}

func TestPublishIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PublishIntegrationTestSuite))
}
