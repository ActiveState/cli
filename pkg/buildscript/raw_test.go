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
		[]*assignment{
			{"runtime", &value{
				FuncCall: &funcCall{"solve", []*value{
					{Assignment: &assignment{"at_time", &value{Ident: ptr.To(`at_time`)}}},
					{Assignment: &assignment{
						"platforms", &value{List: &[]*value{
							{Str: ptr.To(`"linux"`)},
							{Str: ptr.To(`"windows"`)},
						}},
					}},
					{Assignment: &assignment{
						"requirements", &value{List: &[]*value{
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("python")}},
									{Assignment: &assignment{"namespace", newString("language")}},
								}}},
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("requests")}},
									{Assignment: &assignment{"namespace", newString("language/python")}},
									{Assignment: &assignment{
										"version", &value{FuncCall: &funcCall{
											Name: "Eq",
											Arguments: []*value{
												{Assignment: &assignment{"value", newString("3.10.10")}},
											},
										}},
									}},
								},
							}},
						}},
					}},
					{Assignment: &assignment{"solver_version", &value{Null: &null{}}}},
				}},
			}},
			{"main", &value{Ident: ptr.To("runtime")}},
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
		[]*assignment{
			{"linux_runtime", &value{
				FuncCall: &funcCall{"solve", []*value{
					{Assignment: &assignment{"at_time", &value{Ident: ptr.To(`at_time`)}}},
					{Assignment: &assignment{
						"requirements", &value{List: &[]*value{
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("python")}},
									{Assignment: &assignment{"namespace", newString("language")}},
								},
							}},
						}},
					}},
					{Assignment: &assignment{
						"platforms", &value{List: &[]*value{{Str: ptr.To(`"67890"`)}}},
					}},
				}},
			}},
			{"win_runtime", &value{
				FuncCall: &funcCall{"solve", []*value{
					{Assignment: &assignment{"at_time", &value{Ident: ptr.To(`at_time`)}}},
					{Assignment: &assignment{
						"requirements", &value{List: &[]*value{
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("perl")}},
									{Assignment: &assignment{"namespace", newString("language")}},
								},
							}},
						}},
					}},
					{Assignment: &assignment{
						"platforms", &value{List: &[]*value{{Str: ptr.To(`"12345"`)}}},
					}},
				}},
			}},
			{"main", &value{
				FuncCall: &funcCall{"merge", []*value{
					{FuncCall: &funcCall{"win_installer", []*value{{Ident: ptr.To("win_runtime")}}}},
					{FuncCall: &funcCall{"tar_installer", []*value{{Ident: ptr.To("linux_runtime")}}}},
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
		[]*assignment{
			{"runtime", &value{
				FuncCall: &funcCall{"solve", []*value{
					{Assignment: &assignment{"at_time", &value{Ident: ptr.To(`at_time`)}}},
					{Assignment: &assignment{
						"platforms", &value{List: &[]*value{
							{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
							{Str: ptr.To(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
						}},
					}},
					{Assignment: &assignment{
						"requirements", &value{List: &[]*value{
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("python")}},
									{Assignment: &assignment{"namespace", newString("language")}},
								},
							}},
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("requests")}},
									{Assignment: &assignment{"namespace", newString("language/python")}},
									{Assignment: &assignment{
										"version", &value{FuncCall: &funcCall{
											Name: "Eq",
											Arguments: []*value{
												{Assignment: &assignment{Key: "value", Value: newString("3.10.10")}},
											},
										}},
									}},
								},
							}},
							{FuncCall: &funcCall{
								Name: "Req",
								Arguments: []*value{
									{Assignment: &assignment{"name", newString("argparse")}},
									{Assignment: &assignment{"namespace", newString("language/python")}},
									{Assignment: &assignment{
										"version", &value{FuncCall: &funcCall{
											Name: "And",
											Arguments: []*value{
												{Assignment: &assignment{Key: "left", Value: &value{FuncCall: &funcCall{
													Name: "Gt",
													Arguments: []*value{
														{Assignment: &assignment{Key: "value", Value: newString("1.0")}},
													},
												}}}},
												{Assignment: &assignment{Key: "right", Value: &value{FuncCall: &funcCall{
													Name: "Lt",
													Arguments: []*value{
														{Assignment: &assignment{Key: "value", Value: newString("2.0")}},
													},
												}}}},
											},
										}},
									}},
								},
							}},
						}},
					}},
					{Assignment: &assignment{"solver_version", &value{Number: ptr.To(float64(0))}}},
				}},
			}},
			{"main", &value{Ident: ptr.To("runtime")}},
		},
		&atTime,
	}, script.raw)
}
