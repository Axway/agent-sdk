# Amplify Agents SDK

The Amplify Agents SDK provides APIs and utilities that developers can use to build Golang based applications to discover APIs hosted on remote API Gateway (for e.g. AWS, Azure, Axway API Manager etc.) and publish their representation in Amplify Central as API server resources and Catalog items. The Amplify Agents SDK can also be used to build applications that can monitor traffic events (for discovered or undiscovered APIs) and publish them to Amplify Central API Observer.

The Amplify Agents SDK helps in reducing complexity in implementing against the direct Amplify Central REST API interface and hides low level plumbing to provide discovery and traceability related features.

## Support Policy

The Amplify Agents SDK is supported under [Axway support policy](https://docs.axway.com/bundle/amplify-central/page/docs/amplify_relnotes/agent_agentsdk_support_policy/index.html).

## Installation

Make sure you have [Go installed](https://golang.org/doc/install) and then use the following command to install the Amplify Agents SDK

go get github.com/Axway/agent-sdk/

## Packages

| Name         | Description                                                                                                                                          |
|--------------|------------------------------------------------------------------------------------------------------------------------------------------------------|
| agent        | This package holds the interface for agent initialization and managing discovered APIs                                                               |
| api          | This package provides client interface for making REST API calls                                                                                     |
| apic         | This package contains Amplify Central service client                                                                                                 |
| cache        | This package can be used to create an in-memory cache of items                                                                                       |
| cmd          | This package provides the implementation of the root command line processor                                                                          |
| config       | This package provides the base configuration required by Amplify Agents SDK to communicate with Amplify Central                                      |
| filter       | This package provides the filter implementation to allow discovering APIs based on certain conditions                                                |
| jobs         | This package provides a tooling to coordinate agent tasks [SDK Jobs](./pkg/jobs/README.md)                                                           |
| notification | This package contains structs that can be used for creating notifications and subscribers to those notifications                                     |
| notify       | This package contains the subscription notification setup for the agents to send SMTP and/or webhook notification for subscription process outcomes  |
| transaction  | This package holds definitions of event and interfaces to process them for traceability                                                              |
| traceability | This package provides the transport lumberjack/HTTP clients that can be used for building traceability agent                                         |
| util         | This package has SDK utility packages for use by all agents                                                                                          |

[Getting started to build discovery agent](./docs/discovery/index.md)

[Getting started to build traceability agent](./docs/traceability/index.md)

[Utilities](./docs/utilities/index.md)

## Sample projects

The developers can use the stubs packaged as zip file to build agents using the Amplify Agents SDK. The zip files contains code for sample discovery and traceability agent respectively, build scripts and instructions in README.md to make modifications to implement their own agents.

[Download the stub project with sample discovery agent](https://github.com/Axway/agent-sdk/raw/main/samples/apic_discovery_agent.zip)

[Download the stub project with sample traceability agent](https://github.com/Axway/agent-sdk/raw/main/samples/apic_traceability_agent.zip)
