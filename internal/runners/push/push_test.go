package push

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

type configMock struct {
	set map[string]string
}

func (c *configMock) GetString(key string) string {
	if v, found := c.set[key]; found {
		return v
	}
	return ""
}

func TestPush_languageForPath(t *testing.T) {
	tests := []struct {
		name          string
		inputConfig   *configMock
		inputPath     string
		wantSupported string
		wantVersion   string
		wantFailure   *failures.Failure
	}{
		{
			"nothing stored",
			&configMock{},
			fileutils.TempDirUnsafe(),
			"",
			"",
			nil,
		},
		{
			"language",
			&configMock{},
			fileutils.TempDirUnsafe(),
			"python3",
			"",
			nil,
		},
		{
			"language & version",
			&configMock{},
			fileutils.TempDirUnsafe(),
			"python3",
			"1.0",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.inputConfig.set = map[string]string{}
			if tt.wantSupported != "" {
				tt.inputConfig.set[tt.inputPath+"_language"] = tt.wantSupported
			}
			if tt.wantVersion != "" {
				tt.inputConfig.set[tt.inputPath+"_language_version"] = tt.wantVersion
			}
			r := &Push{
				config: tt.inputConfig,
			}
			got, got1, got2 := r.languageForPath(tt.inputPath)
			if got != nil && !reflect.DeepEqual(got.String(), tt.wantSupported) {
				t.Errorf("Push.languageForPath() got = %v, want %v", got, tt.wantSupported)
			}
			if got1 != tt.wantVersion {
				t.Errorf("Push.languageForPath() got1 = %v, want %v", got1, tt.wantVersion)
			}
			if !reflect.DeepEqual(got2, tt.wantFailure) {
				t.Errorf("Push.languageForPath() got2 = %v, want %v", got2, tt.wantFailure)
			}
		})
	}
}
