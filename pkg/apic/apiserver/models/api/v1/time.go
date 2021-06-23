package v1

import (
	"time"
)

const (
	// RFC3339Z api-server time lacks the colon in timezone
	RFC3339Z = "2006-01-02T15:04:05.000Z0700"

	// RFC3339Z_ time with the colon in timezone
	RFC3339Z_ = "2006-01-02T15:04:05.000Z07:00"
)

// Time - time
type Time time.Time

// UnmarshalJSON - unmarshal json for time
func (t *Time) UnmarshalJSON(bytes []byte) error {
	tt, err := time.Parse(`"`+RFC3339Z+`"`, string(bytes))

	if err == nil {
		*t = Time(tt)
		return nil
	}
	tt, err = time.Parse(`"`+RFC3339Z_+`"`, string(bytes))
	if err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

// MarshalJSON -
func (t Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)

	b := make([]byte, 0, len(RFC3339Z)+2)
	b = append(b, '"')
	b = tt.AppendFormat(b, RFC3339Z)
	b = append(b, '"')
	return b, nil
}
