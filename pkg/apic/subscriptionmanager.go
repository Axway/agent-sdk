package apic

import (
	"sync"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/notification"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
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
	OnConfigChange(apicClient *ServiceClient)
}

// subscriptionManager -
type subscriptionManager struct {
	jobs.Job
	isRunning           bool
	publisher           notification.Notifier
	publishChannel      chan interface{}
	receiveChannel      chan interface{}
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

	if apicClient.cfg.GetSubscriptionConfig().PollingEnabled() {
		_, err := jobs.RegisterIntervalJob(subscriptionMgr, apicClient.cfg.GetPollInterval())
		if err != nil {
			log.Errorf("Error registering interval job to poll for subscriptions: %s", err.Error())
		}
	}

	return subscriptionMgr
}

// OnConfigChange - config change handler
func (sm *subscriptionManager) OnConfigChange(apicClient *ServiceClient) {
	sm.apicClient = apicClient
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

func (sm *subscriptionManager) Ready() bool {
	return sm.isRunning
}

func (sm *subscriptionManager) Status() error {
	if !sm.isRunning {
		return ErrSubscriptionManagerDown
	}
	return nil
}

func (sm *subscriptionManager) Execute() error {
	subscriptions, err := sm.apicClient.getSubscriptions(sm.statesToQuery)
	if err == nil {
		for _, subscription := range subscriptions {
			if _, found := sm.blacklist[subscription.GetCatalogItemID()]; !found {
				sm.publishChannel <- subscription
			}
		}
	}
	return nil
}

func (sm *subscriptionManager) processSubscriptions() {
	for {
		select {
		case msg, ok := <-sm.receiveChannel:
			if ok {
				subscription, _ := msg.(CentralSubscription)
				sm.preprocessSubscription(&subscription)
				if subscription.ApicID != "" && subscription.GetRemoteAPIID() != "" {
					sm.invokeProcessor(subscription)
				}
			}
		case <-sm.receiverQuitChannel:
			return
		}
	}
}

func (sm *subscriptionManager) preprocessSubscription(subscription *CentralSubscription) {
	subscription.ApicID = subscription.GetCatalogItemID()
	subscription.apicClient = sm.apicClient

	apiserverInfo, err := sm.apicClient.getCatalogItemAPIServerInfoProperty(subscription.GetCatalogItemID(), subscription.GetID())
	if err != nil {
		log.Error(utilerrors.Wrap(ErrGetCatalogItemServerInfoProperties, err.Error()))
		return
	}
	if apiserverInfo.Environment.Name != sm.apicClient.cfg.GetEnvironmentName() {
		log.Debugf("Subscription '%s' skipped because associated catalog item belongs to '%s' environment and the agent is configured for managing '%s' environment", subscription.GetName(), apiserverInfo.Environment.Name, sm.apicClient.cfg.GetEnvironmentName())
		return
	}
	if apiserverInfo.ConsumerInstance.Name == "" {
		log.Debugf("Subscription '%s' skipped because associated catalog item is not created by agent", subscription.GetName())
		return
	}
	sm.preprocessSubscriptionForConsumerInstance(subscription, apiserverInfo.ConsumerInstance.Name)
}

func (sm *subscriptionManager) preprocessSubscriptionForConsumerInstance(subscription *CentralSubscription, consumerInstanceName string) {
	consumerInstance, err := sm.apicClient.getAPIServerConsumerInstance(consumerInstanceName, nil)
	if err == nil {
		if sm.apicClient.cfg.IsPublishToEnvironmentAndCatalogMode() {
			resource, _ := consumerInstance.AsInstance()
			sm.setSubscriptionInfo(subscription, resource)
		} else {
			log.Debug("Preprocess subscription for environment mode only")
			sm.preprocessSubscriptionForAPIServiceInstance(subscription, consumerInstance)
		}
	}
}

func (sm *subscriptionManager) preprocessSubscriptionForAPIServiceInstance(subscription *CentralSubscription, consumerInstance *v1alpha1.ConsumerInstance) {
	if consumerInstance != nil && len(consumerInstance.Metadata.References) > 0 {
		for _, reference := range consumerInstance.Metadata.References {
			if reference.Kind == "APIServiceInstance" {
				apiServiceInstance, err := sm.apicClient.getAPIServiceInstanceByName(reference.ID)
				if err == nil {
					resource, _ := apiServiceInstance.AsInstance()
					sm.setSubscriptionInfo(subscription, resource)
				} else {
					log.Errorf(err.Error())
				}
			}
		}
	}
}

// setSubscriptionInfo - Sets subscription identifier that will be used as references
// - ApicID - using the metadata of API server resource metadata.id
// - RemoteAPIID - using the attribute externalAPIID on API server resource
// - RemoteAPIStage - using the attribute externalAPIStage on API server resource (if present)
func (sm *subscriptionManager) setSubscriptionInfo(subscription *CentralSubscription, apiServerResource *v1.ResourceInstance) {
	if apiServerResource != nil {
		subscription.ApicID = apiServerResource.Metadata.ID
		subscription.RemoteAPIID = apiServerResource.Attributes[AttrExternalAPIID]
		subscription.RemoteAPIStage = apiServerResource.Attributes[AttrExternalAPIStage]
		if subscription.RemoteAPIStage != "" {
			log.Debugf("Subscription Details (ID: %s, Reference type: %s, Reference ID: %s, Remote API ID: %s)",
				subscription.GetID(), apiServerResource.Kind, subscription.ApicID, subscription.RemoteAPIID)
		} else {
			log.Debugf("Subscription Details (ID: %s, Reference type: %s, Reference ID: %s, Remote API ID: %s, Remote API Stage: %s)",
				subscription.GetID(), apiServerResource.Kind, subscription.ApicID, subscription.RemoteAPIID, subscription.RemoteAPIStage)
		}
	}
}

func (sm *subscriptionManager) invokeProcessor(subscription CentralSubscription) {
	invokeProcessor := true
	if sm.validator != nil {
		invokeProcessor = sm.validator(&subscription)
	}
	if invokeProcessor {
		processorList, ok := sm.processorMap[SubscriptionState(subscription.GetState())]
		if ok {
			for _, processor := range processorList {
				processor(&subscription)
			}
		}
	}
}

// Start - Start processing subscriptions
func (sm *subscriptionManager) Start() {
	// Add an polling interval delay prior to starting, but do not make calling function wait
	go func() {
		time.Sleep(sm.apicClient.cfg.GetPollInterval())
		if !sm.isRunning {
			sm.receiverQuitChannel = make(chan bool)

			sm.publishChannel = make(chan interface{})
			sm.receiveChannel = make(chan interface{})

			sm.publisher, _ = notification.RegisterNotifier("CentralSubscriptions", sm.publishChannel)
			notification.Subscribe("CentralSubscriptions", sm.receiveChannel)

			go sm.publisher.Start()
			go sm.processSubscriptions()
			sm.isRunning = true
		}
	}()
}

// Stop - Stop processing subscriptions
func (sm *subscriptionManager) Stop() {
	if sm.isRunning {
		sm.publisher.Stop()
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
