package download

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
)

func Test_parseS3URL(t *testing.T) {
	tests := []struct {
		url     string
		want    s3Meta
		wantErr bool
	}{
		{
			"https://bucket-name.s3.region.amazonaws.com/key-name/extra",
			s3Meta{
				"bucket-name",
				"region",
				"/key-name/extra",
			},
			false,
		},
		{
			"https://bucket-name.s3.amazonaws.com/key-name/extra",
			s3Meta{
				"bucket-name",
				constants.DefaultS3Region,
				"/key-name/extra",
			},
			false,
		},
		{
			"https://s3.region.amazonaws.com/bucket-name/key-name/extra",
			s3Meta{
				"bucket-name",
				"region",
				"/key-name/extra",
			},
			false,
		},
		{
			"https://my.site.com/key-name/extra",
			s3Meta{},
			true,
		},
	}
	for _, tt := range tests {
		url, err := url.Parse(tt.url)
		if err != nil {
			t.Errorf("Invalid URL: %v", err)
			return
		}
		t.Run(url.Host, func(t *testing.T) {
			got, err := parseS3URL(url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseS3URL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseS3URL() got = %v, want %v", got, tt.want)
			}
		})
	}
}
