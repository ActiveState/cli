package buildscript

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawRepresentation(t *testing.T) {
	script, err := Unmarshal([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = ["linux", "windows"],
	requirements = [
		Req(name = "python", namespace = "language"),
		Req(name = "requests", namespace = "language/python", version = Eq(value = "3.10.10"))
	],
	solver_version = null
)

main = runtime
`))
	require.NoError(t, err)

	atTimeStrfmt, err := strfmt.ParseDateTime("2000-01-01T00:00:00.000Z")
	require.NoError(t, err)
	atTime := time.Time(atTimeStrfmt)

	assert.Equal(t, &rawBuildScript{
		[]*Assignment{
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
									{Assignment: &Assignment{"name", newString("python")}},
									{Assignment: &Assignment{"namespace", newString("language")}},
								}}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", newString("requests")}},
									{Assignment: &Assignment{"namespace", newString("language/python")}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{"value", newString("3.10.10")}},
											},
										}},
									}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{"solver_version", &Value{Null: &Null{}}}},
				}},
			}},
			{"main", &Value{Ident: ptr.To("runtime")}},
		},
		&atTime,
	}, script.raw)
}

func TestComplex(t *testing.T) {
	script, err := Unmarshal([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
linux_runtime = solve(
		at_time = at_time,
		requirements=[
			Req(name = "python", namespace = "language")
		],
		platforms=["67890"]
)

win_runtime = solve(
		at_time = at_time,
		requirements=[
			Req(name = "perl", namespace = "language")
		],
		platforms=["12345"]
)

main = merge(
		win_installer(win_runtime),
		tar_installer(linux_runtime)
)
`))
	require.NoError(t, err)

	atTimeStrfmt, err := strfmt.ParseDateTime("2000-01-01T00:00:00.000Z")
	require.NoError(t, err)
	atTime := time.Time(atTimeStrfmt)

	assert.Equal(t, &rawBuildScript{
		[]*Assignment{
			{"linux_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`at_time`)}}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", newString("python")}},
									{Assignment: &Assignment{"namespace", newString("language")}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{{Str: ptr.To(`"67890"`)}}},
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
									{Assignment: &Assignment{"name", newString("perl")}},
									{Assignment: &Assignment{"namespace", newString("language")}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{{Str: ptr.To(`"12345"`)}}},
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
	}, script.raw)
}

const buildscriptWithComplexVersions = `at_time = "2023-04-27T17:30:05.999Z"
runtime = solve(
	at_time = at_time,
	platforms = ["96b7e6f2-bebf-564c-bc1c-f04482398f38", "96b7e6f2-bebf-564c-bc1c-f04482398f38"],
	requirements = [
		Req(name = "python", namespace = "language"),
		Req(name = "requests", namespace = "language/python", version = Eq(value = "3.10.10")),
		Req(name = "argparse", namespace = "language/python", version = And(left = Gt(value = "1.0"), right = Lt(value = "2.0")))
	],
	solver_version = 0
)

main = runtime`

func TestComplexVersions(t *testing.T) {
	script, err := Unmarshal([]byte(buildscriptWithComplexVersions))
	require.NoError(t, err)

	atTimeStrfmt, err := strfmt.ParseDateTime("2023-04-27T17:30:05.999Z")
	require.NoError(t, err)
	atTime := time.Time(atTimeStrfmt)

	assert.Equal(t, &rawBuildScript{
		[]*Assignment{
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`at_time`)}}},
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
									{Assignment: &Assignment{"name", newString("python")}},
									{Assignment: &Assignment{"namespace", newString("language")}},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", newString("requests")}},
									{Assignment: &Assignment{"namespace", newString("language/python")}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "value", Value: newString("3.10.10")}},
											},
										}},
									}},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", newString("argparse")}},
									{Assignment: &Assignment{"namespace", newString("language/python")}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "And",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "left", Value: &Value{FuncCall: &FuncCall{
													Name: "Gt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: newString("1.0")}},
													},
												}}}},
												{Assignment: &Assignment{Key: "right", Value: &Value{FuncCall: &FuncCall{
													Name: "Lt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: newString("2.0")}},
													},
												}}}},
											},
										}},
									}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{"solver_version", &Value{Number: ptr.To(float64(0))}}},
				}},
			}},
			{"main", &Value{Ident: ptr.To("runtime")}},
		},
		&atTime,
	}, script.raw)
}
