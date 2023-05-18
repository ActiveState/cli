package buildscript

import (
	"bytes"
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
	assert.Equal(t, file.Script, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{String: p.StrP(`"linux"`)},
								&Value{String: p.StrP(`"windows"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"python"`)}},
									&Assignment{"namespace", &Value{String: p.StrP(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"requests"`)}},
									&Assignment{"namespace", &Value{String: p.StrP(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{String: p.StrP(`"eq"`)}},
											&Assignment{"version", &Value{String: p.StrP(`"3.10.10"`)}},
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
	})
}

func TestComplex(t *testing.T) {
	file, err := get(filepath.Join("testdata", "complex.lo"))
	require.NoError(t, err)
	assert.Equal(t, file.Script, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"linux_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"language/python"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{String: p.StrP(`"67890"`)}},
							},
						}},
					}},
				}},
				&Assignment{"win_runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"language/perl"`)}},
								}},
							}},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{String: p.StrP(`"12345"`)}},
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
	})
}

func TestExample(t *testing.T) {
	file, err := get(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	assert.Equal(t, file.Script, &Script{
		&Let{
			[]*Assignment{
				&Assignment{"runtime", &Value{
					FuncCall: &FuncCall{"solve", []*Value{
						&Value{Assignment: &Assignment{
							"at_time", &Value{String: p.StrP(`"2023-04-27T17:30:05.999000Z"`)},
						}},
						&Value{Assignment: &Assignment{
							"platforms", &Value{List: &[]*Value{
								&Value{String: p.StrP(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
								&Value{String: p.StrP(`"96b7e6f2-bebf-564c-bc1c-f04482398f38"`)},
							}},
						}},
						&Value{Assignment: &Assignment{
							"requirements", &Value{List: &[]*Value{
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"python"`)}},
									&Assignment{"namespace", &Value{String: p.StrP(`"language"`)}},
								}},
								&Value{Object: &[]*Assignment{
									&Assignment{"name", &Value{String: p.StrP(`"requests"`)}},
									&Assignment{"namespace", &Value{String: p.StrP(`"language/python"`)}},
									&Assignment{"version_requirements", &Value{List: &[]*Value{
										&Value{Object: &[]*Assignment{
											&Assignment{"comparator", &Value{String: p.StrP(`"eq"`)}},
											&Assignment{"version", &Value{String: p.StrP(`"3.10.10"`)}},
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
	})
}

func TestWrite(t *testing.T) {
	file, err := get(filepath.Join("testdata", "moderate.lo"))
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	file.Script.Write(buf)
	assert.Equal(t, buf.String(),
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
	runtime`)
}

func TestRoundTrip(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "buildscript-")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	file, err := get(filepath.Join("testdata", "example.lo"))
	require.NoError(t, err)
	script := file.Script

	file.Script.Write(tmpfile)
	tmpfile.Close()

	file, err = get(tmpfile.Name())
	require.NoError(t, err)

	assert.Equal(t, file.Script, script)
}
