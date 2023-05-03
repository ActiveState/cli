package errs

import (
	"reflect"
	"testing"
)

func TestEncodeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want interface{}
	}{
		{
			"Single",
			New("error1"),
			"error1",
		},
		{
			"Wrapped",
			Wrap(New("error1"), "error2"),
			map[string]interface{}{
				"error2": "error1",
			},
		},
		{
			"Stacked",
			Pack(New("error1"), New("error2"), New("error3")),
			[]interface{}{
				"error1",
				"error2",
				"error3",
			},
		},
		{
			"Stacked and Wrapped",
			Pack(
				New("error1"),
				Wrap(New("error2"), "error2-wrap"),
				New("error3"),
			),
			[]interface{}{
				"error1",
				map[string]interface{}{
					"error2-wrap": "error2",
				},
				"error3",
			},
		},
		{
			"Stacked, Wrapped and Stacked",
			Pack(
				New("error1"),
				Wrap(
					Pack(New("error2a"), New("error2b")),
					"error2-wrap",
				),
				New("error3")),
			[]interface{}{
				"error1",
				map[string]interface{}{
					"error2-wrap": []interface{}{"error2a", "error2b"},
				},
				"error3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeErrorForJoin(tt.err)
			if !reflect.DeepEqual(encoded, tt.want) {
				t.Fatalf("got %v, want %v", encoded, tt.want)
			}
		})
	}
}
