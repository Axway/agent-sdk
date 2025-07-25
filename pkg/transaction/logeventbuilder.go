package transaction

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util"
)

const defaultAPICDeployment = "prod"

// IsValid - Validator for TxEventStatus
func (s TxEventStatus) IsValid() error {
	switch s {
	case TxEventStatusPass, TxEventStatusFail:
		return nil
	}
	return errors.New("invalid transaction event status")
}

// IsValid - Validator for TxSummaryStatus
func (s TxSummaryStatus) IsValid() error {
	switch s {
	case TxSummaryStatusSuccess, TxSummaryStatusFailure, TxSummaryStatusException, TxSummaryStatusUnknown:
		return nil
	}
	return errors.New("invalid transaction summary status")
}

// EventBuilder - Interface to build the log event of type transaction event
type EventBuilder interface {
	SetTimestamp(timestamp int64) EventBuilder
	SetTransactionID(transactionID string) EventBuilder
	SetAPICDeployment(apicDeployment string) EventBuilder
	SetEnvironmentName(environmentName string) EventBuilder
	SetEnvironmentID(environmentID string) EventBuilder
	SetTenantID(tenantID string) EventBuilder
	SetTrcbltPartitionID(trcbltPartitionID string) EventBuilder
	SetTargetPath(targetPath string) EventBuilder
	SetResourcePath(resourcePath string) EventBuilder

	SetID(id string) EventBuilder
	SetParentID(parentID string) EventBuilder
	SetSource(source string) EventBuilder
	SetDestination(destination string) EventBuilder
	SetDuration(duration int) EventBuilder
	SetDirection(direction string) EventBuilder
	SetStatus(status TxEventStatus) EventBuilder
	SetProtocolDetail(protocolDetail interface{}) EventBuilder
	SetRedactionConfig(config redaction.Redactions) EventBuilder

	Build() (*LogEvent, error)
}

type transactionEventBuilder struct {
	EventBuilder
	err             error
	logEvent        *LogEvent
	redactionConfig redaction.Redactions
}

// SummaryBuilder - Interface to build the log event of type transaction summary
type SummaryBuilder interface {
	SetTimestamp(timestamp int64) SummaryBuilder
	SetTransactionID(transactionID string) SummaryBuilder
	SetAPICDeployment(apicDeployment string) SummaryBuilder
	SetEnvironmentName(environmentName string) SummaryBuilder
	SetEnvironmentID(environmentID string) SummaryBuilder
	SetTenantID(tenantID string) SummaryBuilder
	SetTrcbltPartitionID(trcbltPartitionID string) SummaryBuilder
	SetTargetPath(targetPath string) SummaryBuilder
	SetResourcePath(resourcePath string) SummaryBuilder

	SetStatus(status TxSummaryStatus, statusDetail string) SummaryBuilder
	SetDuration(duration int) SummaryBuilder
	SetApplication(appID, appName string) SummaryBuilder
	SetProduct(id, name, version string) SummaryBuilder
	SetTeam(teamID string) SummaryBuilder
	SetProxy(proxyID, proxyName string, proxyRevision int) SummaryBuilder
	SetProxyWithStage(proxyID, proxyName, proxyStage string, proxyRevision int) SummaryBuilder
	SetProxyWithStageVersion(proxyID, proxyName, proxyStage, proxyVersion string, proxyRevision int) SummaryBuilder
	SetRunTime(runtimeID, runtimeName string) SummaryBuilder
	SetEntryPoint(entryPointType, method, path, host string) SummaryBuilder
	SetIsInMetricEvent(isInMetricEvent bool) SummaryBuilder
	SetRedactionConfig(config redaction.Redactions) SummaryBuilder
	Build() (*LogEvent, error)
}

type transactionSummaryBuilder struct {
	SummaryBuilder
	err             error
	logEvent        *LogEvent
	redactionConfig redaction.Redactions
}

// NewTransactionEventBuilder - Creates a new log event builder
func NewTransactionEventBuilder() EventBuilder {
	txEventBuilder := &transactionEventBuilder{
		logEvent: &LogEvent{
			Version:            "1.0",
			Stamp:              time.Now().Unix(),
			Type:               TypeTransactionEvent,
			TransactionEvent:   &Event{},
			TransactionSummary: nil,
			APICDeployment:     defaultAPICDeployment,
		},
	}

	cfg := agent.GetCentralConfig()
	// Check interface and value are not nil
	if cfg != nil && !reflect.ValueOf(cfg).IsNil() {
		txEventBuilder.logEvent.TenantID = cfg.GetTenantID()
		txEventBuilder.logEvent.EnvironmentName = cfg.GetEnvironmentName()
		txEventBuilder.logEvent.EnvironmentID = cfg.GetEnvironmentID()
		txEventBuilder.logEvent.APICDeployment = cfg.GetAPICDeployment()
	}
	return txEventBuilder
}

