package harvester

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	defaultEventPageSize = 100
)

// Harvest is an interface for retrieving harvester events
type Harvest interface {
	EventCatchUp(link string, events chan *proto.Event) error
	ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error)
}

// Config for harvester
type Config struct {
	ClientTimeout    time.Duration
	Host             string
	PageSize         int
	Port             uint32
	Protocol         string
	ProxyURL         string
	SequenceProvider events.SequenceProvider
	TenantID         string
	TLSCfg           *tls.Config
	TokenGetter      func() (string, error)
}

// Client for connecting to harvester
type Client struct {
	Cfg              *Config
	Client           api.Client
	URL              string
	cacheBuildSignal chan interface{}
	logger           log.FieldLogger
}

// NewConfig creates a config for harvester connections
func NewConfig(cfg config.CentralConfig, getToken auth.TokenGetter, seq events.SequenceProvider) *Config {
	parsed, _ := url.Parse(cfg.GetURL())
	port := util.ParsePort(parsed)

	return &Config{
		ClientTimeout:    cfg.GetClientTimeout(),
		Host:             parsed.Host,
		PageSize:         100,
		Port:             uint32(port),
		Protocol:         parsed.Scheme,
		ProxyURL:         cfg.GetProxyURL(),
		SequenceProvider: seq,
		TenantID:         cfg.GetTenantID(),
		TLSCfg:           cfg.GetTLSConfig().BuildTLSConfig(),
		TokenGetter:      getToken.GetToken,
	}
}

// NewClient creates a new harvester client
func NewClient(cfg *Config, cacheBuildSignal chan interface{}) *Client {
	if cfg.Protocol == "" {
		cfg.Protocol = "https"
	}
	if cfg.PageSize == 0 {
		cfg.PageSize = defaultEventPageSize
	}
	tlsCfg := corecfg.NewTLSConfig().(*corecfg.TLSConfiguration)
	tlsCfg.LoadFrom(cfg.TLSCfg)
	clientTimeout := cfg.ClientTimeout
	if clientTimeout == 0 {
		clientTimeout = util.DefaultKeepAliveTimeout
	}

	return &Client{
		cacheBuildSignal: cacheBuildSignal,
		URL:              cfg.Protocol + "://" + cfg.Host + ":" + strconv.Itoa(int(cfg.Port)) + "/events",
		Cfg:              cfg,
		Client:           api.NewSingleEntryClient(tlsCfg, cfg.ProxyURL, clientTimeout),
		logger:           log.NewFieldLogger().WithComponent("Client").WithPackage("harvester"),
	}
}

// ReceiveSyncEvents fetches events based on the sequence id and watch topic self link, and publishes the events to the event channel
func (h *Client) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	var lastID int64
	token, err := h.Cfg.TokenGetter()
	if err != nil {
		return lastID, err
	}

	morePages := true
	page := 1

	for morePages {
		pageableQueryParams := h.buildParams(sequenceID, page, h.Cfg.PageSize)

		req := api.Request{
			Method:      http.MethodGet,
			URL:         h.URL + topicSelfLink,
			Headers:     make(map[string]string),
			QueryParams: pageableQueryParams,
		}

		req.Headers["Authorization"] = "Bearer " + token
		req.Headers["X-Axway-Tenant-Id"] = h.Cfg.TenantID
		req.Headers["Content-Type"] = "application/json"
		res, err := h.Client.Send(req)
		if err != nil {
			// send signal to discovery cache
			h.signalErr(err)
			return lastID, err
		}

		if res.Code != 200 {
			h.signalErr(err)
			return lastID, fmt.Errorf("expected a 200 response but received %d", res.Code)
		}

		pagedEvents := make([]*resourceEntryExternalEvent, 0)
		err = json.Unmarshal(res.Body, &pagedEvents)
		if err != nil {
			h.signalErr(err)
			return lastID, err
		}

		if len(pagedEvents) < h.Cfg.PageSize {
			morePages = false
		}

		for _, event := range pagedEvents {
			lastID = event.Metadata.GetSequenceID()
			eventCh <- event.toProtoEvent()
		}
		page++
	}

	return lastID, err
}

func (h *Client) buildParams(sequenceID int64, page, pageSize int) map[string]string {
	if sequenceID > 0 {
		return map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(pageSize),
			"query":    fmt.Sprintf("sequenceID>%d", sequenceID),
			"sort":     "sequenceID,ASC",
		}
	}

	// if the sequence id is 0, then there are no events to catch up on,
	// so make a request to get the latest event so that we can save the sequence id to the cache.

	return map[string]string{
		"pageSize": strconv.Itoa(1),
		"sort":     "sequenceID,DESC",
	}
}

// EventCatchUp syncs all events
func (h *Client) EventCatchUp(link string, events chan *proto.Event) error {
	if h.Client == nil || h.Cfg.SequenceProvider == nil {
		return nil
	}

	sequenceID := h.Cfg.SequenceProvider.GetSequence()
	if sequenceID > 0 {
		var err error
		lastSequenceID, err := h.ReceiveSyncEvents(link, sequenceID, events)
		if err != nil {
			return err
		}

		if lastSequenceID > 0 {
			// wait for all current sequences to be processed before processing new ones
			for sequenceID < lastSequenceID {
				sequenceID = h.Cfg.SequenceProvider.GetSequence()
			}
		} else {
			return nil
		}
	} else {
		return nil
	}

	return h.EventCatchUp(link, events)
}

func (h *Client) signalErr(err error) {
	h.Cfg.SequenceProvider.SetSequence(0)
	h.logger.WithError(err).Info("sending signal to rebuild cache")
	if h.cacheBuildSignal != nil {
		go func() {
			h.cacheBuildSignal <- struct{}{}
		}()
	}
}
