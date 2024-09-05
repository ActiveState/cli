package integration

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

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

	tempFile := fileutils.TempFilePathUnsafe("", "*.zip")
	defer os.Remove(tempFile)

	tempFileInvalid := fileutils.TempFilePathUnsafe("", "*.notzip")
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
					"Expected file extension to be either",
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

func TestPublishIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PublishIntegrationTestSuite))
}
