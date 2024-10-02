package buildscript

import (
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProject = "https://platform.activestate.com/org/project?branch=main&commitID=00000000-0000-0000-0000-000000000000"
const testTime = "2000-01-01T00:00:00.000Z"

func checkoutInfoString(project, time string) string {
	return "```\n" +
		"Project: " + project + "\n" +
		"Time: " + time + "\n" +
		"```\n"
}

var testCheckoutInfo string

func init() {
	testCheckoutInfo = checkoutInfoString(testProject, testTime)
}

func TestRawRepresentation(t *testing.T) {
	script, err := Unmarshal([]byte(
		testCheckoutInfo + `
runtime = solve(
	at_time = TIME,
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

	assert.Contains(t, *script.raw.Info, "2000-01-01T00:00:00.000Z")

	assert.Equal(t, &rawBuildScript{
		Info: ptr.To(testCheckoutInfo[2 : len(testCheckoutInfo)-3]),
		Assignments: []*Assignment{
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`TIME`)}}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`linux`)},
							{Str: ptr.To(`windows`)},
						}},
					}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("python")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language")}}},
								}}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("requests")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language/python")}}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{"value", &Value{Str: ptr.To("3.10.10")}}},
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
	}, script.raw)
}

func TestComplex(t *testing.T) {
	script, err := Unmarshal([]byte(
		testCheckoutInfo + `
linux_runtime = solve(
		at_time = TIME,
		requirements=[
			Req(name = "python", namespace = "language")
		],
		platforms=["67890"]
)

win_runtime = solve(
		at_time = TIME,
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

	assert.Contains(t, *script.raw.Info, "2000-01-01T00:00:00.000Z")

	assert.Equal(t, &rawBuildScript{
		Info: ptr.To(testCheckoutInfo[2 : len(testCheckoutInfo)-3]),
		Assignments: []*Assignment{
			{"linux_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`TIME`)}}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("python")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language")}}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{{Str: ptr.To(`67890`)}}},
					}},
				}},
			}},
			{"win_runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`TIME`)}}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("perl")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language")}}},
								},
							}},
						}},
					}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{{Str: ptr.To(`12345`)}}},
					}},
				}},
			}},
			{"main", &Value{
				FuncCall: &FuncCall{"merge", []*Value{
					{FuncCall: &FuncCall{"win_installer", []*Value{{Ident: ptr.To("win_runtime")}}}},
					{FuncCall: &FuncCall{"tar_installer", []*Value{{Ident: ptr.To("linux_runtime")}}}},
				}}}},
		},
	}, script.raw)
}

func TestComplexVersions(t *testing.T) {
	checkoutInfo := checkoutInfoString(testProject, "2023-04-27T17:30:05.999Z")
	script, err := Unmarshal([]byte(
		checkoutInfo + `
runtime = solve(
	at_time = TIME,
	platforms = ["96b7e6f2-bebf-564c-bc1c-f04482398f38", "96b7e6f2-bebf-564c-bc1c-f04482398f38"],
	requirements = [
		Req(name = "python", namespace = "language"),
		Req(name = "requests", namespace = "language/python", version = Eq(value = "3.10.10")),
		Req(name = "argparse", namespace = "language/python", version = And(left = Gt(value = "1.0"), right = Lt(value = "2.0")))
	],
	solver_version = 0
)

main = runtime
`))
	require.NoError(t, err)

	assert.Contains(t, *script.raw.Info, "2023-04-27T17:30:05.999Z")

	assert.Equal(t, &rawBuildScript{
		Info: ptr.To(checkoutInfo[2 : len(checkoutInfo)-3]),
		Assignments: []*Assignment{
			{"runtime", &Value{
				FuncCall: &FuncCall{"solve", []*Value{
					{Assignment: &Assignment{"at_time", &Value{Ident: ptr.To(`TIME`)}}},
					{Assignment: &Assignment{
						"platforms", &Value{List: &[]*Value{
							{Str: ptr.To(`96b7e6f2-bebf-564c-bc1c-f04482398f38`)},
							{Str: ptr.To(`96b7e6f2-bebf-564c-bc1c-f04482398f38`)},
						}},
					}},
					{Assignment: &Assignment{
						"requirements", &Value{List: &[]*Value{
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("python")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language")}}},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("requests")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language/python")}}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "Eq",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To("3.10.10")}}},
											},
										}},
									}},
								},
							}},
							{FuncCall: &FuncCall{
								Name: "Req",
								Arguments: []*Value{
									{Assignment: &Assignment{"name", &Value{Str: ptr.To("argparse")}}},
									{Assignment: &Assignment{"namespace", &Value{Str: ptr.To("language/python")}}},
									{Assignment: &Assignment{
										"version", &Value{FuncCall: &FuncCall{
											Name: "And",
											Arguments: []*Value{
												{Assignment: &Assignment{Key: "left", Value: &Value{FuncCall: &FuncCall{
													Name: "Gt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To("1.0")}}},
													},
												}}}},
												{Assignment: &Assignment{Key: "right", Value: &Value{FuncCall: &FuncCall{
													Name: "Lt",
													Arguments: []*Value{
														{Assignment: &Assignment{Key: "value", Value: &Value{Str: ptr.To("2.0")}}},
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
	}, script.raw)
}
