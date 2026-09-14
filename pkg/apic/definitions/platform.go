package definitions

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type PlatformResponseMetadata struct {
	Count   int `json:"count"`
	Matched int `json:"matched"`
	Skip    int `json:"skip"`
	Page    int `json:"page"`
	Pages   int `json:"pages"`
}

// PlatformTeam - represents team from Central Client registry
type PlatformTeam struct {
	ID      string   `json:"guid"`
	Name    string   `json:"name"`
	Tags    []string `json:"tags"`
	Default bool     `json:"default"`
}

// OrgEntitlements contains only the entitlements map for an org.
// Values can be numbers, booleans, or arrays.
type OrgEntitlements struct {
	Entitlements map[string]interface{} `json:"entitlements"`
}

// PlatformAPIResponse
type PlatformResponse struct {
	Success  bool                     `json:"success"`
	Result   any                      `json:"result"`
	Metadata PlatformResponseMetadata `json:"_metadata"`
}

func (r *PlatformResponse) UnmarshalJSON(data []byte) error {
	// capture Result as raw JSON so we can decide its concrete type
	type alias PlatformResponse
	aux := struct {
		Result json.RawMessage `json:"result"`
		*alias
	}{
		alias: (*alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	result := bytes.TrimSpace(aux.Result)
	if len(result) == 0 || string(result) == "null" {
		r.Result = nil
		return nil
	}

	var res any
	var err error

	switch result[0] {
	case '[':
		res, err = unmarshalArrayResult(result)
	case '{':
		res, err = unmarshalObjectResult(result)
	default:
		return fmt.Errorf("unexpected result type in platform response")
	}

	if err != nil {
		return err
	}
	r.Result = res

	return nil
}

// unmarshalArrayResult decodes a JSON array result
func unmarshalArrayResult(data []byte) (any, error) {
	var teams []PlatformTeam
	if err := decodeStrict(data, &teams); err == nil {
		return teams, nil
	}

	return nil, fmt.Errorf("could not decode the returned result into a known type")
}

// unmarshalObjectResult decodes a JSON object result
func unmarshalObjectResult(data []byte) (any, error) {
	var entitlements OrgEntitlements
	if err := decodeStrict(data, &entitlements); err == nil {
		return entitlements, nil
	}

	var platformTeam PlatformTeam
	if err := decodeStrict(data, &platformTeam); err == nil {
		return platformTeam, nil
	}

	return nil, fmt.Errorf("could not decode the returned result into a known type")
}

// decodeStrict unmarshals data into v, rejecting unknown fields
func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
