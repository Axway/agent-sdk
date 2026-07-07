package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

const (
	removeClientID    = "remove-me"
	errForbidden      = "returns error on forbidden"
	nilOn200          = "returns nil on 200"
	nilOn204          = "returns nil on 204"
	notFoundAsSuccess = "treats 404 as success"
	testPolicyName    = "my-policy"
	testClientID      = "client1"
	testRuleName      = "rule-1"
	testGrantType     = "client_credentials"
	testRuleScope     = "read:api"
	testGroupID       = "grp-123"
	testGroupName     = "Marketplace"
	testAppID         = "app-abc"
	testAuthServerID  = "as1"
	testPolicyID      = "pol1"
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
		errForbidden:    {code: http.StatusForbidden, wantErr: true},
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
		errForbidden:                {code: http.StatusForbidden, body: `forbidden`, wantErr: true},
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
		errForbidden: {
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

func TestOktaActivatePolicy(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn200:          {code: http.StatusOK, wantErr: false},
		nilOn204:          {code: http.StatusNoContent, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:      {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.ActivatePolicy(testAuthServerID, testPolicyID), tc.wantErr)
		})
	}
}

func TestOktaPolicyHasRuleForScope(t *testing.T) {
	cases := map[string]struct {
		code    int
		body    string
		scope   string
		wantHas bool
		wantErr bool
	}{
		"returns true when rule contains scope": {
			code:    http.StatusOK,
			body:    fmt.Sprintf(`[{"conditions":{"scopes":{"include":[%q,"write:api"]}}}]`, testRuleScope),
			scope:   testRuleScope,
			wantHas: true,
		},
		"returns false when rules list is empty": {
			code:    http.StatusOK,
			body:    `[]`,
			scope:   testRuleScope,
			wantHas: false,
		},
		"returns false when no rule contains scope": {
			code:    http.StatusOK,
			body:    `[{"conditions":{"scopes":{"include":["write:api"]}}}]`,
			scope:   testRuleScope,
			wantHas: false,
		},
		"trims whitespace when matching scope": {
			code:    http.StatusOK,
			body:    fmt.Sprintf(`[{"conditions":{"scopes":{"include":[" %s "]}}}]`, testRuleScope),
			scope:   testRuleScope,
			wantHas: true,
		},
		errForbidden: {
			code:    http.StatusForbidden,
			body:    `forbidden`,
			wantErr: true,
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			})
			defer close()
			got, err := client.PolicyHasRuleForScope(testAuthServerID, testPolicyID, tc.scope)
			assertOktaErr(t, err, tc.wantErr)
			if !tc.wantErr {
				assert.Equal(t, tc.wantHas, got)
			}
		})
	}
}

func TestOktaDeactivatePolicy(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn200:          {code: http.StatusOK, wantErr: false},
		nilOn204:          {code: http.StatusNoContent, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:      {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.DeactivatePolicy(testAuthServerID, testPolicyID), tc.wantErr)
		})
	}
}

func TestOktaDeletePolicy(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn204:          {code: http.StatusNoContent, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:      {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.DeletePolicy(testAuthServerID, testPolicyID), tc.wantErr)
		})
	}
}

func TestOktaDeactivateApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn200:    {code: http.StatusOK, wantErr: false},
		nilOn204:    {code: http.StatusNoContent, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:        {code: http.StatusForbidden, wantErr: true},
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

func TestOktaFindGroupByName(t *testing.T) {
	cases := map[string]struct {
		responseCode int
		responseBody string
		wantID       string
		wantErr      bool
	}{
		"returns group ID when found": {
			responseCode: http.StatusOK,
			responseBody: `[{"id":"grp-123","profile":{"name":"Marketplace"}},{"id":"grp-999","profile":{"name":"MarketplaceOther"}}]`,
			wantID:       testGroupID,
		},
		"returns empty string when no exact match": {
			responseCode: http.StatusOK,
			responseBody: `[{"id":"grp-999","profile":{"name":"MarketplaceOther"}}]`,
			wantID:       "",
		},
		"returns empty string on empty list": {
			responseCode: http.StatusOK,
			responseBody: `[]`,
			wantID:       "",
		},
		errForbidden: {
			responseCode: http.StatusForbidden,
			responseBody: `forbidden`,
			wantErr:      true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseCode)
				_, _ = w.Write([]byte(tc.responseBody))
			})
			defer close()
			got, err := client.FindGroupByName(testGroupName)
			assertOktaErr(t, err, tc.wantErr)
			if !tc.wantErr {
				assert.Equal(t, tc.wantID, got)
			}
		})
	}
}

func TestOktaAssignGroupToApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn200:   {code: http.StatusOK, wantErr: false},
		"returns nil on 201":   {code: http.StatusCreated, wantErr: false},
		errForbidden:       {code: http.StatusForbidden, wantErr: true},
		"returns error on 404": {code: http.StatusNotFound, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.AssignGroupToApp(testAppID, testGroupID), tc.wantErr)
		})
	}
}

func TestOktaUnassignGroupFromApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn204:    {code: http.StatusNoContent, wantErr: false},
		nilOn200:    {code: http.StatusOK, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:        {code: http.StatusForbidden, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client, close := newTestOktaClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			})
			defer close()
			assertOktaErr(t, client.UnassignGroupFromApp(testAppID, testGroupID), tc.wantErr)
		})
	}
}

func TestOktaDeleteApp(t *testing.T) {
	cases := map[string]struct {
		code    int
		wantErr bool
	}{
		nilOn204:    {code: http.StatusNoContent, wantErr: false},
		notFoundAsSuccess: {code: http.StatusNotFound, wantErr: false},
		errForbidden:        {code: http.StatusForbidden, wantErr: true},
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
