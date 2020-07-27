# APIC Agents SDK Errors

| Code | Description                                                                        | Code Path                      |
| ---- | ---------------------------------------------------------------------------------- | ------------------------------ |
| 1000 | general configuration error in CENTRAL                                             | pkg/apic/ErrCentralConfig      |
| 1010 | connection to AMPLIFY Central failed, possibly network                             | pkg/apic/ErrNetwork            |
| 1020 | request to get authentication token failed, possibly network or CENTAL_AUTH config | pkg/apic/ErrAuthenticationCall |
| 1021 | token retrieved but was invlaid on request to Central, likely CENTRAL_AUTH config  | pkg/apic/ErrAuthentication     |
| 1030 | request to Central failed, could be bad value for CENTRAL_ENVIRONMENT              | pkg/apic/ErrEnvironmentQuery   |
