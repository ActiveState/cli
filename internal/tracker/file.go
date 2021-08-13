package tracker

type File struct {
	Path  string
	label string
}

func (f *File) Type() TrackingType {
	return Files
}

func (f *File) Label() string {
	return f.label
}
