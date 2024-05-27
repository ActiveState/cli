package languages

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/model"
)

func Test_parseLanguage(t *testing.T) {
	type args struct {
		langName string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.Language
		wantErr bool
	}{
		{
			"Language with version",
			args{"Python@2"},
			&model.Language{Name: "Python", Version: "2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLanguage(tt.args.langName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLanguage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
