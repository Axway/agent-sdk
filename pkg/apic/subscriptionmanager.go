package apic

import (
	"time"

	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/notification"
)

// SubscriptionManager - Interface for subscription manager
type SubscriptionManager interface {
	RegisterProcessor(state string, processor SubscriptionProcessor)
	RegisterValidator(validator SubscriptionValidator)
	Start()
	Stop()
	getPublisher() notification.Notifier
	getProcessorMap() map[string][]SubscriptionProcessor
}

// subscriptionManager -
type subscriptionManager struct {
	isRunning           bool
	publisher           notification.Notifier
	publishChannel      chan interface{}
	receiveChannel      chan interface{}
	publishQuitChannel  chan bool
	receiverQuitChannel chan bool
	processorMap        map[string][]SubscriptionProcessor
	validator           SubscriptionValidator
	statesToQuery       []string
	apicClient          *ServiceClient
}

// newSubscriptionManager - Creates a new subscription manager
func newSubscriptionManager(apicClient *ServiceClient) SubscriptionManager {
	subscriptionMgr := &subscriptionManager{
		isRunning:     false,
		apicClient:    apicClient,
		processorMap:  make(map[string][]SubscriptionProcessor),
		statesToQuery: make([]string, 0),
	}

	return subscriptionMgr
}

// RegisterCallback - Register subscription processor callback for specified state
func (sm *subscriptionManager) RegisterProcessor(state string, processor SubscriptionProcessor) {
	processorList, ok := sm.processorMap[state]
	if !ok {
		processorList = make([]SubscriptionProcessor, 0)
	}
	sm.statesToQuery = append(sm.statesToQuery, state)
	sm.processorMap[state] = append(processorList, processor)
}

// RegisterValidator - Registers validator for subscription to be processed
func (sm *subscriptionManager) RegisterValidator(validator SubscriptionValidator) {
	sm.validator = validator
}

func (sm *subscriptionManager) pollSubscriptions() {
	ticker := time.NewTicker(sm.apicClient.cfg.GetPollInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			subscriptions, err := sm.apicClient.getSubscriptions(sm.statesToQuery)
			if err == nil {
				for _, subscription := range subscriptions {
					if sm.apicClient.cfg.GetAgentMode() == corecfg.Connected {
						// Get API Service info
						// Get Consumer Instance
						// Assign subscription ApicID with ApiServiceInstanceId
						apiserverInfo, err := sm.apicClient.getCatalogItemAPIServerInfoProperty(subscription.CatalogItemID)
						if err == nil && apiserverInfo.Environment.Name == sm.apicClient.cfg.GetEnvironmentName() {
							consumerInstance, err := sm.apicClient.getAPIServerConsumerInstance(apiserverInfo.ConsumerInstance.Name)
							if err == nil && consumerInstance.Metadata != nil && len(consumerInstance.Metadata.References) > 0 {
								for _, references := range consumerInstance.Metadata.References {
									if references.Kind == "APIServiceInstance" {
										subscription.ApicID = references.ID
									}
								}
							}
						}
					} else {
						// Use catalog item id as ApicID
						subscription.ApicID = subscription.CatalogItemID
					}
					sm.publishChannel <- subscription

				}
			}
		case <-sm.publishQuitChannel:
			return
		}
	}
}

func (sm *subscriptionManager) processSubscriptions() {
	for {
		select {
		case msg, ok := <-sm.receiveChannel:
			if ok {
				subscription, _ := msg.(Subscription)
				invokeProcessor := true
				if sm.validator != nil {
					invokeProcessor = sm.validator(subscription)
				}

				if invokeProcessor {
					processorList, ok := sm.processorMap[subscription.State]
					if ok {
						for _, processor := range processorList {
							processor(subscription)
						}
					}
				}
			}
		case <-sm.receiverQuitChannel:
			return
		}
	}
}

// Start - Start processing subscriptions
func (sm *subscriptionManager) Start() {
	if !sm.isRunning {
		sm.publishQuitChannel = make(chan bool)
		sm.receiverQuitChannel = make(chan bool)

		sm.publishChannel = make(chan interface{})
		sm.receiveChannel = make(chan interface{})

		sm.publisher, _ = notification.RegisterNotifier("CentralSubscriptions", sm.publishChannel)
		notification.Subscribe("CentralSubscriptions", sm.receiveChannel)

		go sm.publisher.Start()
		go sm.pollSubscriptions()
		go sm.processSubscriptions()
		sm.isRunning = true
	}
}

// Stop - Stop processing subscriptions
func (sm *subscriptionManager) Stop() {
	if sm.isRunning {
		sm.publisher.Stop()
		sm.publishQuitChannel <- true
		sm.receiverQuitChannel <- true
		sm.isRunning = false
	}
}

func (sm *subscriptionManager) getPublisher() notification.Notifier {
	return sm.publisher
}

func (sm *subscriptionManager) getProcessorMap() map[string][]SubscriptionProcessor {
	return sm.processorMap
}
