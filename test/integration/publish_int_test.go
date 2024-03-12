package integration

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

var editorFileRx = regexp.MustCompile(`file:\s*?(.*?)\.\s`)

type PublishIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PublishIntegrationTestSuite) TestPublish() {
	suite.OnlyRunForTags(tagsuite.Publish)

	// For development convenience, should not be committed without commenting out..
	os.Setenv(constants.APIHostEnvVarName, "pr13361.activestate.build")
	os.Setenv(constants.DebugServiceRequestsEnvVarName, "true")

	if v := os.Getenv(constants.APIHostEnvVarName); v == "" || v == constants.DefaultAPIHost {
		suite.T().Skipf("Skipping test as %s is not set, this test can only be run against non-production envs.", constants.APIHostEnvVarName)
		return
	}

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

	ts.Env = append(ts.Env,
		// Publish tests shouldn't run against staging as they pollute the inventory db and artifact cache
		constants.APIHostEnvVarName+"="+os.Getenv(constants.APIHostEnvVarName),
	)

	user := ts.CreateNewUser()

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
			fmt.Sprintf("org/%s", user.Username),
			"2.3.4",
			[]invocation{
				{
					input{
						[]string{
							tempFile,
							"--name", "im-a-name-test1",
							"--namespace", "org/{{.Username}}",
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
							`Publish following ingredient?`,
							`name: im-a-name-test1`,
							`namespace: org/{{.Username}}`,
							`version: 2.3.4`,
							`description: im-a-description`,
							`name: author-name`,
							`email: author-email@domain.tld`,
						},
						"",
						false,
						0,
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
				},
			},
			},
		},
		{
			"New ingredient with meta file",
			"im-a-name-test2",
			fmt.Sprintf("org/%s", user.Username),
			"2.3.4",
			[]invocation{{
				input{
					[]string{"--meta", "{{.MetaFile}}", tempFile},
					ptr.To(`
name: im-a-name-test2
namespace: org/{{.Username}}
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
						`Publish following ingredient?`,
						`name: im-a-name-test2`,
						`namespace: org/{{.Username}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name`,
						`email: author-email@domain.tld`,
					},
					"",
					false,
					0,
				},
			}},
		},
		{
			"New ingredient with meta file and flags",
			"im-a-name-from-flag",
			fmt.Sprintf("org/%s", user.Username),
			"2.3.4",
			[]invocation{{
				input{
					[]string{"--meta", "{{.MetaFile}}", tempFile, "--name", "im-a-name-from-flag", "--author", "author-name-from-flag <author-email-from-flag@domain.tld>"},
					ptr.To(`
name: im-a-name
namespace: org/{{.Username}}
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
						`Publish following ingredient?`,
						`name: im-a-name-from-flag`,
						`namespace: org/{{.Username}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name-from-flag`,
						`email: author-email-from-flag@domain.tld`,
					},
					"",
					false,
					0,
				},
			}},
		},
		{
			"New ingredient with editor flag",
			"im-a-name-test3",
			fmt.Sprintf("org/%s", user.Username),
			"2.3.4",
			[]invocation{{
				input{
					[]string{tempFile, "--editor"},
					nil,
					ptr.To(`
name: im-a-name-test3
namespace: org/{{.Username}}
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
						`Publish following ingredient?`,
						`name: im-a-name-test3`,
						`namespace: org/{{.Username}}`,
						`version: 2.3.4`,
						`description: im-a-description`,
						`name: author-name`,
						`email: author-email@domain.tld`,
					},
					"",
					false,
					0,
				},
			}},
		},
		{
			"Cancel Publish",
			"bogus",
			fmt.Sprintf("org/%s", user.Username),
			"2.3.4",
			[]invocation{{
				input{
					[]string{tempFile, "--name", "bogus", "--namespace", "org/{{.Username}}"},
					nil,
					nil,
					false,
				},
				expect{
					[]string{`name: bogus`},
					"",
					false,
					0,
				},
			}},
		},
		{
			"Edit ingredient without file arg and with flags",
			"editable",
			fmt.Sprintf("org/%s", user.Username),
			"1.0.1",
			[]invocation{
				{ // Create ingredient
					input{
						[]string{tempFile,
							"--name", "editable",
							"--namespace", "org/{{.Username}}",
							"--version", "1.0.0",
						},
						nil,
						nil,
						true,
					},
					expect{
						[]string{
							`Publish following ingredient?`,
							`name: editable`,
						},
						"",
						false,
						0,
					},
				},
				{ // Edit ingredient
					input{
						[]string{
							tempFile,
							"--edit",
							"--name", "editable",
							"--namespace", "org/{{.Username}}",
							"--version", "1.0.1",
							"--author", "author-name-edited <author-email-edited@domain.tld>",
						},
						nil,
						nil,
						true,
					},
					expect{
						[]string{
							`Publish following ingredient?`,
							`name: editable`,
							`namespace: org/{{.Username}}`,
							`version: 1.0.1`,
							`name: author-name-edited`,
							`email: author-email-edited@domain.tld`,
						},
						"",
						false,
						0,
					},
				},
				{ // description editing not supported
					input{
						[]string{
							"--edit",
							"--name", "editable",
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
					},
				},
			},
		},
	}
	for n, tt := range tests {
		suite.Run(tt.name, func() {
			templateVars := map[string]interface{}{
				"Username": user.Username,
				"Email":    user.Email,
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

					if inv.input.confirmUpload {
						cp.SendLine("Y")
					} else {
						cp.SendLine("n")
						cp.Expect("Publish cancelled")
						return
					}

					cp.Expect("Successfully published")
					cp.Expect("Name: " + tt.ingredientName)
					cp.Expect("Namespace: " + tt.ingredientNamespace)
					cp.Expect("Version: " + tt.ingredientVersion)
					cp.ExpectExitCode(inv.expect.exitCode)

					cp = ts.Spawn("search", tt.ingredientNamespace+"/"+tt.ingredientName, "--ts=now")
					cp.Expect(tt.ingredientVersion)
					time.Sleep(time.Second)
					cp.Send("q")
					cp.ExpectExitCode(0)
				})
			}
		})
	}
}

func TestPublishIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PublishIntegrationTestSuite))
}
