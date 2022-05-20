package registration

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Scopes - type for serializing scopes in client representation
type Scopes []string

// MarshalJSON - serializes the scopes in array as space separated string
func (s *Scopes) MarshalJSON() ([]byte, error) {
	scope := strings.Join([]string(*s), " ")
	return json.Marshal(scope)
}

// UnmarshalJSON - deserializes the scopes from space separated string to array
func (s *Scopes) UnmarshalJSON(data []byte) error {
	strScopes := string(data)
	strScopes, _ = strconv.Unquote(strScopes)
	scopes := strings.Split(strScopes, " ")

	for _, scope := range scopes {
		*s = append([]string(*s), scope)
	}
	return nil
}
