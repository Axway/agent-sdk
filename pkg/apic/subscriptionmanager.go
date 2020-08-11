package apic

import (
	"sync"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/notification"
)

// SubscriptionManager - Interface for subscription manager
type SubscriptionManager interface {
	RegisterProcessor(state SubscriptionState, processor SubscriptionProcessor)
	RegisterValidator(validator SubscriptionValidator)
	Start()
	Stop()
	getPublisher() notification.Notifier
	getProcessorMap() map[SubscriptionState][]SubscriptionProcessor
	AddBlacklistItem(id string)
	RemoveBlacklistItem(id string)
}

// subscriptionManager -
type subscriptionManager struct {
	isRunning           bool
	publisher           notification.Notifier
	publishChannel      chan interface{}
	receiveChannel      chan interface{}
	publishQuitChannel  chan bool
	receiverQuitChannel chan bool
	processorMap        map[SubscriptionState][]SubscriptionProcessor
	validator           SubscriptionValidator
	statesToQuery       []string
	apicClient          *ServiceClient
	blacklist           map[string]string // subscription items to skip
	blacklistLock       *sync.RWMutex     // Use lock when making changes/reading the blacklist map
}

// newSubscriptionManager - Creates a new subscription manager
func newSubscriptionManager(apicClient *ServiceClient) SubscriptionManager {
	subscriptionMgr := &subscriptionManager{
		isRunning:     false,
		apicClient:    apicClient,
		processorMap:  make(map[SubscriptionState][]SubscriptionProcessor),
		statesToQuery: make([]string, 0),
		blacklist:     make(map[string]string),
		blacklistLock: &sync.RWMutex{},
	}

	return subscriptionMgr
}

// RegisterCallback - Register subscription processor callback for specified state
func (sm *subscriptionManager) RegisterProcessor(state SubscriptionState, processor SubscriptionProcessor) {
	processorList, ok := sm.processorMap[state]
	if !ok {
		processorList = make([]SubscriptionProcessor, 0)
	}
	sm.statesToQuery = append(sm.statesToQuery, string(state))
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
					if _, found := sm.blacklist[subscription.GetCatalogItemID()]; !found {
						sm.publishChannel <- subscription
					}
				}
			}
		case <-sm.publishQuitChannel:
			return
		}
		// Set sleep to throttle loop
		time.Sleep(1 * time.Second)
	}
}

func (sm *subscriptionManager) processSubscriptions() {
	for {
		select {
		case msg, ok := <-sm.receiveChannel:
			if ok {
				subscription, _ := msg.(CentralSubscription)
				sm.preprocessSubscription(&subscription)
				if subscription.ApicID != "" {
					sm.invokeProcessor(subscription)
				}
			}
		case <-sm.receiverQuitChannel:
			return
		}
	}
}

func (sm *subscriptionManager) preprocessSubscription(subscription *CentralSubscription) {
	subscription.ApicID = subscription.CatalogItemID
	subscription.apicClient = sm.apicClient
	apiserverInfo, err := sm.apicClient.getCatalogItemAPIServerInfoProperty(subscription.CatalogItemID)
	if err == nil && apiserverInfo.Environment.Name == sm.apicClient.cfg.GetEnvironmentName() {
		consumerInstance, err := sm.apicClient.getAPIServerConsumerInstance(apiserverInfo.ConsumerInstance.Name, nil)
		if sm.apicClient.cfg.IsPublishToEnvironmentAndCatalogMode() {
			if err == nil && consumerInstance.Metadata != nil {
				subscription.ApicID = consumerInstance.Metadata.ID
			}
		} else {
			if err == nil && consumerInstance.Metadata != nil && len(consumerInstance.Metadata.References) > 0 {
				for _, reference := range consumerInstance.Metadata.References {
					if reference.Kind == "APIServiceInstance" {
						subscription.ApicID = reference.ID
					}
				}
			}
		}
	}
}

func (sm *subscriptionManager) invokeProcessor(subscription CentralSubscription) {
	invokeProcessor := true
	if sm.validator != nil {
		invokeProcessor = sm.validator(&subscription)
	}
	if invokeProcessor {
		processorList, ok := sm.processorMap[SubscriptionState(subscription.State)]
		if ok {
			for _, processor := range processorList {
				processor(&subscription)
			}
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

func (sm *subscriptionManager) getProcessorMap() map[SubscriptionState][]SubscriptionProcessor {
	return sm.processorMap
}

func (sm *subscriptionManager) AddBlacklistItem(id string) {
	sm.blacklistLock.RLock()
	defer sm.blacklistLock.RUnlock()
	sm.blacklist[id] = "" // don't care about the value
}

func (sm *subscriptionManager) RemoveBlacklistItem(id string) {
	sm.blacklistLock.RLock()
	defer sm.blacklistLock.RUnlock()
	delete(sm.blacklist, id)
}
