package output

import (
	"fmt"
	"reflect"
	"strings"
)

type structMeta struct {
	fields           []string
	serializedFields []string
	values           []interface{}
}

func parseStructMeta(v interface{}) (structMeta, error) {
	structRfl := reflect.ValueOf(v)

	// Fail if the passed type is not a struct
	if structRfl.Kind() != reflect.Struct {
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

		serialized := strings.ToLower(string(fieldRfl.Name[0:1])) + fieldRfl.Name[1:(len(fieldRfl.Name))]
		if v, ok := fieldRfl.Tag.Lookup("serialized"); ok {
			serialized = v
		}
		info.serializedFields = append(info.serializedFields, serialized)
	}

	return info, nil
}

func parseSlice(v interface{}) ([]interface{}, error) {
	structRfl := reflect.ValueOf(v)

	// Fail if the passed type is not a slice
	if structRfl.Kind() != reflect.Slice {
		return []interface{}{}, fmt.Errorf("Expected slice, got: %s", structRfl.Kind().String())
	}

	return v.([]interface{}), nil
}
