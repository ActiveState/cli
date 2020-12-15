package model

import (
	"time"
)

const (
	ISO8601LocalTime = "2006-01-02T15:04:05"
)

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	var err error
	t.Time, err = time.Parse(`"`+ISO8601LocalTime+`"`, string(data))
	return err
}

func (t *Time) MarshalJSON() ([]byte, error) {
	d := make([]byte, 0, len(ISO8601LocalTime)+2)
	d = append(d, '"')
	d = t.Time.AppendFormat(d, ISO8601LocalTime)
	d = append(d, '"')
	return d, nil
}
