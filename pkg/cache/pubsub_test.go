package cache

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/notification"

	"github.com/stretchr/testify/assert"
)

func TestPubSub(t *testing.T) {
	// CreateTopic
	topic1 := "topic1"
	data1 := "topic1 data1"
	_, err := CreateTopic("niltopic")
	assert.Nil(t, err, "Unexpected error hit creating a topic with nil as the initial data")
	createpubsub, err := CreateTopicWithInitData(topic1, data1)
	assert.Nil(t, err, "Unexpected error hit in Create Topic")
	assert.IsType(t, &cachePubSub{}, createpubsub, "Returned object not of cachePubSub type")
	pubsub, err := CreateTopicWithInitData(topic1, data1)
	assert.NotNil(t, err, "Expected a duplicate topic error")
	assert.Nil(t, pubsub, "The returned PubSub object should have been nil")
	tempName := "tempName"
	notification.RegisterNotifier(tempName, nil)
	pubsub, err = CreateTopicWithInitData(tempName, data1)
	assert.NotNil(t, err, "Expected a duplicate topic error")
	assert.Nil(t, pubsub, "The returned PubSub object should have been nil")

	// RemoveTopic
	removetopic := "removetopic"
	assert.Len(t, topics, 2, "The length of topics was not what was expected")
	_, err = CreateTopic(removetopic)
	assert.Len(t, topics, 3, "Expected a new topic in the topics array")
	assert.Nil(t, err, "Unexpected error hit creating a topic with nil as the initial data")
	err = RemoveTopic(removetopic)
	assert.Len(t, topics, 2, "Expected the topics array to be 1 less")
	assert.Nil(t, err, "Unexpected error hit removing a topic")
	err = RemoveTopic("badtopicname")
	assert.Len(t, topics, 2, "Expected the topics array length to not have changed")
	assert.NotNil(t, err, "Expected an error to be returned from a bad topic name")
	CreateTopic(removetopic)
	assert.Len(t, topics, 3, "Expected the topics array length to have grown")
	globalCache.Delete(removetopic)
	err = RemoveTopic(removetopic)
	assert.Len(t, topics, 2, "Expected the topics array length to have been 1 less")
	assert.NotNil(t, err, "Expected an error to be returned when removing a topic without a cache item")

	// GetPubSub
	getpubsub, err := GetPubSub(topic1)
	assert.Nil(t, err, "Unexpected error hit in Create Topic")
	assert.IsType(t, &cachePubSub{}, getpubsub, "Returned object not of cachePubSub type")
	assert.Equal(t, createpubsub, getpubsub, "Expected the PubSub object to be the same as the one previously created")
	pubsub, err = GetPubSub(tempName)
	assert.NotNil(t, err, "Expected a could not find topic error")
	assert.Nil(t, pubsub, "The returned PubSub object should have been nil")

	// Publish and Subscribe
	topic2 := "topic2"
	data2 := "topic2 data1"
	pubsub2, err := CreateTopicWithInitData(topic2, data2)
	assert.Nil(t, err, "Unexpected error hit in Create Topic")
	assert.NotNil(t, pubsub2, "Unexpected nil for pubsub object")
	subChan, id := pubsub2.Subscribe()
	assert.NotNil(t, id, "Expected an ID to be returned from Subscribe")

	dataReceived := ""
	dataChan := make(chan struct{})
	go func() {
		for {
			data, ok := <-subChan
			if !ok {
				return
			}
			dataReceived = data.(string)
			dataChan <- struct{}{}
		}
	}()

	err = pubsub2.Publish("topic2", "", map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "Expected error since data can't be marshaled")
	err = pubsub2.Publish("topic2", "", data2)
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, "", dataReceived, "Data changed unexpectedly")
	data2a := "topic2 data2"
	err = pubsub2.Publish("topic2", "", data2a)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2a, dataReceived, "Data changed successfully")

	// PublishToTopic
	data2b := "topic2 data2b"
	err = pubsub2.PublishToTopic(data2b)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2b, dataReceived, "Data changed successfully")

	// PublishToTopicWithSecondaryKey
	data2c := "topic2 data2c"
	err = pubsub2.PublishToTopicWithSecondaryKey("", data2c)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2c, dataReceived, "Data changed successfully")

	// PublishCacheHash
	data2d := "topic2 data2d"
	err = pubsub2.PublishCacheHash("topic2", "", data2d)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2d, dataReceived, "Data changed successfully")

	// PublishCacheHashToTopic
	data2e := "topic2 data2e"
	err = pubsub2.PublishCacheHashToTopic(data2e)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2e, dataReceived, "Data changed successfully")

	// PublishCacheHashToTopicWithSecondaryKey
	data2f := "topic2 data2f"
	err = pubsub2.PublishCacheHashToTopicWithSecondaryKey("", data2f)
	<-dataChan // Wait for the go function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data2f, dataReceived, "Data changed successfully")

	// Publish and SubscribeWithCallback
	topic3 := "topic3"
	data3 := "topic3 data1"
	pubsub3, _ := CreateTopicWithInitData(topic3, data3)

	dataReceived = ""
	cbCalled := make(chan struct{})
	cbFunc := func(data interface{}) {
		dataReceived = data.(string)
		close(cbCalled)
	}
	subID := pubsub3.SubscribeWithCallback(cbFunc)
	assert.NotNil(t, subID, "Expected an ID to be returned from Subscribe")

	err = pubsub3.Publish("topic3", "", map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "Expected error since data can't be marshaled")
	err = pubsub3.Publish("topic3", "", data3)
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, "", dataReceived, "Data changed unexpectedly")
	data3a := "topic3 data3"
	err = pubsub3.Publish("topic3", "", data3a)
	<-cbCalled // Wait for the callback function to have been executed
	assert.Nil(t, err, "Unexpected error hit in Publish")
	assert.Equal(t, data3a, dataReceived, "Data changed successfully")

	// Unsubscribe
	err = pubsub2.Unsubscribe(id)
	assert.Nil(t, err, "Unexpected error hit in Unsubscribe")
	err = pubsub3.Unsubscribe(subID)
	assert.Nil(t, err, "Unexpected error hit in Unsubscribe")
}
