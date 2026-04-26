package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdPRegistryIDPResourceName(t *testing.T) {
	const (
		metadataURL  = "https://idp.example.com/.well-known/openid-configuration"
		resourceName = "my-idp-resource"
	)

	tests := map[string]struct {
		lookupURL     string
		preSet        bool
		expectedName  string
		expectedFound bool
	}{
		"not found before set": {
			lookupURL:     metadataURL,
			preSet:        false,
			expectedName:  "",
			expectedFound: false,
		},
		"found after set": {
			lookupURL:     metadataURL,
			preSet:        true,
			expectedName:  resourceName,
			expectedFound: true,
		},
		"different URL not found": {
			lookupURL:     "https://other.example.com/",
			preSet:        true,
			expectedName:  "",
			expectedFound: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			reg := NewIdpRegistry()
			if tc.preSet {
				reg.SetIDPResourceName(metadataURL, resourceName)
			}

			got, ok := reg.GetIDPResourceName(tc.lookupURL)
			assert.Equal(t, tc.expectedFound, ok)
			assert.Equal(t, tc.expectedName, got)
		})
	}
}
