package tracker

type Directory struct {
	Path  string
	label string
}

func (d *Directory) Type() TrackingType {
	return Directories
}

func (d *Directory) Label() string {
	return d.label
}
