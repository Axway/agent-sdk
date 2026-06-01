package transaction

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	insightsEventVersion = "4"
	legDataVersion       = "2"
	summaryDataVersion   = "2"
)

// ReporterInfo holds the agent build metadata for the reporter block.
type ReporterInfo struct {
	AgentVersion    string
	AgentType       string
	AgentSDKVersion string
	AgentName       string
	// ObservationDelta is only meaningful for metric events; leave 0 for transaction events.
	ObservationDelta int64
}

// insightsAPIDetail is the api sub-object shared by leg and summary v2 data.
type insightsAPIDetail struct {
	ID           string             `json:"id"`
	APIServiceID string             `json:"apiServiceId,omitempty"`
	Name         string             `json:"name,omitempty"`
	Owner        *models.OwnerBlock `json:"owner,omitempty"`
}

// insightsConsumerAppDetail is the application sub-object within consumerDetails.
type insightsConsumerAppDetail struct {
	ID            string             `json:"id,omitempty"`
	Name          string             `json:"name,omitempty"`
	ConsumerOrgID string             `json:"consumerOrgId,omitempty"`
	Owner         *models.OwnerBlock `json:"owner,omitempty"`
}

// insightsPublishedProduct is the publishedProduct sub-object within consumerDetails.
type insightsPublishedProduct struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// insightsSubscription is the subscription sub-object within consumerDetails.
type insightsSubscription struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// insightsConsumerDetails is the consumerDetails sub-object for leg and summary v2.
type insightsConsumerDetails struct {
	ConsumerOrgID    string                     `json:"consumerOrgId,omitempty"`
	Marketplace      *insightsMarketplace       `json:"marketplace,omitempty"`
	Application      *insightsConsumerAppDetail `json:"application,omitempty"`
	PublishedProduct *insightsPublishedProduct  `json:"publishedProduct,omitempty"`
	Subscription     *insightsSubscription      `json:"subscription,omitempty"`
}

type insightsMarketplace struct {
	GUID string `json:"guid,omitempty"`
}

// insightsSummaryProduct is the product sub-object for summary v2.
type insightsSummaryProduct struct {
	ID          string             `json:"id,omitempty"`
	Name        string             `json:"name,omitempty"`
	VersionID   string             `json:"versionId,omitempty"`
	VersionName string             `json:"versionName,omitempty"`
	Owner       *models.OwnerBlock `json:"owner,omitempty"`
}

// insightsReporter is the reporter sub-object.
type insightsReporter struct {
	Version          string `json:"version,omitempty"`
	Type             string `json:"type,omitempty"`
	AgentSDKVersion  string `json:"agentSDKVersion,omitempty"`
	AgentName        string `json:"agentName,omitempty"`
	ObservationDelta int64  `json:"observationDelta,omitempty"`
}

// legProtocol is the protocol sub-object for TransactionLegV2Data.
type legProtocol struct {
	Type   string `json:"type,omitempty"`
	URI    string `json:"uri,omitempty"`
	Method string `json:"method,omitempty"`
	Status int    `json:"status"`
}

// insightsProxy is the nested proxy sub-object for leg and summary v2 data.
type insightsProxy struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// TransactionLegV2Data is the data payload for api.transaction.event (version "2").
type TransactionLegV2Data struct {
	Version        string             `json:"version"`
	APICDeployment string             `json:"apicDeployment,omitempty"`
	TransactionID  string             `json:"transactionId,omitempty"`
	ID             string             `json:"id,omitempty"`
	LegID          int                `json:"legId"`
	ParentID       string             `json:"parentId,omitempty"`
	Source         string             `json:"source,omitempty"`
	Destination    string             `json:"destination,omitempty"`
	Status         string             `json:"status,omitempty"`
	Duration       int                `json:"duration"`
	Direction      string             `json:"direction,omitempty"`
	Protocol       *legProtocol       `json:"protocol,omitempty"`
	API            *insightsAPIDetail `json:"api,omitempty"`
	Reporter       *insightsReporter  `json:"reporter,omitempty"`
	Proxy          *insightsProxy     `json:"proxy,omitempty"`
}

// GetStartTime implements metric.V4Data.
func (d *TransactionLegV2Data) GetStartTime() time.Time { return time.Time{} }

// GetType implements metric.V4Data.
func (d *TransactionLegV2Data) GetType() string { return TypeTransactionEvent }

// GetEventID implements metric.V4Data.
func (d *TransactionLegV2Data) GetEventID() string { return d.TransactionID }

// GetLogFields implements metric.V4Data.
func (d *TransactionLegV2Data) GetLogFields() logrus.Fields {
	f := logrus.Fields{"transactionId": d.TransactionID, "legId": d.LegID}
	if d.API != nil {
		f["apiId"] = d.API.ID
	}
	return f
}

