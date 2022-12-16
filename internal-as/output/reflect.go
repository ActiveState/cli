package output

import (
	"fmt"
	"reflect"
	"strings"
)

type structField struct {
	name  string
	l10n  string
	opts  []string
	value interface{}
}

// structMeta holds the basic meta information required by the Plain outputer
type structMeta []structField

// parseStructMeta will use reflect to populate structMeta for the given struct
func parseStructMeta(v interface{}) (structMeta, error) {
	structRfl := reflect.Indirect(reflect.ValueOf(v))

	// Fail if the passed type is not a struct
	if !isStruct(structRfl) {
		return structMeta{}, fmt.Errorf("Expected struct, got: %s", structRfl.Kind().String())
	}

	var meta structMeta
	for i := 0; i < structRfl.Type().NumField(); i++ {
		fieldRfl := structRfl.Type().Field(i)
		valueRfl := structRfl.Field(i)

		if strings.ToLower(fieldRfl.Name[0:1]) == fieldRfl.Name[0:1] {
			continue // don't include unexported fields
		}

		serialized := strings.ToLower(string(fieldRfl.Name[0:1])) + fieldRfl.Name[1:]
		if v, ok := fieldRfl.Tag.Lookup("locale"); ok {
			serialized = v
		}

		opts := []string{}
		if v, ok := fieldRfl.Tag.Lookup("opts"); ok {
			opts = strings.Split(v, ",")
		}

		field := structField{
			name:  fieldRfl.Name,
			l10n:  serialized,
			opts:  opts,
			value: valueRfl.Interface(),
		}
		meta = append(meta, field)
	}

	return meta, nil
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

// parseMap will turn an interface that is a map into a slice with interface entries
func parseMap(v interface{}) (map[string]interface{}, error) {
	structRfl := reflect.ValueOf(v)

	result := map[string]interface{}{}

	// Fail if the passed type is not a map
	if structRfl.Kind() != reflect.Map {
		return result, fmt.Errorf("Expected map, got: %s", structRfl.Kind().String())
	}

	mapRange := structRfl.MapRange()
	for mapRange.Next() {
		result[mapRange.Key().String()] = mapRange.Value().Interface()
	}

	return result, nil
}

func valueOf(v interface{}) reflect.Value {
	return reflect.ValueOf(v)
}

func indirectKind(v interface{}) reflect.Kind {
	return reflect.Indirect(valueOf(v)).Kind()
}

func isStruct(v interface{}) bool {
	return indirectKind(v) == reflect.Struct
}

func isSlice(v interface{}) bool {
	return indirectKind(v) == reflect.Slice
}
