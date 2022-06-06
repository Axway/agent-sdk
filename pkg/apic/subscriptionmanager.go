package apic

// TODO - this file should be able to be removed once Unified Catalog support has been removed
import (
	"sync"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	getProcessorMap() map[SubscriptionState][]SubscriptionProcessor
	OnConfigChange(apicClient *ServiceClient)
}

// subscriptionManager -
type subscriptionManager struct {
	jobs.Job
	isRunning            bool
	ucSubPublisher       notification.Notifier // unified catalog subscrtion notifier
	ucSubPublishChan     chan interface{}      // unified catalog subscrtion publish channel
	ucSubReceiveChannel  chan interface{}      // unified catalog subscrtion receive channel
	accReqPublisher      notification.Notifier // access request notifier
	accReqPublishChan    chan interface{}      // access request publish channel
	accReqReceiveChannel chan interface{}      // access request receive channel
	receiverQuitChannel  chan bool
	processorMap         map[SubscriptionState][]SubscriptionProcessor
	validator            SubscriptionValidator
	ucStatesToQuery      []string // states to query for unified catalog subscriptions
	arStatesToQuery      []string // states to query for access requests
	apicClient           *ServiceClient
	locklist             map[string]string // subscription items to skip because they are locked
	locklistLock         *sync.RWMutex     // Use lock when making changes/reading the locklist map
	jobID                string
	pollingEnabled       bool
	pollInterval         time.Duration
	useAccessRequests    bool
}

// newSubscriptionManager - Creates a new subscription manager
func newSubscriptionManager(apicClient *ServiceClient) SubscriptionManager {
	subscriptionMgr := &subscriptionManager{
		isRunning:       false,
		apicClient:      apicClient,
		processorMap:    make(map[SubscriptionState][]SubscriptionProcessor),
		ucStatesToQuery: make([]string, 0),
		arStatesToQuery: make([]string, 0),
		locklist:        make(map[string]string),
		locklistLock:    &sync.RWMutex{},
		pollingEnabled:  apicClient.cfg.GetSubscriptionConfig().PollingEnabled(),
		pollInterval:    apicClient.cfg.GetPollInterval(),
	}

	return subscriptionMgr
}

// OnConfigChange - config change handler
func (sm *subscriptionManager) OnConfigChange(apicClient *ServiceClient) {
	sm.apicClient = apicClient
}

// RegisterCallback - Register subscription processor callback for specified state
func (sm *subscriptionManager) RegisterProcessor(state SubscriptionState, processor SubscriptionProcessor) {
	if state.isUnifiedCatalogState() {
		processorList, ok := sm.processorMap[state]
		if !ok {
			processorList = make([]SubscriptionProcessor, 0)
		}
		sm.ucStatesToQuery = append(sm.ucStatesToQuery, string(state))
		sm.processorMap[state] = append(processorList, processor)
	}

	processorList, ok := sm.processorMap[state.getAccessRequestState()]
	if !ok {
		processorList = make([]SubscriptionProcessor, 0)
	}
	sm.arStatesToQuery = append(sm.arStatesToQuery, string(state.getAccessRequestState()))
	sm.processorMap[state.getAccessRequestState()] = append(processorList, processor)
}

// RegisterValidator - Registers validator for subscription to be processed
func (sm *subscriptionManager) RegisterValidator(validator SubscriptionValidator) {
	sm.validator = validator
}

func (sm *subscriptionManager) Ready() bool {
	sm.locklistLock.Lock()
	defer sm.locklistLock.Unlock()
	return sm.isRunning
}

func (sm *subscriptionManager) Status() error {
	if !sm.isRunning {
		return ErrSubscriptionManagerDown
	}
	return nil
}

