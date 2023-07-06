package integration

import (
	"os"
	"regexp"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
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
	// os.Setenv(constants.APIHostEnvVarName, "staging.activestate.build")

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
		confirmPrompt   []string
		immediateOutput string
		exitCode        int
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
		name   string
		input  input
		expect expect
	}{
		{
			"New ingredient with file arg and flags",
			input{
				[]string{tempFile,
					"--name", "im-a-name",
					"--namespace", "{{.Username}}/shared",
					"--version", "2.3.4",
					"--description", "im-a-description",
					"--author", "author-name <author-email@domain.tld>",
				},
				nil,
				nil,
				true,
			},
			expect{
				[]string{
					`name: im-a-name`,
					`namespace: {{.Username}}/shared`,
					`version: 2.3.4`,
					`description: im-a-description`,
					`name: author-name`,
					`email: author-email@domain.tld`,
				},
				"",
				0,
			},
		},
		{
			"New ingredient with invalid filename",
			input{
				[]string{tempFileInvalid},
				nil,
				nil,
				true,
			},
			expect{
				[]string{},
				"Expected file extension to be either",
				1,
			},
		},
		{
			"New ingredient with meta file",
			input{
				[]string{"--meta", "{{.MetaFile}}", tempFile},
				p.StrP(`
name: im-a-name
namespace: {{.Username}}/shared
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
					`name: im-a-name`,
					`namespace: {{.Username}}/shared`,
					`version: 2.3.4`,
					`description: im-a-description`,
					`name: author-name`,
					`email: author-email@domain.tld`,
				},
				"",
				0,
			},
		},
		{
			"New ingredient with meta file and flags",
			input{
				[]string{"--meta", "{{.MetaFile}}", tempFile, "--name", "im-a-name-from-flag", "--author", "author-name-from-flag <author-email-from-flag@domain.tld>"},
				p.StrP(`
name: im-a-name
namespace: {{.Username}}/shared
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
					`name: im-a-name-from-flag`,
					`namespace: {{.Username}}/shared`,
					`version: 2.3.4`,
					`description: im-a-description`,
					`name: author-name-from-flag`,
					`email: author-email-from-flag@domain.tld`,
				},
				"",
				0,
			},
		},
		{
			"New ingredient with editor flag",
			input{
				[]string{tempFile, "--editor"},
				nil,
				p.StrP(`
name: im-a-name
namespace: {{.Username}}/shared
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
					`name: im-a-name`,
					`namespace: {{.Username}}/shared`,
					`version: 2.3.4`,
					`description: im-a-description`,
					`name: author-name`,
					`email: author-email@domain.tld`,
				},
				"",
				0,
			},
		},
		{
			"Cancel upload",
			input{
				[]string{tempFile, "--name", "bogus", "--namespace", "{{.Username}}/shared"},
				nil,
				nil,
				false,
			},
			expect{
				[]string{`name: bogus`},
				"",
				0,
			},
		},
		// --edit tests are currently not addressed, tracked here: https://activestatef.atlassian.net/browse/DX-1944
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			templateVars := map[string]interface{}{
				"Username": user.Username,
				"Email":    user.Email,
			}

			if tt.input.metafile != nil {
				inputMetaParsed, err := strutils.ParseTemplate(*tt.input.metafile, templateVars, nil)
				suite.Require().NoError(err)
				metafile, err := fileutils.WriteTempFile("metafile.yaml", []byte(inputMetaParsed))
				suite.Require().NoError(err)
				templateVars["MetaFile"] = metafile
			}

			args := make([]string, len(tt.input.args))
			copy(args, tt.input.args)

			for k, v := range args {
				vp, err := strutils.ParseTemplate(v, templateVars, nil)
				suite.Require().NoError(err)
				args[k] = vp
			}

			cp := ts.SpawnWithOpts(
				e2e.WithArgs(append([]string{"publish"}, args...)...),
			)

			if tt.expect.immediateOutput != "" {
				cp.Expect(tt.expect.immediateOutput)
			}

			// Send custom input via --editor
			if tt.input.editorValue != nil {
				cp.Expect("Press enter when done editing")
				snapshot := cp.Snapshot()
				match := editorFileRx.FindSubmatch([]byte(snapshot))
				if len(match) != 2 {
					suite.Fail("Could not match rx in snapshot: %s", editorFileRx.String())
				}
				fpath := match[1]
				inputEditorValue, err := strutils.ParseTemplate(*tt.input.editorValue, templateVars, nil)
				suite.Require().NoError(err)
				suite.Require().NoError(fileutils.WriteFile(string(fpath), []byte(inputEditorValue)))
				cp.SendLine("")
			}

			cp.Expect("Upload following ingredient?")
			for _, value := range tt.expect.confirmPrompt {
				v, err := strutils.ParseTemplate(value, templateVars, nil)
				suite.Require().NoError(err)
				cp.Expect(v)
			}

			if tt.input.confirmUpload {
				cp.SendLine("Y")
			} else {
				cp.SendLine("n")
				cp.Expect("Upload cancelled")
			}

			cp.ExpectExitCode(tt.expect.exitCode)
		})
	}
}

func TestPublishIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PublishIntegrationTestSuite))
}
