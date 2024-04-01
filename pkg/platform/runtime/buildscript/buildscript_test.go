package buildscript

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

// toBuildExpression converts given script constructed by Participle into a buildexpression.
// This function should not be used to convert an arbitrary script to buildexpression.
// New*() populates the Expr field with the equivalent build expression.
// This function exists solely for testing that functionality.
func toBuildExpression(script *Script) (*buildexpression.BuildExpression, error) {
	bytes, err := json.Marshal(script)
	if err != nil {
		return nil, err
	}
	return buildexpression.New(bytes)
}

func TestBasic(t *testing.T) {
	script, err := New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = ["linux", "windows"],
	requirements = [
		Req(name = "language/python"),
		Req(name = "language/python/requests", version = Eq(value = "3.10.10"))
	]
)

main = runtime
`))
	require.NoError(t, err)

	atTime, err := strfmt.ParseDateTime("2000-01-01T00:00:00.000Z")
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"at_time", &Value{Str: ptr.To(`"2000-01-01T00:00:00.000Z"`)}},
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`at_time`)}}},
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
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To(`"3.10.10"`)}}},
											},
										}},
									}},
								},
							}},
						}},
					}},
				}},
			}},
			{"main", &Value{Ident: ptr.To("runtime")}},
		},
		&atTime,
		expr,
	}, script)
}

func TestComplex(t *testing.T) {
	script, err := New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
linux_runtime = solve(
		at_time = at_time,
		requirements=[
			Req(name = "language/python")
		],
		platforms=["67890"]
)

win_runtime = solve(
		at_time = at_time,
		requirements=[
			Req(name = "language/perl")
		],
		platforms=["12345"]
)

main = merge(
		win_installer(win_runtime),
		tar_installer(linux_runtime)
)
`))
	require.NoError(t, err)

	atTime, err := strfmt.ParseDateTime("2000-01-01T00:00:00.000Z")
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"at_time", &Value{Str: ptr.To(`"2000-01-01T00:00:00.000Z"`)}},
			{"linux_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`at_time`)}}},
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
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`at_time`)}}},
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
		&atTime,
		expr,
	}, script)
}

const example = `at_time = "2023-04-27T17:30:05.999Z"
runtime = solve(
	at_time = at_time,
	platforms = ["96b7e6f2-bebf-564c-bc1c-f04482398f38", "96b7e6f2-bebf-564c-bc1c-f04482398f38"],
	requirements = [
		Req(name = "language/python"),
		Req(name = "language/python/requests", version = Eq(value = "3.10.10")),
		Req(name = "language/python/argparse", version = And(left = Gt(value = "1.0"), right = Lt(value = "2.0")))
	],
	solver_version = 0
)

main = runtime`

func TestExample(t *testing.T) {
	script, err := New([]byte(example))
	require.NoError(t, err)

	atTime, err := strfmt.ParseDateTime("2023-04-27T17:30:05.999Z")
	require.NoError(t, err)

	expr, err := toBuildExpression(script)
	require.NoError(t, err)

	assert.Equal(t, &Script{
		[]*Assignment{
			{"at_time", &Value{Str: ptr.To(`"2023-04-27T17:30:05.999Z"`)}},
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{
						"at_time", &Value{Ident: ptr.To(`at_time`)},
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
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To(`"3.10.10"`)}}},
											},
										}},
									}},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{
										"name", &Value{Str: ptr.To(`"language/python/argparse"`)}},
									},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "And",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "left", Value: &Value{FuncCall: &FuncCall{
													Name: "Gt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To(`"1.0"`)}}},
													},
												}}}},
												{Assignment: &Assignment{Key: "right", Value: &Value{FuncCall: &FuncCall{
													Name: "Lt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To(`"2.0"`)}}},
													},
												}}}},
											},
										}},
									}},
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
		&atTime,
		expr,
	}, script)
}

func TestString(t *testing.T) {
	script, err := New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
		at_time = at_time,
		platforms=["12345", "67890"],
		requirements=[Req(name = "language/python", version = Eq(value = "3.10.10"))]
)

main = runtime
`))
	require.NoError(t, err)

	assert.Equal(t,
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/python", version = Eq(value = "3.10.10"))
	]
)

main = runtime`, script.String())
}

func TestRoundTrip(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "buildscript-")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	script, err := New([]byte(example))
	require.NoError(t, err)

	_, err = tmpfile.Write([]byte(script.String()))
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	roundTripScript, err := ScriptFromFile(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
}

func TestJson(t *testing.T) {
	script, err := New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
		at_time = at_time,
		requirements=[
			Req(name = "language/python")
		],
		platforms=["12345", "67890"]
)

main = runtime
`))
	require.NoError(t, err)

	inputJson := &bytes.Buffer{}
	err = json.Compact(inputJson, []byte(`{
    "let": {
      "runtime": {
        "solve": {
          "at_time": "$at_time",
          "requirements": [
            {
              "name": "python",
              "namespace": "language"
            }
          ],
          "platforms": ["12345", "67890"]
        }
      },
      "in": "$runtime"
    }
  }`))
	require.NoError(t, err)
	// Cannot compare marshaled JSON directly with inputJson due to key sort order, so unmarshal and
	// remarshal before making the comparison. json.Marshal() produces the same key sort order.
	marshaledInput := make(map[string]interface{})
	err = json.Unmarshal(inputJson.Bytes(), &marshaledInput)
	require.NoError(t, err)
	expectedJson, err := json.Marshal(marshaledInput)
	require.NoError(t, err)

	actualJson, err := json.Marshal(script.Expr)
	require.NoError(t, err)
	assert.Equal(t, string(expectedJson), string(actualJson))
}

func TestBuildExpression(t *testing.T) {
	expr, err := buildexpression.New([]byte(`{
  "let": {
    "runtime": {
      "solve_legacy": {
        "at_time": "2023-04-27T17:30:05.999Z",
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
	script, err := NewFromCommit(nil, expr)
	require.NoError(t, err)
	require.NotNil(t, script)
	newExpr := script.Expr
	exprBytes, err := json.Marshal(expr)
	require.NoError(t, err)
	newExprBytes, err := json.Marshal(newExpr)
	require.NoError(t, err)
	assert.Equal(t, string(exprBytes), string(newExprBytes))

	// Verify comparisons between buildscripts is accurate.
	newScript, err := NewFromCommit(nil, newExpr)
	require.NoError(t, err)
	assert.True(t, script.Equals(newScript))

	// Verify null JSON value is handled correctly.
	var null *string
	nullHandled := false
	for _, assignment := range newExpr.Let.Assignments {
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