// insightsEntryPoint is the entryPoint sub-object for summary v2.
type insightsEntryPoint struct {
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Host   string `json:"host,omitempty"`
}

// insightsResourceRef is a minimal {id} sub-object.
type insightsResourceRef struct {
	ID string `json:"id,omitempty"`
}

// TransactionSummaryV2Data is the data payload for api.transaction.summary (version "2").
type TransactionSummaryV2Data struct {
	Version            string                   `json:"version"`
	APICDeployment     string                   `json:"apicDeployment,omitempty"`
	Status             string                   `json:"status,omitempty"`
	StatusDetail       string                   `json:"statusDetail,omitempty"`
	Duration           int                      `json:"duration"`
	API                *insightsAPIDetail       `json:"api,omitempty"`
	EntryPoint         *insightsEntryPoint      `json:"entryPoint,omitempty"`
	AssetResource      *insightsResourceRef     `json:"assetResource,omitempty"`
	APIServiceRevision *insightsResourceRef     `json:"apiServiceRevision,omitempty"`
	Product            *insightsSummaryProduct  `json:"product,omitempty"`
	ProductPlan        *insightsResourceRef     `json:"productPlan,omitempty"`
	Quota              *insightsResourceRef     `json:"quota,omitempty"`
	Reporter           *insightsReporter        `json:"reporter,omitempty"`
	ConsumerDetails    *insightsConsumerDetails `json:"consumerDetails,omitempty"`
	Proxy              *insightsProxy           `json:"proxy,omitempty"`
}

// GetStartTime implements metric.V4Data.
func (d *TransactionSummaryV2Data) GetStartTime() time.Time { return time.Time{} }

// GetType implements metric.V4Data.
func (d *TransactionSummaryV2Data) GetType() string { return TypeTransactionSummary }

// GetEventID implements metric.V4Data.
func (d *TransactionSummaryV2Data) GetEventID() string { return "" }

// GetLogFields implements metric.V4Data.
func (d *TransactionSummaryV2Data) GetLogFields() logrus.Fields {
	f := logrus.Fields{"status": d.Status}
	if d.API != nil {
		f["apiId"] = d.API.ID
	}
	return f
}

// InsightsEvent is the top-level wire envelope for transaction leg and summary events.
type InsightsEvent struct {
	ID           string                      `json:"id"`
	Org          string                      `json:"org"`
	Event        string                      `json:"event"`
	Version      string                      `json:"version"`
	Timestamp    int64                       `json:"timestamp"`
	Distribution *metric.V4EventDistribution `json:"distribution"`
	Data         metric.V4Data               `json:"data"`
	Session      *metric.V4Session           `json:"session,omitempty"`
}

// eventTypeMap maps LogEvent.Type to the insights event name.
var eventTypeMap = map[string]string{
	TypeTransactionEvent:   "api.transaction.event",
	TypeTransactionSummary: "api.transaction.summary",
}

// BuildTransactionV2Data constructs an InsightsEvent from a LogEvent for the ground agent path.
// cacheManager may be nil in tests; owner resolution will return "unknown" in that case.
func BuildTransactionV2Data(
	logger log.FieldLogger,
	logEvent LogEvent,
	orgID string,
	environmentID string,
	cacheManager cache.Manager,
	reporter ReporterInfo,
) (*InsightsEvent, error) {
	logger.
		WithField("transactionID", logEvent.TransactionID).
		WithField("eventType", logEvent.Type).
		Debug("building insights v2 event")

	eventName, ok := eventTypeMap[logEvent.Type]
	if !ok {
		return nil, fmt.Errorf("unknown logEvent type %q for InsightsEvent construction", logEvent.Type)
	}

	env := &metric.V4EventDistribution{Environment: environmentID}
	ie := &InsightsEvent{
		ID:           uuid.NewString(),
		Org:          orgID,
		Event:        eventName,
		Version:      insightsEventVersion,
		Timestamp:    logEvent.Stamp,
		Distribution: env,
	}

	switch logEvent.Type {
	case TypeTransactionEvent:
		data, err := buildLegV2Data(logEvent, cacheManager, reporter)
		if err != nil {
			return nil, err
		}
		ie.Data = data
		ie.Session = &metric.V4Session{ID: logEvent.TransactionID}
	case TypeTransactionSummary:
		data, err := buildSummaryV2Data(logger, logEvent, cacheManager, reporter)
		if err != nil {
			return nil, err
		}
		ie.Data = data
		ie.Session = &metric.V4Session{ID: logEvent.TransactionID}
	}

	logger.
		WithField("transactionID", logEvent.TransactionID).
		WithField("insightsEventID", ie.ID).
		WithField("eventName", ie.Event).
		Debug("insights v2 event built successfully")

	return ie, nil
}

