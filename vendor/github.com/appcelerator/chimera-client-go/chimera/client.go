package chimera

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

const (
	apiVersion        = "v2"
	publishEndPoint   = "message"
	subscribeEndPoint = "subscription"
	authEndPoint      = "auth"
)

type Client struct {
	options       ClientOptions
	httpClient    *http.Client
	subscriptions map[string]Subscription
}

func NewClient(ctx context.Context, opts ClientOptions) (*Client, error) {
	log.Println("creating new client")

	client := &Client{
		options: opts,
		httpClient: &http.Client{
			Timeout: opts.getTimeout(),
		},
		subscriptions: make(map[string]Subscription),
	}
	transport := opts.getTransport()
	if transport != nil {
		client.httpClient.Transport = transport
	}
	return client, nil
}

func (client *Client) Publish(ctx context.Context, msg Message, opts PublishOptions) (*http.Response, error) {
	log.Println("client publish: ", client.options.Host)

	messages := []Message{msg}
	data, err := json.Marshal(messages)
	if err != nil {
		log.Println("message parsing err: ", err)
		return nil, err
	}

	resp, err := client.request(ctx, publishEndPoint, "POST", data)
	if err != nil {
		log.Println("client publish err: ", err)
		return nil, err
	}

	return resp, nil
}

func (client *Client) PublishMessages(ctx context.Context, messages []Message, opts PublishOptions) (*http.Response, error) {
	log.Println("client publish: ", client.options.Host)

	data, err := json.Marshal(messages)
	if err != nil {
		log.Println("message parsing err: ", err)
		return nil, err
	}

	resp, err := client.request(ctx, publishEndPoint, "POST", data)
	if err != nil {
		log.Println("client publish err: ", err)
		return nil, err
	}

	return resp, nil
}

func (client *Client) dial(ctx context.Context, endpoint string, sub []byte, opts *SubscribeOptions, redialch <-chan bool) chan chan Subscription {
	subscriptions := make(chan chan Subscription)

	go func() {
		subs := make(chan Subscription)
		defer close(subscriptions)

		for {
			resp, err := client.request(ctx, endpoint, "PUT", sub)
			if err != nil {
				log.Println("client subscribe err: ", err)
				return
			}

			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			var meta SubscriptionMeta
			json.Unmarshal(body, &meta)
			log.Println("subscribe meta:", meta)

			if meta.URI == "" {
				log.Println("no subscription meta.")
				return
			}

			select {
			case subscriptions <- subs:
			case <-ctx.Done():
				log.Println("shutting down subscription")
				return
			}

			for _, queue := range meta.Queues {
				conn, err := amqp.Dial(meta.URI)
				if err != nil {
					log.Fatalf("cannot (re)dial: %v: %q", err, meta.URI)
				}

				ch, err := conn.Channel()
				if err != nil {
					log.Fatalf("cannot create channel: %v", err)
				}

				select {
				case subs <- Subscription{meta.URI, queue, conn, ch}:
				case <-ctx.Done():
					log.Println("shutting down subscription")
					ch.Close()
					return
				}
			}
			<-redialch
		}
	}()

	return subscriptions
}

func (client *Client) Subscribe(ctx context.Context, subscribe Subscribe, opts SubscribeOptions, messagesChan chan<- []byte) {
	sub, err := json.Marshal(&subscribe)
	if err != nil {
		log.Println("subscribe parsing err: ", err)
		return
	}

	redialCh := make(chan bool)

	subscriptions := client.dial(ctx, subscribeEndPoint, sub, &opts, redialCh)
	for subscription := range subscriptions {
		sub := <-subscription
		deliveries, err := sub.channel.Consume(
			sub.queue, // queue
			"",        // consumer
			true,      // auto ack
			false,     // exclusive
			false,     // no local
			false,     // no wait
			nil,       // args
		)

		if err != nil {
			log.Printf("resubscribing ...")
			sub.channel.Close()
			redialCh <- true
			continue
		}

		log.Printf("subscribed...")

		go func() {
			for msg := range deliveries {
				messagesChan <- []byte(msg.Body)
			}
		}()

		if opts.Resubscribe > 0 {
			time.Sleep(time.Duration(opts.Resubscribe) * time.Second)
			log.Printf("resubscribing...")
			sub.channel.Close()
			redialCh <- true
		}
	}

	close(messagesChan)
}

func (client *Client) Ping(ctx context.Context) bool {
	resp, err := client.request(ctx, authEndPoint, "GET", nil)
	if err != nil {
		return false
	}

	return resp.StatusCode == 200
}

func (client *Client) request(ctx context.Context, endpoint string, method string, data []byte) (*http.Response, error) {
	uri := client.options.Protocol.String() + "://" + client.options.Host + "/" + apiVersion + "/" + endpoint
	req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
	if err != nil {
		log.Println("new request err: ", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", client.options.AuthKey)
	req.ContentLength = int64(binary.Size(data))

	resp, err := client.httpClient.Do(req)
	if err != nil {
		log.Println("client request err: ", err)
		return nil, err
	}

	return resp, nil
}
