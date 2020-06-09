package v1

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshall(t *testing.T) {
	// TODO test error cases
	testCases := []struct {
		name     string
		input    []byte
		expected error
	}{
		{
			"correct format",
			[]byte(`"2006-01-02T15:04:05+0700"`),
			nil,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		time := &Time{}
		err := time.UnmarshalJSON(tc.input)

		if err != tc.expected {
			t.Fatalf("%s: expected err: %s got %s", tc.name, tc.expected, err)
		}
	}
}

func TestMarshall(t *testing.T) {
	time := &Time{}
	in := `"2006-01-02T15:04:05+0700"`

	err := time.UnmarshalJSON([]byte(in))
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	outB, err := time.MarshalJSON()
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	out := string(outB)

	if in != out {
		t.Fatalf("Expected %s got %s", in, out)
	}
}

type testStruct struct{ Time Time }

func Test(t *testing.T) {
	out, err := json.Marshal(testStruct{Time(time.Now())})
	if err != nil {
		t.Fatalf("Failed due: %s", err)
	}

	t.Log(string(out))
}
