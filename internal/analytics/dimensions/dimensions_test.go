package dimensions

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/p"
)

func TestMap_Merge(t *testing.T) {
	tests := []struct {
		name      string
		input     *Values
		mergeWith *Values
		expected  *Values
	}{
		{
			"Simple",
			&Values{Version: p.StrP("inputVersion")},
			&Values{BranchName: p.StrP("mergeBranchName")},
			&Values{Version: p.StrP("inputVersion"), BranchName: p.StrP("mergeBranchName")},
		},
		{
			"Override",
			&Values{Version: p.StrP("inputVersion")},
			&Values{Version: p.StrP("mergeVersion")},
			&Values{Version: p.StrP("mergeVersion")},
		},
		{
			"Nils don't count",
			&Values{Version: p.StrP("inputVersion")},
			&Values{Version: nil},
			&Values{Version: p.StrP("inputVersion")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Merge(tt.mergeWith)
			different, diffValue := diff(tt.input, tt.expected)
			if different {
				t.Errorf("%v", diffValue)
			}
		})
	}
}

func diff(m1 *Values, m2 *Values) (bool, string) {
	fields1 := reflect.ValueOf(m1).Elem().Type()
	fields2 := reflect.ValueOf(m2).Elem().Type()
	values1 := reflect.ValueOf(m1)
	values2 := reflect.ValueOf(m2)

	result := []string{}
	for i := 0; i < fields1.NumField(); i++ {
		field1 := fields1.Field(i)
		field2 := fields2.Field(i)
		val1Elem := values1.Elem().Field(i)
		val2Elem := values2.Elem().Field(i)

		if strings.ToUpper(field1.Name[0:1]) != field1.Name[0:1] {
			continue
		}

		val1 := val1Elem.Elem().String()
		val2 := val2Elem.Elem().String()

		if val1Elem.IsNil() {
			val1 = "<nil>"
		}
		if val2Elem.IsNil() {
			val2 = "<nil>"
		}

		if val1 != val2 {
			result = append(result, fmt.Sprintf("%s:%s != %s:%s", field1.Name, val1, field2.Name, val2))
		}
	}

	if len(result) > 0 {
		return true, strings.Join(result, "\n")
	}

	return false, ""
}
