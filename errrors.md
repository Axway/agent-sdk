# APIC Agents SDK Errors

| Code | Description                                                                                                 | Code Path                               |
| ---- | ----------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| 1001 | timeout error checking for dependencies to respond, possibly network or settings                            | pkg/util/errors/ErrServicesNotReady     |
|      |                                                                                                             |                                         |
| 1100 | general configuration error in CENTRAL                                                                      | pkg/apic/ErrCentralConfig               |
| 1101 | error attempting to query for ENVIRONMENT, check CENTRAL_ENVIRONMENT                                        | pkg/apic/ErrEnvironmentQuery            |
| 1110 | connection to AMPLIFY Central failed, possibly network                                                      | pkg/apic/ErrNetwork                     |
| 1120 | request to get authentication token failed, possibly network or CENTAL_AUTH config                          | pkg/apic/ErrAuthenticationCall          |
| 1130 | request to Central failed, could be bad value for CENTRAL_ENVIRONMENT                                       | pkg/apic/ErrEnvironmentQuery            |
| 1131 | token retrieved but was invalid on request to Central, likely CENTRAL_AUTH config                           | pkg/apic/ErrAuthentication              |
| 1140 | couldn't find a subscriber email address based on the ID in teh subscription event                          | pkg/apic/ErrNoAddressFound              |
| 1141 | couldn't contact AMPLIFY Central for subscription, possible network error                                   | pkg/apic/ErrSubscriptionQuery           |
| 1142 | couldn't get subscription data from AMPLIFY Central, check network and CENTRAL_AUTH                         | pkg/apic/ErrSubscriptionResp            |
| 1143 | couldn't create or update subscription schema data, possible Network error                                  | pkg/apic/ErrSubscriptionSchemaCreate    |
| 1144 | unexpected response when managing subscription schema on AMPLIFY Central, check network and CENTRAL_AUTH    | pkg/apic/ErrSubscriptionSchemaResp      |
|      |                                                                                                             |                                         |
| 1300 | error communicating with server for subscription notifications (SMTP or webhook), check SUBSCRIPTION config | pkg/apic/ErrSubscriptionNotification    |
| 1301 | subscription notifications not configured, check SUBSCRIPTION config                                        | pkg/apic/ErrSubscriptionNoNotifications |
| 1302 | error creating data for sending subscription notification                                                   | pkg/apic/ErrSubscriptionData            |
