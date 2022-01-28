# Watch Manager
The watch library provides the ability to subscribe with gRPC based service in Amplify Central and receive event on API server resources based on configured filter defined using WatchTopic resource.

## Table of Contents
- [Overview](#Overview)
- [Watch manager](#Watch-manager)
    - [Watch client configuration](#Watch-client-configuration)
    - [Watch client options](#Watch-client-options)
    - [Registration](#Registration)
- [Client Example](#Client-Example)

## Overview
Amplify Central provides a gRPC based watch service that allows the ability to register a subscription and receive event for API server resources on a bi-directional gRPC stream. The subscription is based on an API server resource called WatchTopic which defines the set of filters. The Amplify Central watch service uses the watch filter to match the API server resource event before pushing them to subscribed clients. 

#### WatchTopic Example
```yaml
group: management
apiVersion: v1alpha1
kind: WatchTopic
name: sample-watch-topic
title: sample-watch-topic
spec:
  filters:
    - kind: APIService
      name: '*'
      type:
        - created
        - updated
        - deleted
      group: management
      scope:
        kind: Environment
        name: sample-env
    - kind: APIServiceInstance
      name: '*'
      type:
        - created
        - updated
        - deleted
      group: management
      scope:
        kind: Environment
        name: sample-env
  description: >-
    Sample watch topic in sample-env environment.
```

#### Creating WatchTopic resource
Use Axway Central CLI to create the watch topic resource. Create a file with YAML or JSON definition for the WatchTopic resource specifying the filters for the resources to subscribe for watch event(see above example).

Use the following command to authenticate with you Amplify platform credentials
```bash
axway auth login
```

Use the command below to create the watch topic resource.
```shell
axway central apply -f <filePath-for-watch-topic-resource>
```

Use the following command to verify the watch topic resource and note the value for "metadata.selfLink" property from the output of the above command. The watch topic self link will be used while registering the watch subscription
```shell
axway central get watchtopic <logical-name-of-watch-topic-resource> -o yaml
```

Once the WatchTopic resources is defined in API server, the Amplify Central watch service starts to monitor API resource events, deliver the event in real time over the subscribed gRPC connections and persists the event to allow the clients to retrieve them at a point of time based on the sequence identifier. This helps the client to catch up with any API server resource events that were missed while the client was not running.

## Watch manager
The watch manager library provides interface to create and manage the client communication with Amplify Central watch service. The library creates the gRPC client connection based on the provided configuration and watch options that can be setup while creating the client. The watch client interface provided by the library allows to register the watch subscription using the provided watch topic self link. 

For registration with the watch service the watch client uses the provided token getter to retrieve the JWT token that client uses to call the subscribe gRPC watch service by including the token as metadata. The watch service uses the metadata to authorize the subscription request and opens a long-lived bi-directional stream connection with the client on successful authorization. The bi-directional stream is then used by client to refresh the token when the token is about to expire to keep the connection active.

The client manages the long-lived gRPC stream connection by sending keep alive pings at a configured interval. The client transport waits for configured timeout interval to receive the ping acknowledgement after which the transport is disconnected.

### Watch client configuration
The watch manager library requires the following configuration to establish the connection with Amplify Central watch service.

```golang
type Config struct {
	Host        string
	Port        uint32
	TenantID    string
	TokenGetter TokenGetter
}
```

- Host: identifies the host for Amplify Central watch service(US region: apicentral.axway.com, EU region: central.eu-fr.axway.com)
- Port: identifies the port for Amplify Central watch service (443)
- TenantID: Amplify platform organization ID
- TokenGetter: interface to retrieve AxwayID token

### Watch client options
The watch manager library provides following set of options that the implementation can choose to use for setting up/overriding the properties for the gRPC stream connection.

- WithLogger - The option method takes *logrus.Entry as argument and allows to override the client stream logger.
- WithTLSConfig - Receives *tls.Config as argument to override the default TLS configuration. 
- WithKeepAlive - The method take keep alive ping interval and timeout to override the default values.
- WithProxy - The method take the proxy url to be used for establishing the gRPC connection via specified proxy
- WithSyncEvents - The method take instance for following interface. If setup, the GetSequence method is invoked on successful gRPC connection to fetch events after the sequence id returned by the method.
    ```golang
    type SequenceProvider interface {
        GetSequence() int64
    }
    ```

### Registration
To create a new watch manager, use the following method from watchmanager package with the watch configuration and set of options
```golang
func New(cfg *Config, opts ...Option) (Manager, error)
```

The method create a new watch manager client and returns the following interface to allow implementation to manage the gRPC watch
```golang
type Manager interface {
	RegisterWatch(topicSelfLink string, eventChan chan *proto.Event, errChan chan error) (string, error)
	CloseWatch(id string) error
	CloseConn()
	Status() bool
}
```

The client can call the *RegisterWatch* method with the topic self link and a set of channels to receive event and error.

When the client initiates the subscription request, it call the sequence getter if configured to get the last known sequence identifier of the resource event that the implementation received. On successful subscription request, the client places an API call to Amplify Central watch service to fetch the events that were missed while the gRPC watch stream connection was not active.

The client on receiving event from the gRPC stream (or fetched by API call) hands over the events to implementation by writing them to event channel configured while registering the watch subscription. In case the client receives any error on gRPC stream connection, the error is written to an error channel configured while registering the watch subscription.

Below is the structure of the event that is received by the Amplify Central watch service(refer [./proto/watch.pb.go](#./proto/watch.pb.go) for more detail)
```golang

type Event struct {
    // Event ID
    Id            string
    // Event Time
    Time          string
    // Event Version
    Version       string
    // Product raising the event
    Product       string
    // Event correlation ID 
    CorrelationId string
    // Organization associated to the event
    Organization  *Organization
    // Event Type
    Type          Event_Type
    // Event payload representing the API server resource instance
    Payload       *ResourceInstance
    // Event metadata holding watch topic id, self link, event sequence ID and sub resource name (if event raised for sub resource) 
    Metadata      *EventMeta
}

```

## Client Example
```golang

type sequenceManager struct {
    ...
}

func (s *sequenceManager) GetSequence() int64 {
    ...
    // get last known sequence ID to fetch event while the client was down
    ...  
    return sequenceID
}


type AxwayIDTokenManager struct {
    ...
}

func (a *AxwayIDTokenManager) GetToken() (string, error)
    // fetch token
    return token, err
}

func defaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		},
	}
}

func startWatch(tenantID string, host string, port uint32, topicSelfLink string, proxyUrl string) error {
    /**
    * 1. Create token getter that will return the AxwayID JWT token for authorizing the client connection
    */
    tokenManager := &AxwayIDTokenManager{}
    // Alternatively use SDK token getter component by calling NewTokenAuth from 
    // github.com/Axway/agent-sdk/pkg/apic/auth package

    /**
    * 2. Setup watch config
    */
    cfg := &watchmanager.Config{
        Host:        host,
        Port:        port,
        TenantID:    tenantID,
        TokenGetter: tokenManager,
    }

    /**
    * 3. Create watch client using supported options
    */
    wm, err := watchmanager.New(cfg,
        watchmanager.WithLogger(entry),
        watchmanager.WithTLSConfig(defaultTLSConfig()),
        watchmanager.WithKeepAlive(30*time.Second, 10*time.Second),
        watchmanager.WithProxy(proxyUrl),
        watchmanager.WithSyncEvents(getSequenceManager()),
    )
    if err != nil {
        return err
    }

    /**
    * 4. Create channels to receive event and error
    */
    eventChannel, errCh := make(chan *proto.Event), make(chan error)

    /**
    * 5. Register the watch subscription
    */
    subscriptionID, err := wm.RegisterWatch(topicSelfLink, eventChannel, errCh)
    if err != nil {
        log.Error(err)
        return
    }

    log.Infof("watch subscription (%s) registered successfully", subscriptionID)

    /**
    * 6. Start to process event and error received on channel
    */
    for {
        select {
        case err = <-errCh:
            log.Error(err)
            wm.CloseWatch(subscriptionID)
            return
        case event := <-eventChannel:
            bts, _ := json.MarshalIndent(event, "", "  ")
            log.Info(string(bts))
        }
    }
}

```