package notification

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const defTimeout = 5

type chanData struct {
	name        string
	msgReceived interface{}      // The message received by the channel
	msgChan     chan interface{} // The channel receiving messages
	closeChan   chan struct{}    // Channel to close the for loop waiting for messages
	notifChan   chan int         // Channel to notify the test to continue validation
}

func createTestChannel(t *testing.T, name string, isSource bool) *chanData {
	thisChan := &chanData{
		name:      name,
		msgChan:   make(chan interface{}),
		closeChan: make(chan struct{}),
		notifChan: make(chan int, 1),
	}

	go func(t *testing.T) {
		if isSource {
			// source channels only need to be stopped in this test, not listend too
			for {
				select {
				case <-thisChan.closeChan:
					return
				}
			}
		} else {
			// output channels need to listen for stops and receive messages
			for {
				select {
				case <-thisChan.closeChan:
					return
				case msg, ok := <-thisChan.msgChan:
					if ok {
						t.Logf("message received %s", thisChan.name)
						thisChan.msgReceived = msg
						thisChan.notifChan <- 1
					}
				}
			}
		}
	}(t)

	return thisChan
}

func channelTimeout(notifChan chan int, timeout int) {
	// Write to a notify channel after waiting timeout seconds
	time.Sleep(time.Duration(timeout) * time.Second)
	select {
	case <-notifChan:
	default:
		notifChan <- 2
	}
}

func closeChannels(cd *chanData) {
	close(cd.closeChan)
	close(cd.msgChan)
	close(cd.notifChan)
	<-cd.closeChan
	<-cd.msgChan
	<-cd.notifChan
}

func TestRegisterNotifier(t *testing.T) {

	// Create a new Notifier
	serv1Name := "regnotifier"
	serv1ChanData := createTestChannel(t, serv1Name, true)
	notifier1, err := RegisterNotifier(serv1Name, serv1ChanData.msgChan)
	// Test GetName
	assert.Equal(t, notifier1.GetName(), serv1Name, "GetName did not return what was expected")
	assert.Nil(t, err, "Error returned by RegisterNotifier: %s", err)
	assert.Len(t, notifiers, 1, "The length of notifiers should have been 1")

	// Attempt to create a notifier with a name that already exists
	_, err = RegisterNotifier(serv1Name, serv1ChanData.msgChan)
	assert.NotNil(t, err, "RegisterNotifier should have returned an err")
	assert.Len(t, notifiers, 1, "The length of notifiers should have been 1")

	closeChannels(serv1ChanData)
	delete(notifiers, serv1Name)
}

