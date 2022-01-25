# Amplify Agents SDK Errors

## Use code 1000s for SDK

| Code | Description                                                                                                 | Code Path                                           |
|------|-------------------------------------------------------------------------------------------------------------|-----------------------------------------------------|
|      | 1000-1099 - for general agent errors                                                                        |                                                     |
| 1000 | Unsupported agent type                                                                                      | pkg/agent/ErrUnsupportedAgentType                   |
| 1001 | initialization error checking for dependencies to respond, possibly network or settings                     | pkg/util/errors/ErrInitServicesNotReady             |
| 1002 | timeout error checking for dependencies to respond, possibly network or settings                            | pkg/util/errors/ErrTimeoutServicesNotReady          |
| 1003 | periodic health checker or status updater failed.  Services are not ready                                   | pkg/util/ErrPeriodicCheck                           |
| 1004 | error starting periodic or immediate status update                                                          | pkg/util/ErrStartingAgentStatusUpdate               |
| 1005 | error starting agent                                                                                        | pkg/util/ErrStartingVersionChecker                  |
| 1006 | error registering subscription webhook                                                                      | pkg/util/ErrRegisterSubscriptionWebhook             |
|      | 1100-1299 - for apic package errors                                                                         |                                                     |
| 1100 | general configuration error in CENTRAL                                                                      | pkg/apic/ErrCentralConfig                           |
| 1101 | error attempting to query for ENVIRONMENT, check CENTRAL_ENVIRONMENT                                        | pkg/apic/ErrEnvironmentQuery                        |
| 1102 | could not find specified team in Amplify Central, check CENTRAL_TEAM                                        | pkg/apic/ErrTeamNotFound                            |
| 1110 | connection to Amplify Central failed, possibly network                                                      | pkg/apic/ErrNetwork                                 |
| 1120 | request to Amplify Central failed, could be bad value for CENTRAL_ENVIRONMENT                               | pkg/apic/ErrRequestQuery                            |
| 1130 | request to get authentication token failed, possibly network or CENTAL_AUTH config                          | pkg/apic/ErrAuthenticationCall                      |
| 1131 | token retrieved but was invalid on request to Amplify Central, likely CENTRAL_AUTH config                   | pkg/apic/ErrAuthentication                          |
| 1140 | couldn't find a subscriber email address based on the ID in the subscription event                          | pkg/apic/ErrNoAddressFound                          |
| 1141 | couldn't contact Amplify Central for subscription, possible network error                                   | pkg/apic/ErrSubscriptionQuery                       |
| 1142 | couldn't get subscription data from Amplify Central, check network and CENTRAL_AUTH                         | pkg/apic/ErrSubscriptionResp                        |
| 1143 | couldn't create or update subscription schema data, possible Network error                                  | pkg/apic/ErrSubscriptionSchemaCreate                |
| 1144 | unexpected response when managing subscription schema on Amplify Central, check network and CENTRAL_AUTH    | pkg/apic/ErrSubscriptionSchemaResp                  |
| 1145 | unable to create webhook                                                                                    | pkg/apic/ErrCreateWebhook                           |
| 1146 | unable to create secret                                                                                     | pkg/apic/ErrCreateSecret                            |
| 1147 | error parsing filter in configuration. Syntax error                                                         | pkg/filter/ErrFilterConfiguration                   |
| 1148 | error parsing filter in configuration. Unrecognized expression                                              | pkg/filter/ErrFilterExpression                      |
| 1149 | error parsing filter in configuration                                                                       | pkg/filter/ErrFilterGeneralParse                    |
| 1150 | error parsing filter in configuration. Invalid call argument                                                | pkg/filter/ErrFilterArgument                        |
| 1151 | error parsing filter in configuration. Invalid selector type                                                | pkg/filter/ErrFilterSelectorType                    |
| 1152 | error parsing filter in configuration. Invalid selector expression                                          | pkg/filter/ErrFilterSelectorExpr                    |
| 1153 | error parsing filter in configuration. Invalid operator                                                     | pkg/filter/ErrFilterOperator                        |
| 1154 | error parsing filter in configuration. Unrecognized condition                                               | pkg/filter/ErrFilterCondition                       |
| 1155 | error getting subscription definition properties in Amplify Central                                         | pkg/apic/ErrGetSubscriptionDefProperties            |
| 1156 | error updating subscription definition properties in Amplify Central                                        | pkg/apic/ErrUpdateSubscriptionDefProperties         |
| 1157 | error getting catalog item API server info properties                                                       | pkg/apic/ErrGetCatalogItemServerInfoProperties      |
| 1158 | subscription manager is not in a running state                                                              | pkg/apic/ErrSubscriptionManagerDown                 |
| 1160 | error getting endpoints for the API specification                                                           | pkg/apic/ErrSetSpecEndPoints                        |
| 1161 | error deleting API Service for catalog item in Amplify Central                                              | pkg/agent/ErrDeletingService                        |
| 1162 | error deleting catalog item in Amplify Central                                                              | pkg/agent/ErrDeletingCatalogItem                    |
| 1163 | error retrieving API Service resource instances                                                             | pkg/agent/ErrUnableToGetAPIV1Resources              |
| 1163 | error retrieving API Service resource instances                                                             | pkg/agent/ErrUnableToGetAPIV1Resources              |
| 1164 | Amplify Central does not contain a team for the API. The Catalog Item will be assigned to the default       | pkg/apic/ErrTeamMismatch                            |
| 1165 | error creating category                                                                                     | pkg/apic/ErrCategoryCreate                          |
|      | 1300-1399 - for subscription notification errors                                                            |                                                     |
| 1300 | error communicating with server for subscription notifications (SMTP or webhook), check SUBSCRIPTION config | pkg/notify/ErrSubscriptionNotification              |
| 1301 | subscription notifications not configured, check SUBSCRIPTION config                                        | pkg/notify/ErrSubscriptionNoNotifications           |
| 1302 | error creating data for sending subscription notification                                                   | pkg/notify/ErrSubscriptionData                      |
| 1303 | email template not updated because an invalid authType was supplied                                         | pkg/notify/ErrSubscriptionBadAuthtype               |
| 1304 | no email template found for action                                                                          | pkg/notify/ErrSubscriptionNoTemplateForAction       |
| 1305 | error sending email to SMTP server                                                                          | pkg/notify/ErrSubscriptionSendEmail                 |
|      | 1400-1499 - for setting and parsing configuration errors                                                    |                                                     |
| 1401 | error parsing configuration values                                                                          | pkg/config/ErrBadConfig                             |
| 1402 | error in overriding configuration using file with environment variables                                     | pkg/config/ErrEnvConfigOverride                     |
| 1403 | invalid value for statusHealthCheckPeriod. Value must be between 1 and 5 minutes                            | pkg/config/ErrStatusHealthCheckPeriod               |
| 1404 | invalid value for statusHealthCheckInterval. Value must be between 30 seconds and 5 minutes                 | pkg/config/ErrStatusHealthCheckInterval             |
| 1405 | a key file could not be read                                                                                | pkg/config/ErrReadingKeyFile                        |
| 1410 | invalid configuration settings for the logging setup                                                        | pkg/config/ErrInvalidLogConfig                      |
| 1411 | invalid secret reference                                                                                    | pkg/cmd/properties/ErrInvalidSecretReference        |
|      | 1500-1599 - errors related to traceability output transport                                                 |                                                     |
| 1503 | http transport is not connected                                                                             | pkg/traceability/ErrHTTPNotConnected                |
| 1504 | failed to encode the json content                                                                           | pkg/traceability/ErrJSONEncodeFailed                |
| 1505 | invalid traceability config                                                                                 | pkg/traceability/ErrInvalidConfig                   |
| 1510 | global redaction have not been initialized                                                                  | pkg/traceability/redaction/ErrGlobalRedactionCfg    |
| 1511 | error while compiling regular expression                                                                    | pkg/traceability/redaction/ErrInvalidRegex          |
| 1520 | global sampling has not been initialized                                                                    | pkg/traceability/sampling/ErrGlobalSamplingCfg      |
| 1521 | invalid sampling configuration                                                                              | pkg/traceability/sampling/ErrSamplingCfg            |
| 1550 | error hit while applying redaction                                                                          | pkg/transaction/ErrInRedactions                     |
|      | 1600-1610 - errors in jobs library                                                                          |                                                     |
| 1600 | error registering job                                                                                       | pkg/jobs/ErrRegisteringJob                          |
| 1601 | error executing job                                                                                         | pkg/jobs/ErrExecutingJob                            |
| 1602 | error executing retry job                                                                                   | pkg/jobs/ErrExecutingRetryJob                       |
|      | 1611-1612 - errors in healthcheck library                                                                   |                                                     |
| 1611 | error starting periodic health check                                                                        | pkg/util/healthcheck/ErrStartingPeriodicHealthCheck |
| 1612 | maximum number of consecutive healthcheck errors hit                                                        | pkg/util/healthcheck/ErrMaxconsecutiveErrors        |
| 1613 | terminating agent, another instance of agent already running                                                | pkg/util/healthcheck/ErrAlreadyRunning              |
|      | 1900-1910 - errors managing agent service                                                                   |                                                     |
| 1900 | unsupported system for service installation                                                                 | pkg/cmd/service/daemon/ErrUnsupportedSystem         |
| 1901 | systemd is required for service installation                                                                | pkg/cmd/service/daemon/ErrNeedSystemd               |
| 1902 | service management requires root privileges                                                                 | pkg/cmd/service/daemon/ErrRootPrivileges            |
| 1903 | service has already been installed                                                                          | pkg/cmd/service/daemon/ErrAlreadyInstalled          |
| 1904 | service is running and cannot be removed until stopped                                                      | pkg/cmd/service/daemon/ErrCurrentlyRunning          |
| 1905 | service is not yet installed                                                                                | pkg/cmd/service/daemon/ErrNotInstalled              |
| 1906 | service is already running                                                                                  | pkg/cmd/service/daemon/ErrAlreadyRunning            |
| 1907 | service is already stopped                                                                                  | pkg/cmd/service/daemon/ErrAlreadyStopped            |
