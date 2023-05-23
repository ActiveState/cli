package buildscript

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	file, err := get(filepath.Join("testdata", "basic.lo"))
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
	file, err := get(filepath.Join("testdata", "complex.lo"))
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
	file, err := get(filepath.Join("testdata", "example.lo"))
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

func TestWrite(t *testing.T) {
	file, err := get(filepath.Join("testdata", "moderate.lo"))
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
	runtime`, file.Script.String())
}

func TestRoundTrip(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "buildscript-")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	file, err := get(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	script := file.Script

	tmpfile.Write([]byte(file.Script.String()))
	tmpfile.Close()

	file, err = get(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, script, file.Script)
}

func TestJson(t *testing.T) {
	file, err := get(filepath.Join("testdata", "moderate.lo"))
	require.NoError(t, err)
	expectedJson := &bytes.Buffer{}
	json.Compact(expectedJson, []byte(`{
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
	assert.Equal(t, string(expectedJson.Bytes()), string(file.Script.ToJson()))
}

func TestBuildExpression(t *testing.T) {
	file, err := get(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	expr, err := file.Script.ToBuildExpression()
	require.NoError(t, err)
	script := FromBuildExpression(expr)
	assert.Equal(t, script, file.Script)
	assert.Equal(t, string(script.ToJson()), string(file.Script.ToJson()))
	assert.True(t, file.Script.EqualsBuildExpression(expr))
}