func TestOneSubscriber(t *testing.T) {
	// Create a new Notifier
	serv1Name := "onenotifier"
	serv1ChanData := createTestChannel(t, serv1Name, true)
	notifier1, err := RegisterNotifier(serv1Name, serv1ChanData.msgChan)
	notifier1.Start()
	assert.Nil(t, err, "Error returned by RegisterNotifier: %s", err)
	assert.Len(t, notifiers, 1, "The length of notifiers should have been 1")

	// Create an output channel
	output1Name := "oneoutput"
	output1ChanData := createTestChannel(t, output1Name, false)

	// Subscribe to a notifier that dows not exist
	subscriberBad, err := Subscribe("bad-notifier", output1ChanData.msgChan)
	assert.Nil(t, subscriberBad, "Expected the Subscribe call to return a nil Subscriber")
	assert.NotNil(t, err, "Expected an error to be returned, since the notifier name does not exist")

	// Subscribe to a good notifier
	subscriber1, err := Subscribe(serv1Name, output1ChanData.msgChan)
	assert.NotNil(t, subscriber1, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber1.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from the Subscribe, when it was not expected")

	// Send a message to the source channel, receive on 1 subscriber
	testMessage := "test message 1"
	serv1ChanData.msgChan <- testMessage

	// Start channel timeout
	go channelTimeout(output1ChanData.notifChan, defTimeout)
	<-output1ChanData.notifChan // Wait for message to be received

	assert.NotNil(t, output1ChanData.msgReceived, "Expected a message to have been received")
	assert.Equal(t, testMessage, output1ChanData.msgReceived.(string), "The message received was not the expected message")

	closeChannels(serv1ChanData)
	// closeChannels(output1ChanData)
	delete(notifiers, serv1Name)
}

func TestTwoSubscribers(t *testing.T) {
	// Create a new Notifier
	serv1Name := "twonotifier"
	serv1ChanData := createTestChannel(t, serv1Name, true)
	notifier1, err := RegisterNotifier(serv1Name, serv1ChanData.msgChan)
	notifier1.Start()
	assert.Nil(t, err, "Error returned by RegisterNotifier: %s", err)
	assert.Len(t, notifiers, 1, "The length of notifiers should have been 1")

	// Create output channels
	output1Name := "twooutput1"
	output1ChanData := createTestChannel(t, output1Name, false)
	output2Name := "twooutput2"
	output2ChanData := createTestChannel(t, output2Name, false)

	// Add 2 subscribers
	subscriber1, err := Subscribe(serv1Name, output1ChanData.msgChan)
	assert.NotNil(t, subscriber1, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber1.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from Subscribe, when it was not expected")
	subscriber2, err := Subscribe(serv1Name, output2ChanData.msgChan)
	assert.NotNil(t, subscriber2, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber2.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from Subscribe, when it was not expected")
	assert.Len(t, notifier1.(*channelNotifier).subscribers, 2, "Expected the notifier to have 2 subscribers")

	testMessage := "test message 2 subs"
	serv1ChanData.msgChan <- testMessage

	// Start channel timeout
	go channelTimeout(output1ChanData.notifChan, defTimeout)
	go channelTimeout(output2ChanData.notifChan, defTimeout)
	<-output1ChanData.notifChan // Wait for message to be received on output1
	<-output2ChanData.notifChan // Wait for message to be received on output2

	assert.NotNil(t, output1ChanData.msgReceived, "Expected a message to have been received on output1")
	assert.Equal(t, testMessage, output1ChanData.msgReceived.(string), "The message received on output1 was not the expected message")
	assert.NotNil(t, output2ChanData.msgReceived, "Expected a message to have been received on output2")
	assert.Equal(t, testMessage, output2ChanData.msgReceived.(string), "The message received on output2 was not the expected message")

	closeChannels(serv1ChanData)
	closeChannels(output1ChanData)
	closeChannels(output2ChanData)
	delete(notifiers, serv1Name)
}

func TestStopUnsubscribe(t *testing.T) {

	// Create a new Notifier
	serv1Name := "stopnotifier"
	serv1ChanData := createTestChannel(t, serv1Name, true)
	notifier1, err := RegisterNotifier(serv1Name, serv1ChanData.msgChan)
	notifier1.Start()
	assert.Nil(t, err, "Error returned by RegisterNotifier: %s", err)
	assert.Len(t, notifiers, 1, "The length of notifiers should have been 1")

	// Create output channels
	output1Name := "stopoutput1"
	output1ChanData := createTestChannel(t, output1Name, false)
	output2Name := "stopoutput2"
	output2ChanData := createTestChannel(t, output2Name, false)
	output3Name := "stopoutput3"
	output3ChanData := createTestChannel(t, output3Name, false)

	// Add 2 subscribers
	subscriber1, err := Subscribe(serv1Name, output1ChanData.msgChan)
	assert.NotNil(t, subscriber1, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber1.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from Subscribe, when it was not expected")
	subscriber2, err := Subscribe(serv1Name, output2ChanData.msgChan)
	assert.NotNil(t, subscriber2, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber2.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from Subscribe, when it was not expected")
	subscriber3, err := Subscribe(serv1Name, output3ChanData.msgChan)
	assert.NotNil(t, subscriber3, "The returned subscriber should not have been nil")
	assert.NotEqual(t, "", subscriber3.GetID(), "An ID was not returned by the subscribe call")
	assert.Nil(t, err, "An error was returned from Subscribe, when it was not expected")
	assert.Len(t, notifier1.(*channelNotifier).subscribers, 3, " Expected the notifier to have 3 subscribers")

	// Unsubscribe output3
	err = Unsubscribe("fakeName", subscriber3.GetID())
	assert.NotNil(t, err, "Expected an error to be returned when unsubscribing from a non-existent notifier")
	err = notifier1.Unsubscribe("fakeID")
	assert.NotNil(t, err, "Expected an error to be returned when unsubscribing with an ID that does not exist")
	err = Unsubscribe(serv1Name, subscriber3.GetID())
	assert.Nil(t, err, "An error was returned from Unsubscribe, when it was not expected")
	assert.Len(t, notifier1.(*channelNotifier).subscribers, 2, " Expected the notifier to have 2 subscribers")

	// Unsubscribe subscriber2
	subscriber2.Close()
	assert.Len(t, notifier1.(*channelNotifier).subscribers, 1, " Expected the notifier to have 1 subscribers")

	// Close the notifier, the subscribers should close too
	notifier1.Stop()
	val1, ok1 := <-output1ChanData.msgChan
	assert.Nil(t, val1, "Didn't expect a message on the channel")
	assert.False(t, ok1, "Expected the message channel to be closed")

	assert.Len(t, notifier1.(*channelNotifier).subscribers, 0, " Expected the notifier to have 0 subscribers")

	close(serv1ChanData.closeChan)
	delete(notifiers, serv1Name)
}
