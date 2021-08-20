package updatemocks

import (
	"net/url"
)

func CreateRequestPath(branch, source, platform, version string) string {
	v := url.Values{}
	v.Set("channel", branch)
	v.Set("source", source)
	v.Set("platform", platform)
	if version != "" {
		v.Set("target-version", version)
	}

	return "/info/legacy?" + v.Encode()
}
