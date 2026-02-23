# Okta Provisioning — PR Specification (agent-sdk)

This document describes the exact changes to implement in `agent-sdk` to support Okta group assignment and Authorization Server policy/rule creation/removal during dynamic client provisioning.

Goal: Add Okta post-registration and post-unregister hooks that use Okta Management API to assign app→group(s), create/delete OAuth scopes, and create/delete policies/rules. Persist created IDs in agent details for cleanup.

Files to change (high-level)
- `pkg/authz/oauth/provider.go` — invoke post-registration and post-unregister hooks after successful RegisterClient/UnregisterClient.
- `pkg/authz/oauth/provider.go` (types) — extend `idpType` interface with optional post-processing hooks.
- `pkg/authz/oauth/oktaprovider.go` — implement Okta-specific hooks; parse `extraProperties` for `group`/`groups` and `createPolicy` payload.
- Add new package: `pkg/authz/oauth/oktaapi` — small wrapper client for Okta Management API.
- `pkg/apic/provisioning/idp/provisioner.go` — no change required (SDK provisioner already uses idp provisioner). Tests/docs updated.

Provider changes (detailed)

1) Add new optional interface methods in `provider.go` (near existing `idpType` definition):

```go
type idpType interface {
    validateExtraProperties(extraProps map[string]interface{}) error
    preProcessClientRequest(clientRequest *clientMetadata)
    // new optional hooks:
    postProcessClientRegistration(clientRes ClientMetadata, extraProps map[string]interface{}, credentialObj interface{}) error
    postProcessClientUnregister(clientID string, agentDetails map[string]string, extraProps map[string]interface{}, credentialObj interface{}) error
}
```

2) In `provider.RegisterClient`, after successful unmarshal and logging and before returning, call the post hook if supported:

```go
clientRes := &clientMetadata{}
err = json.Unmarshal(response.Body, clientRes)
// ... existing handling ...
// invoke post-process hook
if idpImpl, ok := p.idpType.(interface{ postProcessClientRegistration(ClientMetadata, map[string]interface{}, interface{}) error }); ok {
    _ = idpImpl.postProcessClientRegistration(clientRes, p.extraProperties, credentialObj) // credentialObj can be nil or the CRD
}
return clientRes, err
```

Note: `credentialObj` is a generic parameter to allow storing agent details on the `Credential` resource — in our IDP provisioner path the credential object is available; where not available this param may be nil.

3) In `provider.UnregisterClient`, after the standard unregister operations (or on success), call `postProcessClientUnregister` with known `agentDetails` and `extraProperties` so the idpType can remove created policies/groups.

Okta provider changes (`oktaprovider.go`)
- Implement `postProcessClientRegistration` to:
  - Parse `extraProps` for `group` (string) or `groups` ([]string). Validate shapes.
  - Use `oktaapi` client to resolve group IDs: `FindGroupByName(name) (groupID, error)`.
    - If not found and `createGroup==true`, call `CreateGroup(name)`.
  - Assign groups to app: `AssignGroupToApp(appID, groupID)`.
  - If `createPolicy==true` (or `policyTemplate` provided): call `CreatePolicy(authServerID, policySpec)` returning `policyID`, then `CreateRule(authServerID, policyID, ruleSpec)` returning `ruleID`.
  - Persist created IDs into `agentDetails` using `util.SetAgentDetailsKey(credentialObj, "oktaGroupAssignments", jsonString)` and `oktaCreatedPolicies`.

- Implement `postProcessClientUnregister` to:
  - Read `oktaCreatedPolicies` and delete rules then policies via `oktaapi.DeleteRule` / `DeletePolicy`.
  - Read `oktaGroupAssignments` and unassign groups using `oktaapi.UnassignGroupFromApp`.

Okta API client (`pkg/authz/oauth/oktaapi`)
- Implement a minimal client using existing `coreapi` or simple `net/http` calls with headers set to `Authorization: SSWS {token}`.
- Expose methods:
  - `FindGroupByName(ctx, name) (groupID string, err error)`
  - `CreateGroup(ctx, name, description) (groupID string, err error)`
  - `AssignGroupToApp(ctx, appID, groupID) error`
  - `UnassignGroupFromApp(ctx, appID, groupID) error`
  - `CreateScope(ctx, authServerID, scopeSpec) (scopeID string, err error)`
  - `DeleteScope(ctx, authServerID, scopeID) error`
  - `CreatePolicy(ctx, authServerID, policySpec) (policyID string, err error)`
  - `CreateRule(ctx, authServerID, policyID, ruleSpec) (ruleID string, err error)`
  - `DeleteRule(ctx, authServerID, policyID, ruleID) error`
  - `DeletePolicy(ctx, authServerID, policyID) error`

Example `AssignGroupToApp` request:

```
POST /api/v1/apps/{appId}/groups
Body: { "id": "{groupId}" }
Header: Authorization: SSWS {TOKEN}
```

Detailed Okta Management API examples (illustrative):

- Create group:

  POST /api/v1/groups
  Body:

  ```json
  { "profile": { "name": "MyAppUsers", "description": "Auto-created group" } }
  ```

  Response: 201 Created with group `id`.

- Find group by name:

  GET /api/v1/groups?q=MyAppUsers

  Response: 200 OK with array of matching groups.

- Create scope on an authorization server:

  POST /api/v1/authorizationServers/{authServerId}/scopes
  Body (minimal):

  ```json
  { "name": "read:items", "displayName": "Read items", "description": "Read access to items" }
  ```

  Response: 201 Created with scope object (including `id`).

- Create policy:

  POST /api/v1/authorizationServers/{authServerId}/policies
  Body (simplified):

  ```json
  { "type": "OAUTH_AUTHORIZATION_POLICY", "name": "AutoPolicy-MyApp", "description": "Auto-created policy" }
  ```

