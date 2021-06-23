package v1

import (
	"fmt"
	"time"
)

const (
	// apiServerTimeFormat - api-server time lacks the colon in timezone
	apiServerTimeFormat = "2006-01-02T15:04:05.000-0700"

	// apiServerTimeFormat_ - time with the colon in timezone
	apiServerTimeFormat_ = "2006-01-02T15:04:05.000-07:00"
)

// Time - time
type Time time.Time

// UnmarshalJSON - unmarshal json for time
func (t *Time) UnmarshalJSON(bytes []byte) error {
	tt, err := time.Parse(`"`+apiServerTimeFormat+`"`, string(bytes))

	if err == nil {
		*t = Time(tt)
		return nil
	}
	tt, err = time.Parse(`"`+apiServerTimeFormat_+`"`, string(bytes))
	if err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

// MarshalJSON -
func (t *Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(*t)

	timeStr := fmt.Sprintf("\"%s\"", tt.Format(apiServerTimeFormat))
	return []byte(timeStr), nil
}