func (sm *subscriptionManager) Execute() error {
	if sm.apicClient.cfg.IsMarketplaceSubsEnabled() {
		log.Trace("Unified catalog polling disabled when using Marketplace Provisioning")
	}
	// query for central subscriptions
	subscriptions, err := sm.apicClient.getSubscriptions(sm.ucStatesToQuery)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		sm.ucSubPublishChan <- subscription
	}
	return err
}

func (sm *subscriptionManager) processSubscriptions() {
	for {
		var subscription Subscription
		select {
		case msg, ok := <-sm.ucSubReceiveChannel:
			if ok {
				centralSub := msg.(CentralSubscription)
				subscription = &centralSub
			}
		case <-sm.receiverQuitChannel:
			return
		}
		if subscription != nil {
			id := subscription.GetID()
			if !sm.isItemOnLocklist(id) {
				sm.addLocklistItem(id)
				log.Tracef("checking if we should handle subscription %s", subscription.GetName())
				centralSub := subscription.(*CentralSubscription)
				process := sm.preprocessSubscription(centralSub)
				if process {
					log.Infof("Subscription %s received", subscription.GetName())
					sm.invokeProcessor(subscription)
					log.Infof("Subscription %s processed", subscription.GetName())
				}
				sm.removeLocklistItem(id)
			}
		}
	}
}

func (sm *subscriptionManager) preprocessSubscription(subscription *CentralSubscription) bool {
	subscription.ApicID = subscription.GetCatalogItemID()
	subscription.apicClient = sm.apicClient
	apiserverInfo, err := sm.apicClient.getCatalogItemAPIServerInfoProperty(subscription.ApicID, subscription.GetID())
	if err != nil {
		log.Error(utilerrors.Wrap(ErrGetCatalogItemServerInfoProperties, err.Error()))
		return false
	}
	if apiserverInfo.Environment.Name != sm.apicClient.cfg.GetEnvironmentName() {
		log.Debugf("Subscription '%s' skipped because associated catalog item belongs to '%s' environment and the agent is configured for managing '%s' environment", subscription.GetName(), apiserverInfo.Environment.Name, sm.apicClient.cfg.GetEnvironmentName())
		return false
	}
	if apiserverInfo.ConsumerInstance.Name == "" {
		log.Debugf("Subscription '%s' skipped because associated catalog item is not created by agent", subscription.GetName())
		return false
	}
	sm.preprocessSubscriptionForConsumerInstance(subscription, apiserverInfo.ConsumerInstance.Name)

	if subscription.GetApicID() != "" && subscription.GetRemoteAPIID() != "" {
		return true
	}
	log.Debugf("Subscription '%s' skipped because it did not have an API Central ID and/or a Remote API ID", subscription.GetName())
	return false
}

func (sm *subscriptionManager) preprocessSubscriptionForConsumerInstance(subscription *CentralSubscription, consumerInstanceName string) {
	consumerInstance, err := sm.apicClient.getAPIServerConsumerInstance(consumerInstanceName, nil)
	if err == nil {
		if !sm.apicClient.cfg.IsMarketplaceSubsEnabled() {
			resource, _ := consumerInstance.AsInstance()
			sm.setSubscriptionInfo(subscription, resource)
		} else {
			log.Trace("Preprocess subscription for environment mode only")
			sm.preprocessSubscriptionForAPIServiceInstance(subscription, consumerInstance)
		}
	}
}

