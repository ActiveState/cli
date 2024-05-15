package buildexpression

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAdd(t *testing.T) {
	exprA, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "DateTime",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}
`))
	require.NoError(t, err)

	exprB, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "JSON",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}
`))
	require.NoError(t, err)

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "DateTime", Operation: mono_models.CommitChangeEditableOperationAdded},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	v, err := json.MarshalIndent(mergedExpr, "", "  ")
	require.NoError(t, err)

	assert.Equal(t,
		`{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "JSON",
            "namespace": "language/perl"
          },
          {
            "name": "DateTime",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}`, string(v))
}

func TestMergeRemove(t *testing.T) {
	exprA, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "DateTime",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}
`))
	require.NoError(t, err)

	exprB, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "JSON",
            "namespace": "language/perl"
          },
          {
            "name": "DateTime",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}
`))

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationRemoved},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	v, err := json.MarshalIndent(mergedExpr, "", "  ")
	require.NoError(t, err)

	assert.Equal(t,
		`{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "DateTime",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}`, string(v))
}

func TestMergeConflict(t *testing.T) {
	exprA, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          }
        ]
      }
    }
  }
}
`))
	require.NoError(t, err)

	exprB, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345"
        ],
        "requirements": [
          {
            "name": "perl",
            "namespace": "language"
          },
          {
            "name": "JSON",
            "namespace": "language/perl"
          }
        ]
      }
    }
  }
}
`))
	require.NoError(t, err)

	assert.False(t, isAutoMergePossible(exprA, exprB)) // platforms do not match

	_, err = Merge(exprA, exprB, nil)
	require.Error(t, err)
}

func TestDeleteKey(t *testing.T) {
	m := map[string]interface{}{"foo": map[string]interface{}{"bar": "baz", "quux": "foobar"}}
	assert.True(t, deleteKey(&m, "quux"), "did not find quux")
	_, exists := m["foo"].(map[string]interface{})["quux"]
	assert.False(t, exists, "did not delete quux")
}
