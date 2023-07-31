package buildscript

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	script, err := NewScript([]byte(
		`let:
  runtime = solve(
    platforms = ["linux", "windows"],
    requirements = [
      {
        name = "python",
        namespace = "language",
      },
      {
        name = "requests",
        namespace = "language/python",
        version_requirements = [
          {
            comparator = "eq",
            version = "3.10.10"
          }
        ]
      }
    ]
  )
in:
  runtime
`))
	require.NoError(t, err)

	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: ptr.To(`"linux"`)},
								&Value{Str: ptr.To(`"windows"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"python"`)}},
									&Assignment{"namespace", &Value{Str: ptr.To(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"requests"`)}},
									&Assignment{"namespace", &Value{Str: ptr.To(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{Str: ptr.To(`"eq"`)}},
											&Assignment{"version", &Value{Str: ptr.To(`"3.10.10"`)}},
										}},
									}}},
								}},
							}},
						}},
					}},
				}},
			},
		},
		&In{Name: ptr.To("runtime")},
	}, script)
}

func TestComplex(t *testing.T) {
	script, err := NewScript([]byte(
		`let:
    linux_runtime = solve(
        requirements=[
            {
                name="language/python"
            }
        ],
        platforms=["67890"]
    )

    win_runtime = solve(
        requirements=[{
                name="language/perl"
            }
        ],
        platforms=["12345"]
    )

in:
   merge(
        win_installer(win_runtime),
        tar_installer(linux_runtime)
    )
`))
	require.NoError(t, err)

	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"linux_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"language/python"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: ptr.To(`"67890"`)}},
							},
						}},
					}},
				}},
				&Assignment{"win_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"language/perl"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: ptr.To(`"12345"`)}},
							},
						}},
					}},
				}},
			},
		},
		&In{FuncCall: &FuncCall{"merge", []*Value{
			&Value{FuncCall: &FuncCall{"win_installer", []*Value{&Value{Ident: ptr.To("win_runtime")}}}},
			&Value{FuncCall: &FuncCall{"tar_installer", []*Value{&Value{Ident: ptr.To("linux_runtime")}}}},
		}}},
	}, script)
}

const example = `let:
  runtime = solve(
    at_time = "2023-04-27T17:30:05.999000Z",
    platforms = ["96b7e6f2-bebf-564c-bc1c-f04482398f38", "96b7e6f2-bebf-564c-bc1c-f04482398f38"],
    requirements = [
      {
        name = "python",
        namespace = "language",
      },
      {
        name = "requests",
        namespace = "language/python",
        version_requirements = [
          {
            comparator = "eq",
            version = "3.10.10"
          }
        ]
      }
    ]
  )
in:
  runtime`

func TestExample(t *testing.T) {
	script, err := NewScript([]byte(example))
	require.NoError(t, err)

	assert.Equal(t, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"at_time", &Value{Str: ptr.To(`"2023-04-27T17:30:05.999000Z"`)},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
								&Value{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"python"`)}},
									&Assignment{"namespace", &Value{Str: ptr.To(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{Str: ptr.To(`"requests"`)}},
									&Assignment{"namespace", &Value{Str: ptr.To(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{Str: ptr.To(`"eq"`)}},
											&Assignment{"version", &Value{Str: ptr.To(`"3.10.10"`)}},
										}},
									}}},
								}},
							}},
						}},
					}},
				}},
			},
		},
		&In{Name: ptr.To("runtime")},
	}, script)
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

	script, err := NewScript([]byte(example))
	require.NoError(t, err)

	tmpfile.Write([]byte(script.String()))
	tmpfile.Close()

	roundTripScript, err := newScriptFromFile(tmpfile.Name(), nil)
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
}

func TestJson(t *testing.T) {
	script, err := NewScript([]byte(
		`let:
    runtime = solve(
        requirements=[
            {
                name="language/python"
            }
        ],
        platforms=["12345", "67890"]
    )
in:
    runtime
`))
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

	actualJson, err := json.Marshal(script)
	require.NoError(t, err)
	assert.Equal(t, string(expectedJson), string(actualJson))
}

func TestBuildExpression(t *testing.T) {
	expr, err := buildexpression.New([]byte(`{
  "let": {
    "runtime": {
      "solve_legacy": {
        "at_time": "2023-04-27T17:30:05.999000Z",
        "build_flags": [],
        "camel_flags": [],
        "platforms": [
          "96b7e6f2-bebf-564c-bc1c-f04482398f38"
        ],
        "requirements": [
          {
            "name": "jinja2-time",
            "namespace": "language/python"
          },
          {
            "name": "jupyter-contrib-nbextensions",
            "namespace": "language/python"
          },
          {
            "name": "python",
            "namespace": "language",
            "version_requirements": [
              {
                "comparator": "eq",
                "version": "3.10.10"
              }
            ]
          },
          {
            "name": "copier",
            "namespace": "language/python"
          },
          {
            "name": "jupyterlab",
            "namespace": "language/python"
          }
        ],
        "solver_version": null
      }
    },
    "in": "$runtime"
  }
}`))
	require.NoError(t, err)

	// Verify conversions between buildscripts and buildexpressions is accurate.
	script, err := NewScriptFromBuildExpression(expr)
	require.NoError(t, err)
	require.NotNil(t, script)
	newExpr, err := script.ToBuildExpression()
	require.NoError(t, err)
	exprBytes, err := json.Marshal(expr)
	require.NoError(t, err)
	newExprBytes, err := json.Marshal(newExpr)
	require.NoError(t, err)
	assert.Equal(t, string(exprBytes), string(newExprBytes))

	// Verify comparisons between buildscripts and buildexpressions is accurate.
	assert.True(t, script.EqualsBuildExpression(expr))
	assert.True(t, script.EqualsBuildExpressionBytes(exprBytes))

	// Verify null JSON value is handled correctly.
	var null *string
	nullHandled := false
	for _, assignment := range script.Let.Assignments {
		if assignment.Key == "runtime" {
			args := assignment.Value.FuncCall.Arguments
			require.NotNil(t, args)
			for _, arg := range args {
				if arg.Assignment != nil && arg.Assignment.Key == "solver_version" {
					assert.Equal(t, null, arg.Assignment.Value.Str)
					nullHandled = true
				}
			}
		}
	}
	assert.True(t, nullHandled, "JSON null not encountered")
}