- Create rule under a policy:

  POST /api/v1/authorizationServers/{authServerId}/policies/{policyId}/rules
  Body (simplified):

  ```json
  { "name": "AutoRule-MyApp", "conditions": { "grantTypes": { "include": ["authorization_code"] } }, "actions": {} }
  ```

Headers for all calls:

- `Authorization: SSWS {API_TOKEN}`
- `Content-Type: application/json`

Token guidance:

- The SDK must be given an Okta SSWS token that has admin privileges for the tenant. If `AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_<n>` is used, verify it's a management SSWS token; otherwise configure `AGENTFEATURES_IDP_MANAGEMENT_TOKEN_<n>` as a dedicated secret.

Persisted agent detail shape
- `oktaGroupAssignments` JSON: `[ { "groupId": "00g...", "groupName": "MyGroup" }, ... ]`
- `oktaCreatedPolicies` JSON: `[ { "authServerId": "default", "policyId": "pol-...", "ruleIds": ["r-..."] } ]`

**DCR verification steps (explicit)**

- **Prerequisites:** Okta dev org, SSWS management token, test `extraProperties` JSON containing `groups`/`createPolicy` fields, and access to the agent's credential resource or persisted agent details.
- **Step 1 — Provision (DCR + Okta post-hooks):**
  - Trigger a provisioning flow that causes the SDK to call `RegisterClient` (via v7 agent or direct SDK test harness) with `extraProperties` including `groups` and optional `createPolicy` payload.
  - Verify the DCR response contains `client_id` and (if applicable) `client_secret` in the `ClientMetadata` returned by the SDK.
  - Verify the SDK persisted Okta-created resource IDs in agent details under these keys:
    - `oktaGroupAssignments` — array of objects with `groupId` and `groupName`.
    - `oktaCreatedScopes` — array with `authServerId`, `scopeId`, `name`.
    - `oktaCreatedPolicies` — array with `authServerId`, `policyId`, and `ruleIds`.
  - Check Okta directly (API or Admin UI) that:
    - The app/client (by `client_id`) has the assigned groups.
    - Any requested authorization server policies/rules exist with returned IDs.

- **Step 2 — Reprovision / Idempotency checks:**
  - Re-run the same provisioning request. Expect the SDK/Okta to treat duplicate-create scenarios idempotently:
    - Creating a group that already exists should return a 409/duplicate-or-resolve the existing group; SDK should interpret this as success.
    - Creating a policy/rule that exists should not produce duplicates; if behavior differs, the SDK must detect and persist the existing IDs rather than creating new ones.

- **Step 3 — Deprovision (Unregister + cleanup):**
  - Trigger deprovisioning that calls `UnregisterClient` in the SDK.
  - Verify the SDK calls `oktaapi.DeleteRule` / `DeletePolicy` for each persisted `oktaCreatedPolicies` entry, and `oktaapi.UnassignGroupFromApp` for each `oktaGroupAssignments` entry.
  - Confirm that the Okta resources were removed/unassigned and that the agent details are cleared or marked as cleaned.

- **Step 4 — Failure & partial cleanup handling:**
  - Simulate partial failures (e.g., policy creation succeeded but group assignment failed). Verify that:
    - The SDK persists the resource IDs that did succeed.
    - The error is surfaced so operators can retry or perform manual cleanup.

- **Step 5 — Unit & Mock Assertions:**
  - Unit tests should assert that the `oktaapi` client is invoked in this order for a typical provisioning: `FindGroupByName` → `CreateGroup` (if needed) → `AssignGroupToApp` → `CreatePolicy` → `CreateRule`.
  - For deprovision: assert `DeleteRule` → `DeletePolicy` → `UnassignGroupFromApp` (order flexible but must be exercised).

- **Where to inspect persisted values:**
  - In the SDK test harness or v7 agent environment, inspect the Credential resource agent details or logs where `util.SetAgentDetailsKey` stores `oktaGroupAssignments` and `oktaCreatedPolicies`.


Config/secrets
- New recommended env var per IDP config: `AGENTFEATURES_IDP_MANAGEMENT_TOKEN_<n>` (or instruct that `AGENTFEATURES_IDP_AUTH_ACCESSTOKEN_<n>` must be an Okta management token). Document in docs.

Tests
- Unit tests for `oktaprovider.postProcessClientRegistration` and `postProcessClientUnregister` using a mocked `oktaapi` client that simulates API responses and errors.
- Integration test (manual/runbook): provision against an Okta dev org and verify group assignment, policy creation, and cleanup.

Docs & changelog
- Update `docs/discovery/provisioning.md` with examples of `AGENTFEATURES_IDP_EXTRAPROPERTIES` including `groups`, `createGroup`, `createPolicy`, and `authServerId` usage.
- Add an entry to `agent-sdk` changelog describing the new optional Okta hooks.

Implementation notes & edge cases
- Idempotency: treat 409/duplicate responses as success for create/assign ops.
- If policy creation succeeds but group assignment fails, persist what succeeded and surface an error so operator can retry/cleanup.
- Use sensible timeouts and retries for Okta calls.

Ready-to-apply PR structure
- `pkg/authz/oauth/provider.go` (small edits to call hooks)
- `pkg/authz/oauth/oktaprovider.go` (add hooks implementation)
- `pkg/authz/oauth/oktaapi/*` (new client files)
- `pkg/authz/oauth/oktaprovider_test.go` (unit tests)
- `docs/*` updates (examples + helm env var docs)

If you want I can now scaffold the exact code diffs for the PR (implementing hooks and `oktaapi` client). Confirm and I'll prepare the code patches and unit test stubs.
