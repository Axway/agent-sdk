package clients

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

const (
	removeClientID   = "remove-me"
	caseErrForbidden = "returns error on forbidden"
	testPolicyName   = "my-policy"
	testClientID     = "client1"
	testRuleName     = "rule-1"
	testGrantType    = "client_credentials"
	testRuleScope    = "read:api"
)

func newTestOktaClient(t *testing.T, handler http.HandlerFunc) (*Okta, func()) {
	t.Helper()
	ts := httptest.NewServer(handler)
	client := New(coreapi.NewClient(nil, ""), ts.URL, "token")
	return client, ts.Close
}

func assertOktaErr(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr && err == nil {
		t.Fatal("expected an error but got nil")
		return
	}
	if !wantErr && err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
}

func TestOktaUpdatePolicy(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		caseErrForbidden:    {code: http.StatusForbidden, wantErr: true},
		"returns nil on ok": {code: http.StatusOK, wantErr: false},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.UpdatePolicy("as1", "pol1", map[string]interface{}{"id": "pol1"}), tc.wantErr)
		})
	}
}

func TestOktaCreatePolicy(t *testing.T) {
	cases := map[string]struct {
		code    int
		body    string
		wantErr bool
	}{
		"returns created policy on 201": {code: http.StatusCreated, body: `{"id":"pol-new","name":"my-policy"}`, wantErr: false},
		caseErrForbidden:                {code: http.StatusForbidden, body: `forbidden`, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			})
			defer close()
			_, err := client.CreatePolicy("as1", testPolicyName, testClientID)
			assertOktaErr(t, err, tc.wantErr)
		})
	}
}

func TestOktaCreatePolicyRequestBody(t *testing.T) {
	var captured []byte
	client, teardown := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"pol1"}`))
	})
	defer teardown()
	_, err := client.CreatePolicy("as1", testPolicyName, testClientID)
	assertOktaErr(t, err, false)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(captured, &body))
	assert.Equal(t, "OAUTH_AUTHORIZATION_POLICY", body["type"])
	assert.Equal(t, "ACTIVE", body["status"])
	assert.EqualValues(t, 1, body["priority"])

	conditions, _ := body["conditions"].(map[string]interface{})
	clients, _ := conditions["clients"].(map[string]interface{})
	include, _ := clients["include"].([]interface{})
	assert.Len(t, include, 1)
	assert.Equal(t, testClientID, include[0])
	for _, v := range include {
		assert.NotEqual(t, "ALL_CLIENTS", v)
	}
}

func TestOktaCreatePolicyRule(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		"returns nil on 201":       {code: http.StatusCreated, wantErr: false},
		"returns error on bad req": {code: http.StatusBadRequest, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			err := client.CreatePolicyRule("as1", "pol1", "rule", testGrantType, testRuleScope)
			assertOktaErr(t, err, tc.wantErr)
		})
	}
}

func TestOktaCreatePolicyRuleRequestBody(t *testing.T) {
	var captured []byte
	client, teardown := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	})
	defer teardown()
	err := client.CreatePolicyRule("as1", "pol1", testRuleName, testGrantType, testRuleScope)
	assertOktaErr(t, err, false)

	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(captured, &body))
	assert.Equal(t, "RESOURCE_ACCESS", body["type"])

	conds, _ := body["conditions"].(map[string]interface{})
	people, _ := conds["people"].(map[string]interface{})
	groups, _ := people["groups"].(map[string]interface{})
	groupIncludes, _ := groups["include"].([]interface{})
	assert.Equal(t, []interface{}{"EVERYONE"}, groupIncludes)

	grantTypes, _ := conds["grantTypes"].(map[string]interface{})
	grantIncludes, _ := grantTypes["include"].([]interface{})
	assert.Contains(t, grantIncludes, testGrantType)

	scopes, _ := conds["scopes"].(map[string]interface{})
	scopeIncludes, _ := scopes["include"].([]interface{})
	assert.Contains(t, scopeIncludes, testRuleScope)

	actions, _ := body["actions"].(map[string]interface{})
	token, _ := actions["token"].(map[string]interface{})
	assert.EqualValues(t, 60, token["accessTokenLifetimeMinutes"])
}

func TestOktaRemoveClientFromPolicy(t *testing.T) {
	policyWith := func(include ...interface{}) map[string]interface{} {
		return map[string]interface{}{
			"id": "pol1",
			"conditions": map[string]interface{}{
				"clients": map[string]interface{}{
					"include": include,
				},
			},
		}
	}

	cases := map[string]struct {
		code    int
		policy  map[string]interface{}
		wantErr bool
		wantPUT bool
	}{
		"returns nil on ok": {
			code:    http.StatusOK,
			policy:  policyWith("keep-me", removeClientID),
			wantErr: false,
			wantPUT: true,
		},
		caseErrForbidden: {
			code:    http.StatusForbidden,
			policy:  policyWith("keep-me", removeClientID),
			wantErr: true,
			wantPUT: true,
		},
		"client not in list no PUT called": {
			code:    http.StatusOK,
			policy:  policyWith("other-client"),
			wantErr: false,
			wantPUT: false,
		},
		"empty include after removal policy not deleted": {
			code:    http.StatusOK,
			policy:  policyWith(removeClientID),
			wantErr: false,
			wantPUT: true,
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			putCalled := false
			deleteCalled := false
			client, teardown := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPut:
					putCalled = true
					w.WriteHeader(tc.code)
				case http.MethodDelete:
					deleteCalled = true
					w.WriteHeader(http.StatusNoContent)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			})
			defer teardown()
			assertOktaErr(t, client.RemoveClientFromPolicy("as1", tc.policy, removeClientID), tc.wantErr)
			assert.Equal(t, tc.wantPUT, putCalled, "PUT call expectation mismatch")
			assert.False(t, deleteCalled, "policy must never be deleted")
		})
	}
}

func TestOktaDeactivateApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		"returns nil on 200":    {code: http.StatusOK, wantErr: false},
		"returns nil on 204":    {code: http.StatusNoContent, wantErr: false},
		"treats 404 as success": {code: http.StatusNotFound, wantErr: false},
		caseErrForbidden:        {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.DeactivateApp("app1"), tc.wantErr)
		})
	}
}

func TestOktaDeleteApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		"returns nil on 204":    {code: http.StatusNoContent, wantErr: false},
		"treats 404 as success": {code: http.StatusNotFound, wantErr: false},
		caseErrForbidden:        {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.DeleteApp("app1"), tc.wantErr)
		})
	}
}
