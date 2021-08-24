package tracker

type Directory struct {
	key  string
	path string
}

func NewDirectory(key, path string) Directory {
	return Directory{key, path}
}

func (d Directory) Type() TrackingType {
	return Directories
}

func (d Directory) Key() string {
	return d.key
}

func (d Directory) Value() string {
	return d.path
}
