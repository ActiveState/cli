package track

import "gopkg.in/yaml.v3"

type Dir struct {
	Path string `yaml:"path"`
}

func NewDir(path string) *Dir {
	return &Dir{path}
}

func (f *Dir) Type() TrackingType {
	return DirType
}

type Dirs []*Dir

func (f Dirs) UnmarshalTrackable(value string) error {
	return yaml.Unmarshal([]byte(value), &f)
}