// NewTransactionSummaryBuilder - Creates a new log event builder
func NewTransactionSummaryBuilder() SummaryBuilder {
	txSummaryBuilder := &transactionSummaryBuilder{
		logEvent: &LogEvent{
			Version:            "1.0",
			Stamp:              time.Now().Unix(),
			Type:               TypeTransactionSummary,
			TransactionEvent:   nil,
			TransactionSummary: &Summary{},
			APICDeployment:     defaultAPICDeployment,
		},
	}

	cfg := agent.GetCentralConfig()
	// Check interface and value are not nil
	if cfg != nil && !reflect.ValueOf(cfg).IsNil() {
		txSummaryBuilder.logEvent.TenantID = cfg.GetTenantID()
		txSummaryBuilder.logEvent.EnvironmentName = cfg.GetEnvironmentName()
		txSummaryBuilder.logEvent.EnvironmentID = cfg.GetEnvironmentID()
		txSummaryBuilder.logEvent.APICDeployment = cfg.GetAPICDeployment()
		txSummaryBuilder.logEvent.TransactionSummary.IsInMetricEvent =
			cfg.GetMetricReportingConfig().CanPublish()
	}
	return txSummaryBuilder
}

func (b *transactionEventBuilder) SetTimestamp(timestamp int64) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.Stamp = timestamp
	return b
}

func (b *transactionEventBuilder) SetTransactionID(transactionID string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionID = transactionID
	return b
}

func (b *transactionEventBuilder) SetAPICDeployment(apicDeployment string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.APICDeployment = apicDeployment
	return b
}

func (b *transactionEventBuilder) SetEnvironmentName(environmentName string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.EnvironmentName = environmentName
	return b
}

func (b *transactionEventBuilder) SetEnvironmentID(environmentID string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.EnvironmentID = environmentID
	return b
}

func (b *transactionEventBuilder) SetTenantID(tenantID string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TenantID = tenantID
	return b
}

func (b *transactionEventBuilder) SetTrcbltPartitionID(trcbltPartitionID string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TrcbltPartitionID = trcbltPartitionID
	return b
}

func (b *transactionEventBuilder) SetTargetPath(targetPath string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TargetPath = targetPath
	return b
}

func (b *transactionEventBuilder) SetResourcePath(resourcePath string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.ResourcePath = resourcePath
	return b
}

func (b *transactionEventBuilder) SetID(id string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.ID = id
	return b
}

func (b *transactionEventBuilder) SetParentID(parentID string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.ParentID = parentID
	return b
}

func (b *transactionEventBuilder) SetSource(source string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.Source = source
	return b
}

func (b *transactionEventBuilder) SetDestination(destination string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.Destination = destination
	return b
}

func (b *transactionEventBuilder) SetDuration(duration int) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.Duration = duration
	return b
}

func (b *transactionEventBuilder) SetDirection(direction string) EventBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionEvent.Direction = direction
	return b
}

func (b *transactionEventBuilder) SetStatus(status TxEventStatus) EventBuilder {
	if b.err != nil {
		return b
	}
	err := status.IsValid()
	if err != nil {
		b.err = err
		return b
	}
	b.logEvent.TransactionEvent.Status = string(status)
	return b
}

func (b *transactionEventBuilder) SetProtocolDetail(protocolDetail interface{}) EventBuilder {
	if b.err != nil {
		return b
	}
	_, ok := protocolDetail.(*Protocol)
	if !ok {
		_, ok = protocolDetail.(*JMSProtocol)
	}
	if ok {
		b.logEvent.TransactionEvent.Protocol = protocolDetail
	} else {
		b.err = errors.New("unsupported protocol type")
	}
	return b
}

func (b *transactionEventBuilder) SetRedactionConfig(config redaction.Redactions) EventBuilder {
	if b.err != nil {
		return b
	}
	b.redactionConfig = config
	return b
}