func (sm *subscriptionManager) preprocessSubscriptionForAPIServiceInstance(subscription *CentralSubscription, consumerInstance *management.ConsumerInstance) {
	if consumerInstance != nil && len(consumerInstance.Metadata.References) > 0 {
		for _, reference := range consumerInstance.Metadata.References {
			if reference.Kind == "APIServiceInstance" {
				apiServiceInstance, err := sm.apicClient.GetAPIServiceInstanceByName(reference.ID)
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
func (sm *subscriptionManager) setSubscriptionInfo(subscription Subscription, apiServerResource *v1.ResourceInstance) {
	if apiServerResource != nil {
		subscription.setAPIResourceInfo((apiServerResource))
		if subscription.GetRemoteAPIStage() != "" {
			log.Debugf("Subscription Details (ID: %s, Reference type: %s, Reference ID: %s, Remote API ID: %s)",
				subscription.GetID(), apiServerResource.Kind, subscription.GetApicID(), subscription.GetRemoteAPIID())
		} else {
			log.Debugf("Subscription Details (ID: %s, Reference type: %s, Reference ID: %s, Remote API ID: %s, Remote API Stage: %s)",
				subscription.GetID(), apiServerResource.Kind, subscription.GetApicID(), subscription.GetRemoteAPIID(), subscription.GetRemoteAPIStage())
		}
	}
}

func (sm *subscriptionManager) invokeProcessor(subscription Subscription) {
	invokeProcessor := true
	if sm.validator != nil {
		invokeProcessor = sm.validator(subscription)
	}
	if invokeProcessor {
		processorList, ok := sm.processorMap[SubscriptionState(subscription.GetState())]
		if ok {
			for _, processor := range processorList {
				processor(subscription)
			}
		}
	}
}

// Start - Start processing subscriptions
func (sm *subscriptionManager) Start() {
	// clean out the map each time start is called
	sm.locklist = make(map[string]string)

	if !sm.isRunning {
		sm.receiverQuitChannel = make(chan bool)

		sm.ucSubPublishChan = make(chan interface{})
		sm.ucSubReceiveChannel = make(chan interface{}) // unified catlog subscriptions channel

		sm.ucSubPublisher, _ = notification.RegisterNotifier("CentralSubscriptions", sm.ucSubPublishChan)
		notification.Subscribe("CentralSubscriptions", sm.ucSubReceiveChannel)

		go sm.ucSubPublisher.Start()

		if sm.useAccessRequests {
			sm.accReqPublishChan = make(chan interface{})
			sm.accReqReceiveChannel = make(chan interface{}) // access request channel

			sm.accReqPublisher, _ = notification.RegisterNotifier("AccessRequests", sm.accReqPublishChan)
			notification.Subscribe("AccessRequests", sm.accReqReceiveChannel)

			go sm.accReqPublisher.Start()
		}

		go sm.processSubscriptions()

		// Wait for at least one processor to register before registering the job
		if (len(sm.ucStatesToQuery) > 0 || len(sm.arStatesToQuery) > 0) && sm.pollingEnabled && sm.jobID == "" {
			var err error
			sm.jobID, err = jobs.RegisterIntervalJobWithName(sm, sm.pollInterval, "Subscription Manager")
			if err != nil {
				log.Errorf("Error registering interval job to poll for subscriptions: %s", err.Error())
			}
		}
		sm.locklistLock.Lock()
		sm.isRunning = true
		sm.locklistLock.Unlock()
	}
}

// Stop - Stop processing subscriptions
func (sm *subscriptionManager) Stop() {
	if sm.isRunning {
		sm.ucSubPublisher.Stop()
		if sm.useAccessRequests {
			sm.accReqPublisher.Stop()
		}
		sm.receiverQuitChannel <- true
		sm.isRunning = false
		jobs.UnregisterJob(sm.jobID)
		sm.jobID = ""
	}
}

func (sm *subscriptionManager) getProcessorMap() map[SubscriptionState][]SubscriptionProcessor {
	return sm.processorMap
}

func (sm *subscriptionManager) addLocklistItem(id string) {
	sm.locklistLock.RLock()
	defer sm.locklistLock.RUnlock()
	sm.locklist[id] = "" // don't care about the value
}

func (sm *subscriptionManager) removeLocklistItem(id string) {
	sm.locklistLock.RLock()
	defer sm.locklistLock.RUnlock()
	delete(sm.locklist, id)
}

func (sm *subscriptionManager) isItemOnLocklist(id string) bool {
	_, found := sm.locklist[id]
	return found
}
