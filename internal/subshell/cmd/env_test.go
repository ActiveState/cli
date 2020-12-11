package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/thoas/go-funk"
	"github.com/ActiveState/cli/internal/osutils"
)

type RegistryKeyMock struct {
	getCalls       []string
	setCalls       []string
	setExpandCalls []string
	delCalls       []string

	getResults map[string]RegistryValue
	setResults map[string]error
	delResults map[string]error
}

type RegistryValue struct {
	Value string
	Error error
}

func (r *RegistryKeyMock) GetStringValue(name string) (string, uint32, error) {
	r.getCalls = append(r.getCalls, name)
	if v, ok := r.getResults[name]; ok {
		return v.Value, 0, v.Error
	}
	return "", 0, osutils.NotExistError()
}

func (r *RegistryKeyMock) SetStringValue(name, value string) error {
	r.setCalls = append(r.setCalls, name+"="+value)
	if v, ok := r.setResults[name]; ok {
		return v
	}
	return nil
}

func (r *RegistryKeyMock) SetExpandStringValue(name, value string) error {
	r.setExpandCalls = append(r.setExpandCalls, name+"="+value)
	if v, ok := r.setResults[name]; ok {
		return v
	}
	return nil
}

func (r *RegistryKeyMock) DeleteValue(name string) error {
	r.delCalls = append(r.getCalls, name)
	if v, ok := r.delResults[name]; ok {
		return v
	}
	return nil
}

func (r *RegistryKeyMock) Close() error {
	return nil
}

func openKeyMock(path string) (osutils.RegistryKey, error) {
	return &RegistryKeyMock{}, nil
}

func TestCmdEnv_unset(t *testing.T) {
	type fields struct {
		registryMock *RegistryKeyMock
		openKeyErr   error
	}
	type args struct {
		name          string
		ifValueEquals string
	}
	type want struct {
		returnValue      error
		registryGetCalls *[]string // nil means it should have no calls
		registrySetCalls *[]string
		registryDelCalls *[]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			"unset, value not equals",
			fields{&RegistryKeyMock{}, nil},
			args{
				"key",
				"value_not_equal",
			},
			want{
				nil,
				&[]string{},
				nil,
				nil,
			},
		},
		{
			"unset, value equals",
			fields{&RegistryKeyMock{
				getResults: map[string]RegistryValue{
					"key": RegistryValue{"value_equals", nil},
				},
			}, nil},
			args{
				"key",
				"value_equals",
			},
			want{
				nil,
				&[]string{},
				nil,
				&[]string{"key"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CmdEnv{
				openKeyFn: func(path string) (osutils.RegistryKey, error) {
					return tt.fields.registryMock, tt.fields.openKeyErr
				},
			}
			if got := c.unset(tt.args.name, tt.args.ifValueEquals); !reflect.DeepEqual(got, tt.want.returnValue) {
				t.Errorf("unset() = %v, want %v", got, tt.want)
			}
			rm := tt.fields.registryMock

			registryValidator(t, rm.getCalls, tt.want.registryGetCalls, "GET")
			registryValidator(t, rm.setCalls, tt.want.registrySetCalls, "SET")
			registryValidator(t, rm.setExpandCalls, &[]string{}, "EXPAND")
			registryValidator(t, rm.delCalls, tt.want.registryDelCalls, "DEL")
		})
	}
}

func TestCmdEnv_set(t *testing.T) {
	type fields struct {
		registryMock *RegistryKeyMock
		openKeyErr   error
	}
	type args struct {
		name  string
		value string
	}
	type want struct {
		returnValue      error
		registryGetCalls *[]string // nil means it should have no calls
		registrySetCalls *[]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			"set",
			fields{&RegistryKeyMock{}, nil},
			args{
				"key",
				"value",
			},
			want{
				nil,
				&[]string{},
				&[]string{"key=value", "!key_original"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CmdEnv{
				openKeyFn: func(path string) (osutils.RegistryKey, error) {
					return tt.fields.registryMock, tt.fields.openKeyErr
				},
			}
			if got := c.set(tt.args.name, tt.args.value); !reflect.DeepEqual(got, tt.want.returnValue) {
				t.Errorf("set() = %v, want %v", got, tt.want)
			}
			rm := tt.fields.registryMock

			registryValidator(t, rm.getCalls, tt.want.registryGetCalls, "GET")
			registryValidator(t, rm.setCalls, tt.want.registrySetCalls, "SET")
		})
	}
}

func TestCmdEnv_get(t *testing.T) {
	type fields struct {
		registryMock *RegistryKeyMock
		openKeyErr   error
	}
	type args struct {
		name string
	}
	type want struct {
		returnValue      string
		returnFailure    error
		registryGetCalls *[]string // nil means it should have no calls
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			"get nonexist",
			fields{&RegistryKeyMock{}, nil},
			args{
				"key",
			},
			want{
				"",
				nil,
				&[]string{"key"},
			},
		},
		{
			"get existing",
			fields{&RegistryKeyMock{
				getResults: map[string]RegistryValue{
					"key": RegistryValue{"value", nil},
				},
			}, nil},
			args{
				"key",
			},
			want{
				"value",
				nil,
				&[]string{"key"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CmdEnv{
				openKeyFn: func(path string) (osutils.RegistryKey, error) {
					return tt.fields.registryMock, tt.fields.openKeyErr
				},
			}
			got, gotFail := c.Get(tt.args.name)
			if !reflect.DeepEqual(got, tt.want.returnValue) {
				t.Errorf("get() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(gotFail, tt.want.returnFailure) {
				t.Errorf("get() err = %v, want %v", gotFail, tt.want)
			}

			rm := tt.fields.registryMock
			registryValidator(t, rm.getCalls, tt.want.registryGetCalls, "GET")
		})
	}
}

func registryValidator(t *testing.T, got []string, want *[]string, name string) {
	if want == nil && len(got) > 0 {
		t.Errorf("%s: registry should have no calls but got: %v", name, got)
		t.FailNow()
	}

	if want != nil {
		for _, v := range *want {
			exclude := strings.HasPrefix(v, "!")
			if exclude {
				v = strings.TrimPrefix(v, "!")
			}
			contains := funk.Contains(got, v)
			if exclude && contains {
				t.Errorf("%s: should not contain: %s, calls: %v", name, v, got)
			}
			if !exclude && !contains {
				t.Errorf("%s: should have contained: %s, calls: %v", name, v, got)
			}
		}
	}
}
