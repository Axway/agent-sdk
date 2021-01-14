package transaction

import (
	"errors"
	"reflect"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
)

const defaultAPICDeployment = "prod"

// IsValid - Validator for TxEventStatus
func (s TxEventStatus) IsValid() error {
	switch s {
	case TxEventStatusPass, TxEventStatusFail:
		return nil
	}
	return errors.New("Invalid transaction event status")
}

// IsValid - Validator for TxSummaryStatus
func (s TxSummaryStatus) IsValid() error {
	switch s {
	case TxSummaryStatusSuccess, TxSummaryStatusFailure, TxSummaryStatusException, TxSummaryStatusUnknown:
		return nil
	}
	return errors.New("Invalid transaction summary status")
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

	Build() (*LogEvent, error)
}

type transactionEventBuilder struct {
	EventBuilder
	err                error
	cfgTenantID        string
	cfgAPICDeployment  string
	cfgEnvironmentName string
	cfgEnvironmentID   string
	logEvent           *LogEvent
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
	SetProduct(product string) SummaryBuilder
	SetTeam(teamID string) SummaryBuilder
	SetProxy(proxyID, proxyName string, proxyRevision int) SummaryBuilder
	SetRunTime(runtimeID, runtimeName string) SummaryBuilder
	SetEntryPoint(entryPointType, method, path, host string) SummaryBuilder

	Build() (*LogEvent, error)
}

type transactionSummaryBuilder struct {
	SummaryBuilder
	err                error
	cfgTenantID        string
	cfgAPICDeployment  string
	cfgEnvironmentName string
	cfgEnvironmentID   string
	logEvent           *LogEvent
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
	}
	return txSummaryBuilder
}

func (b *transactionEventBuilder) SetTimestamp(timestamp int64) EventBuilder {
	b.logEvent.Stamp = timestamp
	return b
}

func (b *transactionEventBuilder) SetTransactionID(transactionID string) EventBuilder {
	b.logEvent.TransactionID = transactionID
	return b
}

func (b *transactionEventBuilder) SetAPICDeployment(apicDeployment string) EventBuilder {
	b.logEvent.APICDeployment = apicDeployment
	return b
}

func (b *transactionEventBuilder) SetEnvironmentName(environmentName string) EventBuilder {
	b.logEvent.EnvironmentName = environmentName
	return b
}

func (b *transactionEventBuilder) SetEnvironmentID(environmentID string) EventBuilder {
	b.logEvent.EnvironmentID = environmentID
	return b
}

func (b *transactionEventBuilder) SetTenantID(tenantID string) EventBuilder {
	b.logEvent.TenantID = tenantID
	return b
}

func (b *transactionEventBuilder) SetTrcbltPartitionID(trcbltPartitionID string) EventBuilder {
	b.logEvent.TrcbltPartitionID = trcbltPartitionID
	return b
}

func (b *transactionEventBuilder) SetTargetPath(targetPath string) EventBuilder {
	b.logEvent.TargetPath = targetPath
	return b
}

func (b *transactionEventBuilder) SetResourcePath(resourcePath string) EventBuilder {
	b.logEvent.ResourcePath = resourcePath
	return b
}

func (b *transactionEventBuilder) SetID(id string) EventBuilder {
	b.logEvent.TransactionEvent.ID = id
	return b
}

func (b *transactionEventBuilder) SetParentID(parentID string) EventBuilder {
	b.logEvent.TransactionEvent.ParentID = parentID
	return b
}

func (b *transactionEventBuilder) SetSource(source string) EventBuilder {
	b.logEvent.TransactionEvent.Source = source
	return b
}

func (b *transactionEventBuilder) SetDestination(destination string) EventBuilder {
	b.logEvent.TransactionEvent.Destination = destination
	return b
}

func (b *transactionEventBuilder) SetDuration(duration int) EventBuilder {
	b.logEvent.TransactionEvent.Duration = duration
	return b
}

func (b *transactionEventBuilder) SetDirection(direction string) EventBuilder {
	b.logEvent.TransactionEvent.Direction = direction
	return b
}

func (b *transactionEventBuilder) SetStatus(status TxEventStatus) EventBuilder {
	err := status.IsValid()
	if err != nil {
		b.err = err
		return b
	}
	b.logEvent.TransactionEvent.Status = string(status)
	return b
}

func (b *transactionEventBuilder) SetProtocolDetail(protocolDetail interface{}) EventBuilder {
	_, ok := protocolDetail.(*Protocol)
	if !ok {
		_, ok = protocolDetail.(*JMSProtocol)
	}
	if ok {
		b.logEvent.TransactionEvent.Protocol = protocolDetail
	} else {
		b.err = errors.New("Unsupported protocol type")
	}
	return b
}

