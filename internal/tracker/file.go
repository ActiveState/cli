package tracker

type File struct {
	key  string
	path string
}

func NewFile(key, path string) File {
	return File{key, path}
}

func (f File) Type() TrackingType {
	return Files
}

func (f File) Key() string {
	return f.key
}

func (f File) Value() string {
	return f.path
}
