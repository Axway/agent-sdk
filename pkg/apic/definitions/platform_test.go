package definitions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformResponseUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    any
		wantErr bool
	}{
		{
			name: "array of platform teams",
			data: `{
				"success": true,
				"result": [
					{"guid": "1", "name": "team1", "tags": ["a"], "default": true},
					{"guid": "2", "name": "team2", "tags": [], "default": false}
				]
			}`,
			want: []PlatformTeam{
				{ID: "1", Name: "team1", Tags: []string{"a"}, Default: true},
				{ID: "2", Name: "team2", Tags: []string{}, Default: false},
			},
		},
		{
			name: "single platform team",
			data: `{
				"success": true,
				"result": {"guid": "1", "name": "team1", "tags": ["a"], "default": true}
			}`,
			want: PlatformTeam{ID: "1", Name: "team1", Tags: []string{"a"}, Default: true},
		},
		{
			name: "org entitlements",
			data: `{
				"success": true,
				"result": {"entitlements": {"feature1": true, "feature2": 5}}
			}`,
			want: OrgEntitlements{Entitlements: map[string]interface{}{"feature1": true, "feature2": float64(5)}},
		},
		{
			name: "null result",
			data: `{"success": true, "result": null}`,
			want: nil,
		},
		{
			name:    "unrecognized object shape",
			data:    `{"success": true, "result": {"unknown": "field"}}`,
			wantErr: true,
		},
		{
			name:    "unrecognized array shape",
			data:    `{"success": true, "result": [{"unknown": "field"}]}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var res PlatformResponse
			err := res.UnmarshalJSON([]byte(tc.data))
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, res.Result)
		})
	}
}