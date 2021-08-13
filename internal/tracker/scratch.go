package tracker

func scratch() {
	t, _ := New()
	ts, _ := t.Get(Files)
	for _, t := range ts {
		switch t.Type() {
		case Files:
			f := File{}
			t.Unmarshal(f)
			// Then use f
		}
	}
}
