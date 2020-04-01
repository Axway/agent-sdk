package wsdl

import (
	"encoding/xml"
)

// Unmarshal unmarshals WSDL documents starting from the <definitions> tag.
//
// The Definitions object it returns is an unmarshalled version of the
// WSDL XML that can be introspected to generate the Web Services API.
func Unmarshal(bytes []byte) (*Definitions, error) {
	var d Definitions
	// decoder := xml.NewDecoder(r)
	// decoder.CharsetReader = charset.NewReaderLabel
	err := xml.Unmarshal(bytes, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
