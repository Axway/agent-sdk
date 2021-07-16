package v1

import (
	"time"
)

const (
	// APIServerTimeFormat - api-server time lacks the colon in timezone
	APIServerTimeFormat = "2006-01-02T15:04:05.000-0700"

	// APIServerTimeFormatAlt - api-server time with colon in timezone
	APIServerTimeFormatAlt = "2006-01-02T15:04:05.000-07:00"
)

// Time - time
type Time time.Time

// UnmarshalJSON - unmarshal json for time
func (t *Time) UnmarshalJSON(bytes []byte) error {
	tt, err := time.Parse(`"`+APIServerTimeFormat+`"`, string(bytes))

	if err == nil {
		*t = Time(tt)
		return nil
	}
	tt, err = time.Parse(`"`+APIServerTimeFormatAlt+`"`, string(bytes))
	if err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

// MarshalJSON -
func (t Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	b := make([]byte, 0, len(APIServerTimeFormat)+2)
	b = append(b, '"')
	b = tt.AppendFormat(b, APIServerTimeFormat)
	b = append(b, '"')
	return b, nil
}
