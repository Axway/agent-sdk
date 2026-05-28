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
	Owner *models.OwnerBlock `json:"owner,omitempty"`
}

// insightsConsumerDetails is the consumerDetails sub-object for leg and summary v2.
type insightsConsumerDetails struct {
	ConsumerOrgID string                     `json:"consumerOrgId,omitempty"`
	Marketplace   *insightsMarketplace       `json:"marketplace,omitempty"`
	Application   *insightsConsumerAppDetail `json:"application,omitempty"`
}

type insightsMarketplace struct {
	GUID string `json:"guid,omitempty"`
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
	// Deprecated fields — populated for backward compatibility only (omitempty suppresses when empty)
	ProxyID   string `json:"proxy.id,omitempty"`
	ProxyName string `json:"proxy.name,omitempty"`
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
	Reporter           *insightsReporter        `json:"reporter,omitempty"`
	ConsumerDetails    *insightsConsumerDetails `json:"consumerDetails,omitempty"`
	// Deprecated fields — populated for backward compatibility only
	ProxyID   string `json:"proxy.id,omitempty"`
	ProxyName string `json:"proxy.name,omitempty"`
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
		data, err := buildLegV2Data(logger, logEvent, cacheManager, reporter)
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

func buildLegV2Data(logger log.FieldLogger, logEvent LogEvent, cacheManager cache.Manager, reporter ReporterInfo) (*TransactionLegV2Data, error) {
	txEvent := logEvent.TransactionEvent
	if txEvent == nil {
		return nil, fmt.Errorf("TransactionEvent is nil for logEvent type %q", logEvent.Type)
	}

	legID := 0
	if txEvent.ID != "" {
		parsed, err := strconv.Atoi(txEvent.ID)
		if err != nil {
			logger.WithField("legID", txEvent.ID).Warn("leg ID is not a valid integer, defaulting to 0. Check agent for a numeric leg ID")
		} else if parsed > 0 {
			legID = parsed
		}
	}

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

	data := &TransactionLegV2Data{
		Version:        legDataVersion,
		APICDeployment: logEvent.APICDeployment,
		TransactionID:  logEvent.TransactionID,
		ID:             fmt.Sprintf("leg%d", legID),
		LegID:          legID,
		ParentID:       txEvent.ParentID,
		Source:         txEvent.Source,
		Destination:    txEvent.Destination,
		Status:         txEvent.Status,
		Duration:       txEvent.Duration,
		Direction:      txEvent.Direction,
		Protocol:       proto,
		API:            resolveAPIDetailFromCache(apiID, cacheManager),
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
			Version:         reporter.AgentVersion,
			Type:            reporter.AgentType,
			AgentSDKVersion: reporter.AgentSDKVersion,
			AgentName:       reporter.AgentName,
		},
	}

	if summary.ConsumerDetails != nil {
		data.ConsumerDetails = buildConsumerDetails(summary.ConsumerDetails, summary.AppOwnerInfo)
	}

	if summary.Proxy != nil {
		data.ProxyID = summary.Proxy.ID
		data.ProxyName = summary.Proxy.Name
	}

	return data, nil
}

func resolveSummaryAPIInfo(summary *Summary) (apiID, apiName, apiServiceRevisionID string) {
	if summary.Proxy != nil {
		apiID = transutil.ResolveIDWithPrefix(summary.Proxy.ID, summary.Proxy.Name)
		apiName = summary.Proxy.Name
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
	if appOwner != nil {
		appDetail.Owner = appOwner
	}
	out.Application = appDetail

	return out
}

// resolveAPIDetailFromCache builds an insightsAPIDetail using only the cache, with no owner lookup
// (used for leg events where the proxy ID may not be available in the leg itself).
func resolveAPIDetailFromCache(apiID string, cacheManager cache.Manager) *insightsAPIDetail {
	detail := &insightsAPIDetail{
		ID:    apiID,
		Owner: &models.OwnerBlock{Type: "unknown"},
	}
	if cacheManager == nil || apiID == "" {
		return detail
	}
	stripped := strings.TrimPrefix(apiID, transutil.SummaryEventProxyIDPrefix)
	if svc := cacheManager.GetAPIServiceWithAPIID(stripped); svc != nil {
		detail.APIServiceID = svc.Metadata.ID
	}
	detail.Owner = transutil.ResolveAPIOwner(apiID, cacheManager)
	return detail
}
