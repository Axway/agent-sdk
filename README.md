# AMPLIFY Central Agents Core

## Packages

| Name         | Description                                                                                                                                         |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| api          | This package contains an API Client for making REST calls                                                                                           |
| apic         | This package contains all communication used for Agent -> AMPLIFY calls                                                                             |
| cache        | This package can be used to create a memory cache of items                                                                                          |
| cmd          | This package has all of the root command for launching agents from the command line                                                                 |
| config       | This package has the base config structs for creating agents                                                                                        |
| filter       | This package contains the filtering mechanism for determining what endpoints are sent to AMPLIFY Central                                            |
| notification | This package contains structs that can be used for creating notifications and subscribers to those notifications                                    |
| notify       | This package contains the subscription notification setup for the agents to send SMTP and/or webhook notification for subscription process outcomes |
| transaction  | This package has the common event information for the traceability agents                                                                           |
| util         | This package has SDK utility packages for use by all agents                                                                                         |