func buildLegV2Data(logEvent LogEvent, cacheManager cache.Manager, reporter ReporterInfo) (*TransactionLegV2Data, error) {
	txEvent := logEvent.TransactionEvent
	if txEvent == nil {
		return nil, fmt.Errorf("TransactionEvent is nil for logEvent type %q", logEvent.Type)
	}

	legID := parseLegID(txEvent.ID)

	var proto *legProtocol
	if httpProto, ok := txEvent.Protocol.(*Protocol); ok && httpProto != nil {
		proto = &legProtocol{
			Type:   httpProto.Type,
			URI:    httpProto.URI,
			Method: httpProto.Method,
			Status: httpProto.Status,
		}
	}

	apiID := ""
	if txEvent.Source != "" {
		apiID = transutil.ResolveIDWithPrefix(txEvent.Source, "")
	}

	var legProxy *insightsProxy
	proxyID := strings.TrimPrefix(apiID, transutil.SummaryEventProxyIDPrefix)
	if proxyID != "" || txEvent.ProxyName != "" {
		legProxy = &insightsProxy{ID: proxyID, Name: txEvent.ProxyName}
	}

	data := &TransactionLegV2Data{
		Version:        legDataVersion,
		APICDeployment: logEvent.APICDeployment,
		TransactionID:  logEvent.TransactionID,
		ID:             fmt.Sprintf("leg%d", legID), // legID already normalized via parseLegID
		LegID:          legID,
		ParentID:       formatLegID(txEvent.ParentID),
		Source:         txEvent.Source,
		Destination:    txEvent.Destination,
		Status:         txEvent.Status,
		Duration:       txEvent.Duration,
		Direction:      strings.ToLower(txEvent.Direction),
		Protocol:       proto,
		API:            resolveAPIDetailFromCache(apiID, cacheManager),
		Proxy:          legProxy,
		Reporter: &insightsReporter{
			Version:         reporter.AgentVersion,
			Type:            reporter.AgentType,
			AgentSDKVersion: reporter.AgentSDKVersion,
			AgentName:       reporter.AgentName,
		},
	}

	return data, nil
}

func buildSummaryV2Data(logger log.FieldLogger, logEvent LogEvent, cacheManager cache.Manager, reporter ReporterInfo) (*TransactionSummaryV2Data, error) {
	summary := logEvent.TransactionSummary
	if summary == nil {
		return nil, fmt.Errorf("TransactionSummary is nil for logEvent type %q", logEvent.Type)
	}

	apiID, apiName, apiServiceRevisionID := resolveSummaryAPIInfo(summary)

	apiServiceID := ""
	if summary.API != nil {
		apiServiceID = summary.API.APIServiceID
	}

	data := &TransactionSummaryV2Data{
		Version:            summaryDataVersion,
		APICDeployment:     logEvent.APICDeployment,
		Status:             summary.Status,
		StatusDetail:       summary.StatusDetail,
		Duration:           summary.Duration,
		API:                buildSummaryAPIDetail(logger, apiID, apiName, apiServiceID, summary.OwnerInfo, cacheManager),
		EntryPoint:         buildEntryPoint(summary.EntryPoint),
		AssetResource:      buildAssetResourceRef(summary.AssetResource),
		APIServiceRevision: buildAPIServiceRevisionRef(apiServiceRevisionID, summary.API),
		Reporter: &insightsReporter{
			Version:          reporter.AgentVersion,
			Type:             reporter.AgentType,
			AgentSDKVersion:  reporter.AgentSDKVersion,
			AgentName:        reporter.AgentName,
			ObservationDelta: reporter.ObservationDelta,
		},
	}

	if summary.Product != nil && summary.Product.ID != "" {
		data.Product = &insightsSummaryProduct{
			ID:          summary.Product.ID,
			Name:        summary.Product.Name,
			VersionID:   summary.Product.VersionID,
			VersionName: summary.Product.VersionName,
			Owner:       summary.Product.Owner,
		}
	}
	if summary.ProductPlan != nil && summary.ProductPlan.ID != "" {
		data.ProductPlan = &insightsResourceRef{ID: summary.ProductPlan.ID}
	}
	if summary.Quota != nil && summary.Quota.ID != "" {
		data.Quota = &insightsResourceRef{ID: summary.Quota.ID}
	}

	if summary.ConsumerDetails != nil {
		data.ConsumerDetails = buildConsumerDetails(summary.ConsumerDetails, summary.AppOwnerInfo)
	}

	if summary.Proxy != nil && (summary.Proxy.ID != "" || summary.Proxy.Name != "") {
		data.Proxy = &insightsProxy{ID: summary.Proxy.ID, Name: summary.Proxy.Name}
	}

	return data, nil
}

