package harvester

import (
	"context"
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

// ErrSeqGone - error for purged sequence
type ErrSeqGone struct {
}

func (e *ErrSeqGone) Error() string {
	return "sequence purged"
}

// Harvest is an interface for retrieving harvester events
type Harvest interface {
	EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error
	ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error)
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
		PageSize:         cfg.GetPageSize(),
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
func (h *Client) ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	h.logger.Tracef("receive sync events based on sequence id %v, and self link %v", sequenceID, topicSelfLink)
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

		msg := "sending request for URL - %s"
		if morePages {
			msg += ", more pages"
		}
		h.logger.Tracef(msg, req.URL)
		res, err := h.Client.Send(req)
		if err != nil {
			return lastID, err
		}

		if res.Code != http.StatusOK && res.Code != http.StatusGone {
			return lastID, fmt.Errorf("expected a 200 response but received %d", res.Code)
		}

		// requested sequence is purged get the current max sequence
		if lastID == 0 && res.Code == http.StatusGone {
			maxSeqId, ok := res.Headers["X-Axway-Max-Sequence-Id"]
			if ok && len(maxSeqId) > 0 {
				lastID, err = strconv.ParseInt(maxSeqId[0], 10, 64)
				if err != nil {
					return lastID, err
				}
				return lastID, &ErrSeqGone{}
			}
		}

		pagedEvents := make([]*resourceEntryExternalEvent, 0)
		err = json.Unmarshal(res.Body, &pagedEvents)
		if err != nil {
			return lastID, err
		}

		if len(pagedEvents) < h.Cfg.PageSize {
			morePages = false
		}

		for _, event := range pagedEvents {
			lastID = event.Metadata.GetSequenceID()
			if ctx.Err() != nil {
				h.logger.WithError(ctx.Err()).Error("context was cancelled, stopping event processing")
				return lastID, ctx.Err()
			}
			if !h.skipPublish && eventCh != nil {
				eventCh <- event.toProtoEvent()
			}
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
func (h *Client) EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error {
	h.logger.Trace("event catchup, to sync all events")
	// TODO REMOVE THESE LOG LINES
	// Will remove after testing
	h.logger.Info("--------- starting event catchup to sync all events")
	defer h.logger.Info("--------- finished event catchup to sync all events")
	if h.Client == nil || h.Cfg.SequenceProvider == nil {
		return nil
	}

	// TODO should this timeout duration be changed?
	// allow up to a minute to sync events
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	return h.syncEvents(ctx, link, events)
}

func (h *Client) syncEvents(ctx context.Context, link string, events chan *proto.Event) error {
	sequenceID := h.Cfg.SequenceProvider.GetSequence()
	if sequenceID > 0 {
		var err error
		lastSequenceID, err := h.ReceiveSyncEvents(ctx, link, sequenceID, events)
		if err != nil {
			if _, ok := err.(*ErrSeqGone); ok {
				// Set the max sequence returned from 410 to sequence provider as processed
				h.Cfg.SequenceProvider.SetSequence(lastSequenceID)
				return nil
			}
			return err
		}

		// TODO REMOVE THESE LOG LINES
		// Will remove after testing
		if lastSequenceID > 0 {
			// wait for all current sequences to be processed before processing new ones
			for sequenceID < lastSequenceID {
				if ctx.Err() != nil {
					h.logger.WithError(ctx.Err()).Error("--------- context was cancelled, stopping event processing")
					return ctx.Err()
				}
				sequenceID = h.Cfg.SequenceProvider.GetSequence()
				h.logger.Info("--------- looping to wait for sequence to be processed, current: ", sequenceID, " last: ", lastSequenceID)
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			return nil
		}
	} else {
		return nil
	}

	return h.syncEvents(ctx, link, events)
}

func newSingleEntryClient(cfg *Config) api.Client {
	tlsCfg := corecfg.NewTLSConfig().(*corecfg.TLSConfiguration)
	tlsCfg.LoadFrom(cfg.TLSCfg)
	clientTimeout := cfg.ClientTimeout
	if clientTimeout == 0 {
		clientTimeout = util.DefaultKeepAliveTimeout
	}

	return api.NewClient(tlsCfg, cfg.ProxyURL,
		api.WithTimeout(clientTimeout), api.WithSingleURL())
}
