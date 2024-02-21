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

// toBuildExpression converts given script constructed by Participle into a buildexpression.
// This function should not be used to convert an arbitrary script to buildexpression.
// NewScript*() populates the expr field with the equivalent build expression.
// This function exists solely for testing that functionality.
func toBuildExpression(script *Script) (*buildexpression.BuildExpression, error) {
	bytes, err := json.Marshal(script)
	if err != nil {
		return nil, err
	}
	return buildexpression.New(bytes)
}

func TestBasic(t *testing.T) {
	script, err := NewScript([]byte(
		`runtime = solve(
	platforms = ["linux", "windows"],
	requirements = [
		Req(name="language/python"),
		Req(name="language/python/requests", version="3.10.10")
	]
)

main = runtime
`))
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`"linux"`)},
							{Str: ptr.To(`"windows"`)},
						}},
					}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python"`)},
									}},
								}}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python/requests"`)},
									}},
									{Assignment: &Assignment{
										"version", &Value{Str: ptr.To(`"3.10.10"`)},
									}},
								},
							}},
						}},
					}},
				}},
			}},
			{"main", &Value{Ident: ptr.To("runtime")}},
		},
		expr,
	}, script)
}

func TestComplex(t *testing.T) {
	script, err := NewScript([]byte(
		`linux_runtime = solve(
		requirements=[
			Req(name="language/python")
		],
		platforms=["67890"]
)

win_runtime = solve(
		requirements=[
			Req(name="language/perl")
		],
		platforms=["12345"]
)

main = merge(
		win_installer(win_runtime),
		tar_installer(linux_runtime)
)
`))
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"linux_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python"`)}},
									},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`"67890"`)}},
						},
					}},
				}},
			}},
			{"win_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/perl"`)}},
									},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`"12345"`)}},
						},
					}},
				}},
			}},
			{"main", &Value{
				FuncCall: &FuncCall{"merge", []*Value{
					{FuncCall: &FuncCall{"win_installer", []*Value{{Ident: ptr.To("win_runtime")}}}},
					{FuncCall: &FuncCall{"tar_installer", []*Value{{Ident: ptr.To("linux_runtime")}}}},
				}}}},
		},
		expr,
	}, script)
}

const example = `runtime = solve(
	at_time = "2023-04-27T17:30:05.999000Z",
	platforms = ["96b7e6f2-bebf-564c-bc1c-f04482398f38", "96b7e6f2-bebf-564c-bc1c-f04482398f38"],
	requirements = [
		Req(name="language/python"),
		Req(name="language/python/requests", version="3.10.10")
	],
	solver_version = 0
)

main = runtime`

func TestExample(t *testing.T) {
	script, err := NewScript([]byte(example))
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{
						"at_time", &Value{Str: ptr.To(`"2023-04-27T17:30:05.999000Z"`)},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
							{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
						}},
					}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python"`)}},
									},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python/requests"`)}},
									},
									{Assignment: &Assignment{
										"version", &Value{Str: ptr.To(`"3.10.10"`)}},
									},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"solver_version", &Value{Number: ptr.To(float64(0))},
					}},
				}},
			}},
			{"main", &Value{Ident: ptr.To("runtime")}},
		},
		expr,
	}, script)
}

func TestString(t *testing.T) {
	script, err := NewScript([]byte(
		`runtime = solve(
		platforms=["12345", "67890"],
		requirements=[Req(name="language/python", version="3.10.10")]
)

main = runtime
`))
	require.NoError(t, err)

	assert.Equal(t,
		`runtime = solve(
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name="language/python", version="3.10.10")
	]
)

main = runtime`, script.String())
}

func TestRoundTrip(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "buildscript-")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	script, err := NewScript([]byte(example))
	require.NoError(t, err)

	tmpfile.Write([]byte(script.String()))
	tmpfile.Close()

	roundTripScript, err := newScriptFromFile(tmpfile.Name(), "", "", nil)
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
}

func TestJson(t *testing.T) {
	script, err := NewScript([]byte(
		`runtime = solve(
		requirements=[
				{
						name="language/python"
				}
		],
		platforms=["12345", "67890"]
)

main = runtime
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
	newExpr := script.Expr
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
	for _, assignment := range script.Expr.Let.Assignments {
		if assignment.Name == "runtime" {
			args := assignment.Value.Ap.Arguments
			require.NotNil(t, args)
			for _, arg := range args {
				if arg.Assignment != nil && arg.Assignment.Name == "solver_version" {
					assert.Equal(t, null, arg.Assignment.Value.Str)
					nullHandled = true
				}
			}
		}
	}
	assert.True(t, nullHandled, "JSON null not encountered")
}