func (b *transactionEventBuilder) Build() (*LogEvent, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Set Target Path
	if b.redactionConfig == nil {
		b.logEvent.TargetPath, _ = redaction.URIRedaction(b.logEvent.TargetPath)
	} else {
		b.logEvent.TargetPath, _ = b.redactionConfig.URIRedaction(b.logEvent.TargetPath)
	}

	//Set Resource Path
	if b.redactionConfig == nil {
		b.logEvent.ResourcePath, _ = redaction.URIRedaction(b.logEvent.ResourcePath)
	} else {
		b.logEvent.ResourcePath, _ = b.redactionConfig.URIRedaction(b.logEvent.ResourcePath)
	}

	if b.logEvent.TrcbltPartitionID == "" {
		b.logEvent.TrcbltPartitionID = b.logEvent.TenantID
	}

	err := b.validateLogEvent()
	if err != nil {
		return nil, err
	}

	return b.logEvent, nil
}

func (b *transactionEventBuilder) validateLogEvent() error {
	if agent.GetCentralConfig() == nil {
		return nil
	}
	if util.IsNotTest() && agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// Do not need this information in offline mode
		return nil
	}

	if b.logEvent.TenantID == "" {
		return errors.New("tenant ID property not set in transaction event")
	}

	if b.logEvent.EnvironmentID == "" {
		return errors.New("environment ID property are not set in transaction event")
	}

	if b.logEvent.TransactionEvent.ID == "" {
		return errors.New("id property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Direction == "" {
		return errors.New("direction property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Status == "" {
		return errors.New("status property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Protocol == nil {
		return errors.New("protocol details not set in transaction event")
	}
	return nil
}

func (b *transactionSummaryBuilder) SetTimestamp(timestamp int64) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.Stamp = timestamp
	return b
}

func (b *transactionSummaryBuilder) SetTransactionID(transactionID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionID = transactionID
	return b
}

func (b *transactionSummaryBuilder) SetAPICDeployment(apicDeployment string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.APICDeployment = apicDeployment
	return b
}

func (b *transactionSummaryBuilder) SetEnvironmentName(environmentName string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.EnvironmentName = environmentName
	return b
}

func (b *transactionSummaryBuilder) SetEnvironmentID(environmentID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.EnvironmentID = environmentID
	return b
}

func (b *transactionSummaryBuilder) SetTenantID(tenantID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TenantID = tenantID
	return b
}

func (b *transactionSummaryBuilder) SetTrcbltPartitionID(trcbltPartitionID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TrcbltPartitionID = trcbltPartitionID
	return b
}

func (b *transactionSummaryBuilder) SetTargetPath(targetPath string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TargetPath = targetPath
	return b
}

func (b *transactionSummaryBuilder) SetResourcePath(resourcePath string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.ResourcePath = resourcePath
	return b
}

func (b *transactionSummaryBuilder) SetStatus(status TxSummaryStatus, statusDetail string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	err := status.IsValid()
	if err != nil {
		b.err = err
		return b
	}
	b.logEvent.TransactionSummary.Status = string(status)
	b.logEvent.TransactionSummary.StatusDetail = statusDetail
	return b
}

func (b *transactionSummaryBuilder) SetDuration(duration int) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionSummary.Duration = duration
	return b
}

func (b *transactionSummaryBuilder) SetApplication(appID, appName string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionSummary.Application = &Application{
		ID:   appID,
		Name: appName,
	}

	return b
}

// Currently, no one is setting Product for transaction summary builder, but leaving function signature as is for now
func (b *transactionSummaryBuilder) SetProduct(id, name, versionID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionSummary.Product = &models.Product{
		ID:        id,
		Name:      name,
		VersionID: versionID,
	}

	return b
}

func (b *transactionSummaryBuilder) SetTeam(teamID string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionSummary.Team = &Team{
		ID: teamID,
	}
	return b
}

func (b *transactionSummaryBuilder) SetProxy(proxyID, proxyName string, proxyRevision int) SummaryBuilder {
	return b.SetProxyWithStage(proxyID, proxyName, "", proxyRevision)
}

func (b *transactionSummaryBuilder) SetProxyWithStage(proxyID, proxyName, proxyStage string, proxyRevision int) SummaryBuilder {
	return b.SetProxyWithStageVersion(proxyID, proxyName, proxyStage, "", proxyRevision)
}

