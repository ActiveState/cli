package output

import (
	"fmt"
	"reflect"
	"strings"
)

// structMeta holds the basic meta information required by the Plain outputer
type structMeta struct {
	fields       []string
	localeFields []string
	values       []interface{}
}

// parseStructMeta will use reflect to populate structMeta for the given struct
func parseStructMeta(v interface{}) (structMeta, error) {
	structRfl := reflect.ValueOf(v)

	// Fail if the passed type is not a struct
	if !isStruct(structRfl) {
		return structMeta{}, fmt.Errorf("Expected struct, got: %s", structRfl.Kind().String())
	}

	info := structMeta{}
	for i := 0; i < structRfl.Type().NumField(); i++ {
		fieldRfl := structRfl.Type().Field(i)
		valueRfl := structRfl.Field(i)

		if strings.ToLower(fieldRfl.Name[0:1]) == fieldRfl.Name[0:1] {
			continue // don't include unexported fields
		}

		info.fields = append(info.fields, fieldRfl.Name)
		info.values = append(info.values, valueRfl.Interface())

		serialized := strings.ToLower(string(fieldRfl.Name[0:1])) + fieldRfl.Name[1:]
		if v, ok := fieldRfl.Tag.Lookup("locale"); ok {
			serialized = v
		}
		info.localeFields = append(info.localeFields, serialized)
	}

	return info, nil
}

// parseSlice will turn an interface that is a slice into a slice with interface entries
func parseSlice(v interface{}) ([]interface{}, error) {
	structRfl := reflect.ValueOf(v)

	result := []interface{}{}

	// Fail if the passed type is not a slice
	if structRfl.Kind() != reflect.Slice {
		return result, fmt.Errorf("Expected slice, got: %s", structRfl.Kind().String())
	}

	for i := 0; i < structRfl.Len(); i++ {
		result = append(result, structRfl.Index(i).Interface())
	}

	return result, nil
}

func isStruct(v interface{}) bool {
	valueRfl := reflect.ValueOf(v)
	return valueRfl.Kind() == reflect.Struct
}
