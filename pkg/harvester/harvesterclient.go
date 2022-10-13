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
	skipPublish      bool
}

// Client for connecting to harvester
type Client struct {
	Cfg         *Config
	Client      api.Client
	URL         string
	logger      log.FieldLogger
	skipPublish bool
}

// NewConfig creates a config for harvester connections
func NewConfig(cfg config.CentralConfig, getToken auth.TokenGetter, seq events.SequenceProvider) *Config {
	parsed, _ := url.Parse(cfg.GetURL())
	hostname := parsed.Hostname()
	port := util.ParsePort(parsed)
	return &Config{
		ClientTimeout:    cfg.GetClientTimeout(),
		Host:             hostname,
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
func NewClient(cfg *Config) *Client {
	if cfg.Protocol == "" {
		cfg.Protocol = "https"
	}
	if cfg.PageSize == 0 {
		cfg.PageSize = defaultEventPageSize
	}

	logger := log.NewFieldLogger().
		WithComponent("Client").
		WithPackage("harvester")

	harvesterURL := fmt.Sprintf("%s://%s:%d/events", cfg.Protocol, cfg.Host, int(cfg.Port))

	return &Client{
		URL:         harvesterURL,
		Cfg:         cfg,
		Client:      newSingleEntryClient(cfg),
		logger:      logger,
		skipPublish: cfg.skipPublish,
	}
}

// ReceiveSyncEvents fetches events based on the sequence id and watch topic self link, and publishes the events to the event channel
func (h *Client) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	h.logger.Trace("receive sync events")
	var lastID int64
	token, err := h.Cfg.TokenGetter()
	if err != nil {
		return lastID, err
	}

	h.logger.Trace("got token successfully")
	morePages := true
	page := 1

	for morePages {
		h.logger.Tracef("more pages %d", page)
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
			h.logger.Tracef("send failure, tenantID %s, last id %d, and err - %s", h.Cfg.TenantID, lastID, err.Error())
			return lastID, err
		}

		h.logger.Tracef("send res code %d", res.Code)
		if res.Code != http.StatusOK {
			return lastID, fmt.Errorf("expected a 200 response but received %d", res.Code)
		}

		pagedEvents := make([]*resourceEntryExternalEvent, 0)
		err = json.Unmarshal(res.Body, &pagedEvents)
		if err != nil {
			h.logger.Tracef("unmarshal error, last id %d, and err - %s", lastID, err.Error())
			return lastID, err
		}

		if len(pagedEvents) < h.Cfg.PageSize {
			h.logger.Tracef("len page events is less than configured page size %d", h.Cfg.PageSize)
			h.logger.Trace("setting more pages to false")
			morePages = false
		}

		for _, event := range pagedEvents {
			lastID = event.Metadata.GetSequenceID()
			h.logger.Tracef("lastID from paged event - %d", lastID)
			if !h.skipPublish && eventCh != nil {
				h.logger.Trace("send to event channel")
				eventCh <- event.toProtoEvent()
			}
		}
		page++
	}

	h.logger.Tracef("last id %d, and err - %s", lastID, err.Error())
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
	h.logger.Trace("event catch up")

	sequenceID := h.Cfg.SequenceProvider.GetSequence()

	h.logger.Tracef("get initial %s", fmt.Sprintf("sequenceID = %d", sequenceID))

	if sequenceID > 0 {
		h.logger.Trace("sequenceID is greater than 0")
		var err error
		lastSequenceID, err := h.ReceiveSyncEvents(link, sequenceID, events)
		h.logger.Tracef("get initial %s", fmt.Sprintf("lastSequenceID = %d", lastSequenceID))
		if err != nil {
			return err
		}

		if lastSequenceID > 0 {
			h.logger.Trace("lastSequenceID is greater than 0")
			// wait for all current sequences to be processed before processing new ones
			for sequenceID < lastSequenceID {
				h.logger.Trace("sequenceID is less than lastSequenceID")
				sequenceID = h.Cfg.SequenceProvider.GetSequence()
				h.logger.Tracef("now sequenceID is %s", fmt.Sprintf("sequenceID = %d", sequenceID))
			}
		} else {
			return nil
		}
	} else {
		return nil
	}
	h.logger.Trace("keep looping")
	return h.EventCatchUp(link, events)
}

func newSingleEntryClient(cfg *Config) api.Client {
	tlsCfg := corecfg.NewTLSConfig().(*corecfg.TLSConfiguration)
	tlsCfg.LoadFrom(cfg.TLSCfg)
	clientTimeout := cfg.ClientTimeout
	if clientTimeout == 0 {
		clientTimeout = util.DefaultKeepAliveTimeout
	}

	return api.NewSingleEntryClient(tlsCfg, cfg.ProxyURL, clientTimeout)
}