func (b *transactionSummaryBuilder) SetProxyWithStageVersion(proxyID, proxyName, proxyStage, proxyVersion string, proxyRevision int) SummaryBuilder {
	if b.err != nil {
		return b
	}

	// Strip the SummaryEventProxyIDPrefix if present
	proxyID = strings.TrimPrefix(proxyID, SummaryEventProxyIDPrefix)

	if proxyID == "" && proxyName != "" {
		proxyID = proxyName
	} else if proxyID == "" && proxyName == "" {
		proxyID = UnknownAPIID
	}

	b.logEvent.TransactionSummary.Proxy = &Proxy{
		ID:       proxyID,
		Revision: proxyRevision,
		Name:     proxyName,
		Stage:    proxyStage,
		Version:  proxyVersion,
	}
	return b
}

func (b *transactionSummaryBuilder) SetRunTime(runtimeID, runtimeName string) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.logEvent.TransactionSummary.Runtime = &Runtime{
		ID:   runtimeID,
		Name: runtimeName,
	}
	return b
}

func (b *transactionSummaryBuilder) SetEntryPoint(entryPointType, method, path, host string) SummaryBuilder {
	if b.err != nil {
		return b
	}

	b.logEvent.TransactionSummary.EntryPoint = &EntryPoint{
		Type:   entryPointType,
		Method: method,
		Path:   path,
		Host:   host,
	}
	return b
}

func (b *transactionSummaryBuilder) SetIsInMetricEvent(isInMetricEvent bool) SummaryBuilder {
	if b.err != nil {
		return b
	}

	b.logEvent.TransactionSummary.IsInMetricEvent = isInMetricEvent

	return b
}

func (b *transactionSummaryBuilder) SetRedactionConfig(config redaction.Redactions) SummaryBuilder {
	if b.err != nil {
		return b
	}
	b.redactionConfig = config
	return b
}

func (b *transactionSummaryBuilder) Build() (*LogEvent, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Set Target Path
	if b.redactionConfig == nil {
		b.logEvent.TargetPath, b.err = redaction.URIRedaction(b.logEvent.TargetPath)
	} else {
		b.logEvent.TargetPath, b.err = b.redactionConfig.URIRedaction(b.logEvent.TargetPath)
	}
	if b.err != nil {
		return nil, b.err
	}

	//Set Resource Path
	if b.redactionConfig == nil {
		b.logEvent.ResourcePath, b.err = redaction.URIRedaction(b.logEvent.ResourcePath)
	} else {
		b.logEvent.ResourcePath, b.err = b.redactionConfig.URIRedaction(b.logEvent.ResourcePath)
	}
	if b.err != nil {
		return nil, b.err
	}

	// Set redacted path in EntryPoint
	if b.logEvent.TransactionSummary.EntryPoint == nil {
		return nil, errors.New("transaction entry point details are not set in transaction summary event")
	}
	if b.redactionConfig == nil {
		b.logEvent.TransactionSummary.EntryPoint.Path, b.err = redaction.URIRedaction(b.logEvent.TransactionSummary.EntryPoint.Path)
	} else {
		b.logEvent.TransactionSummary.EntryPoint.Path, b.err = b.redactionConfig.URIRedaction(b.logEvent.TransactionSummary.EntryPoint.Path)
	}
	if b.err != nil {
		return nil, b.err
	}

	if b.logEvent.TrcbltPartitionID == "" {
		b.logEvent.TrcbltPartitionID = b.logEvent.TenantID
	}

	if b.logEvent.TransactionSummary.Proxy == nil {
		b.logEvent.TransactionSummary.Proxy = &Proxy{
			ID: UnknownAPIID,
		}
	}

	if b.logEvent.TransactionSummary.Proxy.Revision == 0 {
		b.logEvent.TransactionSummary.Proxy.Revision = 1
	}

	err := b.validateLogEvent()
	if err != nil {
		return nil, err
	}
	return b.logEvent, b.err
}

func (b *transactionSummaryBuilder) validateLogEvent() error {
	if agent.GetCentralConfig() == nil {
		return nil
	}

	if util.IsNotTest() && agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// Do not need this information in offline mode
		return nil
	}

	if b.logEvent.TenantID == "" {
		return errors.New("tenant ID property not set in transaction summary event")
	}

	if b.logEvent.EnvironmentID == "" {
		return errors.New("environment ID property are not set in transaction summary event")
	}

	if b.logEvent.TransactionSummary.Status == "" {
		return errors.New("status property not set in transaction summary event")
	}

	if b.logEvent.TransactionSummary.Product != nil && b.logEvent.TransactionSummary.Product.ID == "" {
		return errors.New("product ID property not set in transaction summary event")
	}
	return nil
}
