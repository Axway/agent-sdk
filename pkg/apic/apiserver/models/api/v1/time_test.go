package v1

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnmarshall(t *testing.T) {
	time := &Time{}
	err := time.UnmarshalJSON([]byte(`"2006-01-02T15:04:05.000+0700"`))

	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	time = &Time{}
	err = time.UnmarshalJSON([]byte("abc"))

	if err == nil {
		t.Fatalf("Expected time.UnmarshalJSON to throw an error")
	}
}

func TestMarshall(t *testing.T) {

	time := &Time{}
	in := `"2006-01-02T15:04:05.000+0700"`

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
