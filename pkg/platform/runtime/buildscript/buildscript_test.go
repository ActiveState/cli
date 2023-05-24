package buildscript

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	file, err := newFile(filepath.Join("testdata", "basic.lo"))
	require.NoError(t, err)
	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: p.StrP(`"linux"`)},
								&Value{Str: p.StrP(`"windows"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"python"`)}},
									&Assignment{"namespace", &Value{Str: p.StrP(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"requests"`)}},
									&Assignment{"namespace", &Value{Str: p.StrP(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{Str: p.StrP(`"eq"`)}},
											&Assignment{"version", &Value{Str: p.StrP(`"3.10.10"`)}},
										}},
									}}},
								}},
							}},
						}},
					}},
				}},
			},
		},
		&In{Name: p.StrP("runtime")},
	}, file.Script)
}

func TestComplex(t *testing.T) {
	file, err := newFile(filepath.Join("testdata", "complex.lo"))
	require.NoError(t, err)
	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"linux_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"language/python"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: p.StrP(`"67890"`)}},
							},
						}},
					}},
				}},
				&Assignment{"win_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"language/perl"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: p.StrP(`"12345"`)}},
							},
						}},
					}},
				}},
			},
		},
		&In{FuncCall: &FuncCall{"merge", []*Value{
			&Value{FuncCall: &FuncCall{"win_installer", []*Value{&Value{Ident: p.StrP("win_runtime")}}}},
			&Value{FuncCall: &FuncCall{"tar_installer", []*Value{&Value{Ident: p.StrP("linux_runtime")}}}},
		}}},
	}, file.Script)
}

func TestExample(t *testing.T) {
	file, err := newFile(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"at_time", &Value{Str: p.StrP(`"2023-04-27T17:30:05.999000Z"`)},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: p.StrP(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
								&Value{Str: p.StrP(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"python"`)}},
									&Assignment{"namespace", &Value{Str: p.StrP(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: p.StrP(`"requests"`)}},
									&Assignment{"namespace", &Value{Str: p.StrP(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{Str: p.StrP(`"eq"`)}},
											&Assignment{"version", &Value{Str: p.StrP(`"3.10.10"`)}},
										}},
									}}},
								}},
							}},
						}},
					}},
				}},
			},
		},
		&In{Name: p.StrP("runtime")},
	}, file.Script)
}

func TestString(t *testing.T) {
	script, err := NewScript([]byte(
		`let:
    runtime = solve(
        requirements=[{name="language/python"}],
        platforms=["12345", "67890"]
    )
in:
    runtime
`))
	require.NoError(t, err)
	assert.Equal(t,
		`let:
	runtime = solve(
		requirements = [
			{
				name = "language/python"
			}
		],
		platforms = [
			"12345",
			"67890"
		]
	)

in:
	runtime`, script.String())
}

func TestRoundTrip(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "buildscript-")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	file, err := newFile(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	script := file.Script

	tmpfile.Write([]byte(file.Script.String()))
	tmpfile.Close()

	file, err = newFile(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, script, file.Script)
}

func TestJson(t *testing.T) {
	file, err := newFile(filepath.Join("testdata", "moderate.lo"))
	require.NoError(t, err)

	inputJson := &bytes.Buffer{}
	json.Compact(inputJson, []byte(`{
    "let": {
      "runtime": {
        "solve": {
          "requirements": [
            {
              "name": "language/python"
            }
          ],
          "platforms": ["12345", "67890"]
        }
      },
      "in": "$runtime"
    }
  }`))
	// Cannot compare marshaled JSON directly with inputJson due to key sort order, so unmarshal and
	// remarshal before making the comparison. json.Marshal() produces the same key sort order.
	marshaledInput := make(map[string]interface{})
	err = json.Unmarshal(inputJson.Bytes(), &marshaledInput)
	require.NoError(t, err)
	expectedJson, err := json.Marshal(marshaledInput)

	actualJson, err := json.Marshal(file.Script)
	require.NoError(t, err)
	assert.Equal(t, string(expectedJson), string(actualJson))
}

func TestBuildExpression(t *testing.T) {
	expr := fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildexpression.json"))
	script, err := NewScriptFromBuildExpression(expr)
	require.NoError(t, err)
	require.NotNil(t, script)
	newExpr, err := json.Marshal(script)
	require.NoError(t, err)

	// Cannot compare expr and newExpr directly due to key sort order, whitespace discrepancies,
	// etc., so unmarshal and remarshal before the comparison. json.Marshal() produces the same key
	// sort order.
	marshaledInput := make(map[string]interface{})
	err = json.Unmarshal(expr, &marshaledInput)
	require.NoError(t, err)
	expectedExpr, err := json.Marshal(marshaledInput)
	assert.Equal(t, string(expectedExpr), string(newExpr))
	assert.True(t, script.EqualsBuildExpression(expr))

	// Verify null JSON value is handled correctly.
	nullHandled := false
	for _, assignment := range script.Let.Assignments {
		if assignment.Key == "runtime" {
			args := assignment.Value.FuncCall.Arguments
			require.NotNil(t, args)
			for _, arg := range args {
				if arg.Assignment != nil && arg.Assignment.Key == "solver_version" {
					assert.Equal(t, p.PstrP(nil), arg.Assignment.Value.Str)
					nullHandled = true
				}
			}
		}
	}
	assert.True(t, nullHandled, "JSON null not encountered")
}
