package notification

import (
	"fmt"
	"sync"

	guuid "github.com/google/uuid"

	log "github.com/Axway/agent-sdk/pkg/util/log"
)

var notifiers map[string]Notifier
var pubLock = &sync.RWMutex{} // Lock used when reading/modifying notifiers map

// Notifier - any channel that has the potential to have listeners (1-n)
type Notifier interface {
	GetName() string
	Stop()
	Subscribe(Subscriber)
	Unsubscribe(string) error
	Start()
}

type channelNotifier struct {
	Notifier
	name        string
	source      chan interface{}
	subscribers map[string]Subscriber
	subLock     *sync.RWMutex // Lock used when reading/modifying subscribers map
	endNotifier chan struct{}
}

func init() {
	notifiers = make(map[string]Notifier)
}

// RegisterNotifier - accepts a name and source channel to make a new notifier object
func RegisterNotifier(name string, source chan interface{}) (Notifier, error) {
	pubLock.Lock() // Adding a notifier
	defer pubLock.Unlock()
	if _, ok := notifiers[name]; ok {
		return nil, fmt.Errorf("A notifier with the name %s already exists", name)
	}

	notifiers[name] = &channelNotifier{
		name:        name,
		source:      source,
		subscribers: make(map[string]Subscriber),
		subLock:     &sync.RWMutex{},
		endNotifier: make(chan struct{}),
	}
	return notifiers[name], nil
}

// Subscribe - subscribes to events on the notifier sending them to the output channel.
func Subscribe(name string, output chan interface{}) (Subscriber, error) {
	pubLock.RLock() // reading the notifiers
	defer pubLock.RUnlock()
	if notifier, ok := notifiers[name]; ok {
		id := guuid.New().String()
		subscriber := &notifierSubscriber{
			id:       id,
			notifier: name,
			output:   output,
		}
		notifier.Subscribe(subscriber)
		return subscriber, nil
	}

	return nil, fmt.Errorf("Could not find notifier %s to subscribe to", name)
}

// Unsubscribe - removes the subscriber, indicated by id, from the notifier, indicated by name
func Unsubscribe(name string, id string) error {
	pubLock.RLock() // reading the notifiers
	defer pubLock.RUnlock()
	if notifier, ok := notifiers[name]; ok {
		return notifier.Unsubscribe(id)
	}

	return fmt.Errorf("Could not find notifier %s to unsubscribe from", name)
}

// GetName - returns the friendly name of this notifier
func (s *channelNotifier) GetName() string {
	return s.name
}

// safeCloseChannel - Recovers from panic on close of closed channel
func safeCloseChannel(ch chan struct{}) (closedRes bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			closedRes = false
		}
	}()
	close(ch)
	return true
}

// Stop - closes all subscribers and stops the subscription loop
func (s *channelNotifier) Stop() {
	safeCloseChannel(s.endNotifier)
	pubLock.Lock() // removing a notifier
	defer pubLock.Unlock()
	delete(notifiers, s.GetName())
}

// Subscribe - adds a subscriber to the subscribers array
func (s *channelNotifier) Subscribe(newSub Subscriber) {
	s.subscribe(newSub)
}

// adds a subscriber to the subscribers array
func (s *channelNotifier) subscribe(newSub Subscriber) {
	s.subLock.Lock() // Adding a subscriber
	defer s.subLock.Unlock()
	s.subscribers[newSub.GetID()] = newSub
}

// Unsubscribe - remove the subscriber identified with id from the notifier list
func (s *channelNotifier) Unsubscribe(id string) error {
	return s.unsubscribe(id)
}

// remove the subscriber identified with id from the notifier list
func (s *channelNotifier) unsubscribe(id string) error {
	s.subLock.Lock() // Removing a dubscriber
	defer s.subLock.Unlock()
	if sub, ok := s.subscribers[id]; ok {
		delete(s.subscribers, id)
		sub.close()
		return nil
	}
	return fmt.Errorf("Could not find subscriber with id: %s", id)
}

func (s *channelNotifier) unsubscribeAll() {
	// sends messages to all of the subscribers
	for _, sub := range s.subscribers {
		s.unsubscribe(sub.GetID())
	}
}

func (s *channelNotifier) sendMsgs(msg interface{}) {
	// sends messages to all of the subscribers
	s.subLock.RLock() // reading the subscribers map
	defer s.subLock.RUnlock()
	for _, sub := range s.subscribers {
		sub.SendMsg(msg)
	}
}

// Start - starts a go routine with the infinite loop waiting for messages
func (s *channelNotifier) Start() {
	go s.start()
}

// infinite loop waiting for messags on the source channel and sending to the subscribers
func (s *channelNotifier) start() {
	for {
		select {
		case <-s.endNotifier: // notifier is closing, close all of its subscribers
			log.Debugf("Received close for notifier %s", s.GetName())
			s.unsubscribeAll()
			close(s.source)
			return
		case msg, ok := <-s.source: // message received, send to all of it's subscribers
			if ok {
				log.Debugf("Received message for notifier %s: sending to %d subscribers", s.GetName(), len(s.subscribers))
				s.sendMsgs(msg)
			}
		}
	}
}
