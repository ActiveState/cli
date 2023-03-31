package track

import "gopkg.in/yaml.v3"

type File struct {
	Path string `yaml:"path"`
}

func NewFile(path string) *File {
	return &File{path}
}

func (f *File) Type() TrackingType {
	return FileType
}

func (f *File) UnmarshalTrackable(value string) error {
	return yaml.Unmarshal([]byte(value), f)
}

type Files []*File

func (f Files) UnmarshalTrackable(value string) error {
	return yaml.Unmarshal([]byte(value), &f)
}
