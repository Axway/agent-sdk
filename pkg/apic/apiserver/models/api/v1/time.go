package v1

import (
	"time"
)

const (
	// RFC3339Z api-server time lacks the colon in timezone
	RFC3339Z = "2006-01-02T15:04:05Z0700"
)

// Time -
type Time time.Time

// UnmarshalJSON -
func (t *Time) UnmarshalJSON(bytes []byte) error {
	tt, err := time.Parse(`"`+RFC3339Z+`"`, string(bytes))

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