func resolveSummaryAPIInfo(summary *Summary) (apiID, apiName, apiServiceRevisionID string) {
	if summary.Proxy != nil {
		apiID = transutil.ResolveIDWithPrefix(summary.Proxy.ID, summary.Proxy.Name)
		apiName = summary.Proxy.Name
		// Still pick up the revision ID from summary.API when available.
		if summary.API != nil {
			apiServiceRevisionID = summary.API.APIServiceInstance
		}
		return
	}
	if summary.API != nil {
		apiID = transutil.ResolveIDWithPrefix(summary.API.ID, summary.API.Name)
		apiName = summary.API.Name
		apiServiceRevisionID = summary.API.APIServiceInstance
	}
	return
}

func buildSummaryAPIDetail(logger log.FieldLogger, apiID, apiName, apiServiceID string, ownerInfo *models.OwnerBlock, cacheManager cache.Manager) *insightsAPIDetail {
	var apiOwner *models.OwnerBlock
	if ownerInfo != nil {
		apiOwner = ownerInfo
		logger.WithField("apiID", apiID).Trace("using pre-populated api owner from summary")
	} else {
		apiOwner = transutil.ResolveAPIOwner(apiID, cacheManager)
		logger.WithField("apiID", apiID).WithField("ownerType", apiOwner.Type).Trace("resolved api owner from cache")
	}

	detail := &insightsAPIDetail{ID: apiID, Name: apiName, Owner: apiOwner, APIServiceID: apiServiceID}
	if apiServiceID == "" && cacheManager != nil {
		stripped := strings.TrimPrefix(apiID, transutil.SummaryEventProxyIDPrefix)
		if svc := cacheManager.GetAPIServiceWithAPIID(stripped); svc != nil {
			detail.APIServiceID = svc.Metadata.ID
		}
	}
	return detail
}

func buildEntryPoint(ep *EntryPoint) *insightsEntryPoint {
	if ep == nil {
		return nil
	}
	return &insightsEntryPoint{Method: ep.Method, Path: ep.Path, Host: ep.Host}
}

func buildAssetResourceRef(ar *models.AssetResource) *insightsResourceRef {
	if ar == nil || ar.ID == "" || ar.ID == unknown {
		return nil
	}
	return &insightsResourceRef{ID: ar.ID}
}

func buildAPIServiceRevisionRef(revisionID string, api *models.APIDetails) *insightsResourceRef {
	if revisionID == "" && api != nil {
		revisionID = api.APIServiceInstance
	}
	if revisionID == "" {
		return nil
	}
	return &insightsResourceRef{ID: revisionID}
}

func buildConsumerDetails(cd *models.ConsumerDetails, appOwner *models.OwnerBlock) *insightsConsumerDetails {
	out := &insightsConsumerDetails{}

	if cd.Marketplace != nil {
		out.ConsumerOrgID = cd.Marketplace.ConsumerOrgID
		out.Marketplace = &insightsMarketplace{GUID: cd.Marketplace.GUID}
	}

	appDetail := &insightsConsumerAppDetail{}
	if cd.Application != nil {
		appDetail.ID = cd.Application.ID
		appDetail.Name = cd.Application.Name
		appDetail.ConsumerOrgID = cd.Application.ConsumerOrgID
	}
	if appOwner != nil {
		appDetail.Owner = appOwner
	}
	out.Application = appDetail

	if cd.PublishedProduct != nil && cd.PublishedProduct.ID != "" {
		out.PublishedProduct = &insightsPublishedProduct{
			ID:   cd.PublishedProduct.ID,
			Name: cd.PublishedProduct.Name,
		}
	}
	if cd.Subscription != nil && cd.Subscription.ID != "" {
		out.Subscription = &insightsSubscription{
			ID:   cd.Subscription.ID,
			Name: cd.Subscription.Name,
		}
	}

	return out
}

// parseLegID accepts "N" or "legN" and returns N; returns 0 on any other input.
func parseLegID(s string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(s, "leg"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// formatLegID normalizes s to "legN" form. Already-prefixed values ("leg0") pass through unchanged.
// Plain integers ("0") are prefixed. Anything else is returned as-is for backwards compatibility.
func formatLegID(s string) string {
	if strings.HasPrefix(s, "leg") {
		return s
	}
	if _, err := strconv.Atoi(s); err == nil {
		return "leg" + s
	}
	return s
}

// resolveAPIDetailFromCache builds the api sub-object for leg events.
// apiServiceId is intentionally omitted — it is summary-only per REQ-050.
func resolveAPIDetailFromCache(apiID string, cacheManager cache.Manager) *insightsAPIDetail {
	detail := &insightsAPIDetail{
		ID:    apiID,
		Owner: &models.OwnerBlock{Type: "unknown"},
	}
	if cacheManager == nil || apiID == "" {
		return detail
	}
	detail.Owner = transutil.ResolveAPIOwner(apiID, cacheManager)
	return detail
}
