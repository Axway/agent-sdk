package clients

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
)

func TestOktaAPIStatusHandling(t *testing.T) {
	type testCase struct {
		name           string
		handler        http.HandlerFunc
		call           func(client *Okta) error
		wantErr        bool
		expectedMethod string
	}

	cases := []testCase{
		{
			name:           "AssignGroupToApp returns error on forbidden",
			expectedMethod: http.MethodPut,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("forbidden"))
			},
			call: func(client *Okta) error {
				return client.AssignGroupToApp("app123", "group456")
			},
			wantErr: true,
		},
		{
			name:           "AssignGroupToApp treats conflict as success",
			expectedMethod: http.MethodPut,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("already assigned"))
			},
			call: func(client *Okta) error {
				return client.AssignGroupToApp("app123", "group456")
			},
			wantErr: false,
		},
		{
			name:           "DeletePolicy treats not found as success",
			expectedMethod: http.MethodDelete,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("not found"))
			},
			call: func(client *Okta) error {
				return client.DeletePolicy("as1", "pol1")
			},
			wantErr: false,
		},
		{
			name:           "DeleteRule returns error on forbidden",
			expectedMethod: http.MethodDelete,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("forbidden"))
			},
			call: func(client *Okta) error {
				return client.DeleteRule("as1", "pol1", "rule1")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.expectedMethod != "" && r.Method != tc.expectedMethod {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				tc.handler(w, r)
			}))
			defer ts.Close()

			client := New(coreapi.NewClient(nil, ""), ts.URL, "token")
			err := tc.call(client)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
