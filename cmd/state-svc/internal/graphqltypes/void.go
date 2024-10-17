package graphqltypes

import "io"

type Void struct{}

func (Void) MarshalGQL(w io.Writer) {
	_, _ = w.Write([]byte("null"))
}

func (v *Void) UnmarshalGQL(input interface{}) error {
	return nil
}