func (b *transactionEventBuilder) Build() (*LogEvent, error) {
	if b.err != nil {
		return nil, b.err
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
	if b.logEvent.TenantID == "" {
		return errors.New("Tenant ID property not set in transaction event")
	}

	if b.logEvent.EnvironmentID == "" {
		return errors.New("Environment ID property are not set in transaction event")
	}

	if b.logEvent.TransactionEvent.ID == "" {
		return errors.New("ID property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Direction == "" {
		return errors.New("Direction property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Status == "" {
		return errors.New("Status property not set in transaction event")
	}

	if b.logEvent.TransactionEvent.Protocol == nil {
		return errors.New("Protocol details not set in transaction event")
	}
	return nil
}

func (b *transactionSummaryBuilder) SetTimestamp(timestamp int64) SummaryBuilder {
	b.logEvent.Stamp = timestamp
	return b
}

func (b *transactionSummaryBuilder) SetTransactionID(transactionID string) SummaryBuilder {
	b.logEvent.TransactionID = transactionID
	return b
}

func (b *transactionSummaryBuilder) SetAPICDeployment(apicDeployment string) SummaryBuilder {
	b.logEvent.APICDeployment = apicDeployment
	return b
}

func (b *transactionSummaryBuilder) SetEnvironmentName(environmentName string) SummaryBuilder {
	b.logEvent.EnvironmentName = environmentName
	return b
}

func (b *transactionSummaryBuilder) SetEnvironmentID(environmentID string) SummaryBuilder {
	b.logEvent.EnvironmentID = environmentID
	return b
}

func (b *transactionSummaryBuilder) SetTenantID(tenantID string) SummaryBuilder {
	b.logEvent.TenantID = tenantID
	return b
}

func (b *transactionSummaryBuilder) SetTrcbltPartitionID(trcbltPartitionID string) SummaryBuilder {
	b.logEvent.TrcbltPartitionID = trcbltPartitionID
	return b
}

func (b *transactionSummaryBuilder) SetTargetPath(targetPath string) SummaryBuilder {
	b.logEvent.TargetPath = targetPath
	return b
}

func (b *transactionSummaryBuilder) SetResourcePath(resourcePath string) SummaryBuilder {
	b.logEvent.ResourcePath = resourcePath
	return b
}

func (b *transactionSummaryBuilder) SetStatus(status TxSummaryStatus, statusDetail string) SummaryBuilder {
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
	b.logEvent.TransactionSummary.Duration = duration
	return b
}

func (b *transactionSummaryBuilder) SetApplication(appID, appName string) SummaryBuilder {
	b.logEvent.TransactionSummary.Application = &Application{
		ID:   appID,
		Name: appName,
	}
	return b
}

func (b *transactionSummaryBuilder) SetProduct(product string) SummaryBuilder {
	b.logEvent.TransactionSummary.Product = product
	return b
}

func (b *transactionSummaryBuilder) SetTeam(teamID string) SummaryBuilder {
	b.logEvent.TransactionSummary.Team = &Team{
		ID: teamID,
	}
	return b
}

func (b *transactionSummaryBuilder) SetProxy(proxyID, proxyName string, proxyRevision int) SummaryBuilder {
	if proxyID == "" {
		proxyID = "unknown"
	}
	b.logEvent.TransactionSummary.Proxy = &Proxy{
		ID:       proxyID,
		Revision: proxyRevision,
		Name:     proxyName,
	}
	return b
}

func (b *transactionSummaryBuilder) SetRunTime(runtimeID, runtimeName string) SummaryBuilder {
	b.logEvent.TransactionSummary.Runtime = &Runtime{
		ID:   runtimeID,
		Name: runtimeName,
	}
	return b
}

func (b *transactionSummaryBuilder) SetEntryPoint(entryPointType, method, path, host string) SummaryBuilder {
	b.logEvent.TransactionSummary.EntryPoint = &EntryPoint{
		Type:   entryPointType,
		Method: method,
		Path:   path,
		Host:   host,
	}
	return b
}

func (b *transactionSummaryBuilder) Build() (*LogEvent, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.logEvent.TrcbltPartitionID == "" {
		b.logEvent.TrcbltPartitionID = b.logEvent.TenantID
	}

	if b.logEvent.TransactionSummary.Proxy == nil {
		b.logEvent.TransactionSummary.Proxy = &Proxy{
			ID: "unknown",
		}
	}

	err := b.validateLogEvent()
	if err != nil {
		return nil, err
	}
	return b.logEvent, b.err
}

func (b *transactionSummaryBuilder) validateLogEvent() error {
	if b.logEvent.TenantID == "" {
		return errors.New("Tenant ID property not set in transaction summary event")
	}

	if b.logEvent.EnvironmentID == "" {
		return errors.New("Environment ID property are not set in transaction summary event")
	}

	if b.logEvent.TransactionSummary.Status == "" {
		return errors.New("Status property not set in transaction summary event")
	}

	if b.logEvent.TransactionSummary.EntryPoint == nil {
		return errors.New("Transaction entry point details are not set in transaction summary event")
	}

	return nil
}
