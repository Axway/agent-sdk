package watchmanager

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	defaultEventPageSize = 100
)

type harvesterConfig struct {
	protocol    string
	host        string
	port        uint32
	tenantID    string
	tokenGetter TokenGetter
	tlsCfg      *tls.Config
	proxyURL    string
	pageSize    int
}

type harvesterClient struct {
	url    string
	cfg    *harvesterConfig
	client api.Client
}

func newHarvesterClient(cfg *harvesterConfig) *harvesterClient {
	if cfg.protocol == "" {
		cfg.protocol = "https"
	}
	if cfg.pageSize == 0 {
		cfg.pageSize = defaultEventPageSize
	}
	tlsCfg := corecfg.NewTLSConfig().(*corecfg.TLSConfiguration)
	tlsCfg.LoadFrom(cfg.tlsCfg)
	return &harvesterClient{
		url:    cfg.protocol + "://" + cfg.host + ":" + strconv.Itoa(int(cfg.port)) + "/events",
		cfg:    cfg,
		client: api.NewClient(tlsCfg, cfg.proxyURL),
	}
}

func (h *harvesterClient) receiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	var lastID int64
	token, err := h.cfg.tokenGetter()
	if err != nil {
		return lastID, err
	}

	morePages := true
	page := 1

	for morePages {
		pageableQueryParams := map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(h.cfg.pageSize),
			"query":    fmt.Sprintf("sequenceID>%d", sequenceID),
			"sort":     "sequenceID,ASC",
		}

		req := api.Request{
			Method:      http.MethodGet,
			URL:         h.url + topicSelfLink,
			Headers:     make(map[string]string),
			QueryParams: pageableQueryParams,
		}

		req.Headers["Authorization"] = "Bearer " + token
		req.Headers["X-Axway-Tenant-Id"] = h.cfg.tenantID
		req.Headers["Content-Type"] = "application/json"
		res, err := h.client.Send(req)
		if err != nil {
			return lastID, err
		}

		if res.Code != 200 {
			return lastID, fmt.Errorf("expected a 200 response but received %d", res.Code)
		}

		pagedEvents := make([]*resourceEntryExternalEvent, 0)
		err = json.Unmarshal(res.Body, &pagedEvents)
		if err != nil {
			return lastID, err
		}

		if len(pagedEvents) < h.cfg.pageSize {
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
