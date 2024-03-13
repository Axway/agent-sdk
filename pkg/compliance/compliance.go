package compliance

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type RuntimeResult struct {
	APIServiceInstance string `json:"apiServiceInstance"`
	CriticalCount      int64  `json:"criticalCount"`
	ErrorCount         int64  `json:"errorCount"`
	WarningCount       int64  `json:"warningCount"`
}

// APIRuntimeCompliance -
type APIRuntimeCompliance struct {
	Results   RuntimeResult `json:"results"`
	EventID   string        `json:"eventID"`
	StartTime time.Time     `json:"-"`
}

// GetStartTime - Returns the start time for subscription metric
func (a *APIRuntimeCompliance) GetStartTime() time.Time {
	return a.StartTime
}

// GetType - Returns APIMetric
func (a *APIRuntimeCompliance) GetType() string {
	return "APIRuntimeCompliance"
}

// GetType - Returns APIMetric
func (a *APIRuntimeCompliance) GetEventID() string {
	return a.EventID
}

type RuntimeResults interface {
	AddRuntimeResult(RuntimeResult)
}

type runtimeResults struct {
	logger  log.FieldLogger
	orgGUID string
	results map[string]RuntimeResult
}

func (r *runtimeResults) AddRuntimeResult(result RuntimeResult) {
	if r.results == nil {
		r.results = make(map[string]RuntimeResult)
	}
	r.results[result.APIServiceInstance] = result
}

func (r *runtimeResults) publish() {
	// r.publishEvents()
	r.publishResources()
}

func (r *runtimeResults) publishEvents() {
	cacheManager := agent.GetCacheManager()
	batch := NewEventBatch()
	for instanceName, result := range r.results {
		ri, err := cacheManager.GetAPIServiceInstanceByName(instanceName)
		if err != nil {
			r.logger.WithError(err).WithField("instanceName", instanceName).Error("skipping instance")
		}
		instance := &management.APIServiceInstance{}
		instance.FromInstance(ri)
		if instance.Source != nil {
			complianceResult := &APIRuntimeCompliance{
				Results:   result,
				EventID:   uuid.NewString(),
				StartTime: time.Now(),
			}
			AddEventToBatch(createV4Event(r.orgGUID, complianceResult), batch)
		}
	}
	batch.publish()
}

func (r *runtimeResults) publishResources() {
	cacheManager := agent.GetCacheManager()
	for instanceName, result := range r.results {
		ri, err := cacheManager.GetAPIServiceInstanceByName(instanceName)
		if err != nil {
			r.logger.WithError(err).WithField("instanceName", instanceName).Error("skipping instance")
			continue
		}

		instance := &management.APIServiceInstance{}
		instance.FromInstance(ri)
		if instance.Source != nil {
			compliance := management.ApiServiceInstanceSourceCompliance{
				Runtime: management.ApiServiceInstanceSourceRuntimeStatus{
					Result: management.ApiServiceInstanceSourceRuntimeStatusResult{
						Timestamp:     v1.Time(time.Now()),
						CriticalCount: int32(result.CriticalCount),
						ErrorCount:    int32(result.ErrorCount),
						WarningCount:  int32(result.WarningCount),
					},
				},
			}

			patches := make([]map[string]interface{}, 0)
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  "/source/compliance",
				"value": compliance,
			})
			instance, err := agent.GetCentralClient().PatchSubResource(instance, "source", patches)
			if err != nil {
				r.logger.WithError(err)
			}
			r.logger.Tracef("%+v", instance)
		}
	}
}

type Processor interface {
	ProcessRuntime(RuntimeResults) error
}

type Manager interface {
	RegisterRuntimeComplianceJob(interval time.Duration, processor Processor)
}

type complianceManager struct {
	jobID   string
	orgGUID string
	logger  log.FieldLogger
}

var manager Manager

func GetManager() Manager {
	if manager == nil {
		cm := &complianceManager{
			logger: log.NewFieldLogger().WithComponent("compliance"),
		}
		cm.setOrgGUID()
		manager = cm
	}
	return manager
}

func (m *complianceManager) setOrgGUID() {
	authToken, _ := agent.GetCentralAuthToken()
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true

	claims := jwt.MapClaims{}
	_, _, _ = parser.ParseUnverified(authToken, claims)

	claim, ok := claims["org_guid"]
	if ok {
		m.orgGUID = claim.(string)
	}
}

func (m *complianceManager) RegisterRuntimeComplianceJob(interval time.Duration, processor Processor) {
	if m.jobID == "" {
		job := &runtimeComplianceJob{
			logger:    m.logger,
			processor: processor,
			orgGUID:   m.orgGUID,
		}
		jobID, err := jobs.RegisterIntervalJobWithName(job, interval, "Runtime compliance")
		if err != nil {
			m.logger.WithError(err).Error("failed to register runtime compliance job")
		}
		m.jobID = jobID
	}
}

type runtimeComplianceJob struct {
	logger    log.FieldLogger
	processor Processor
	orgGUID   string
}

func (j *runtimeComplianceJob) Status() error {
	return nil
}

func (j *runtimeComplianceJob) Ready() bool {
	return true
}

func (j *runtimeComplianceJob) Execute() error {
	if j.processor != nil {
		results := &runtimeResults{
			logger:  j.logger,
			orgGUID: j.orgGUID,
		}

		j.processor.ProcessRuntime(results)
		results.publish()
	}
	return nil
}

// func Start() {
// 	client := agent.GetCentralClient()
// 	cfg := agent.GetCentralConfig()

// 	env, _ := client.GetEnvironment()
// 	envRuntimeSpec := env.Spec.Compliance.Runtime
// 	if envRuntimeSpec != "" {
// 		runtimeSpecURL := fmt.Sprintf("%s/%s/%s", cfg.GetURL(), "/apis/management/v1/apiruntimerulesets", envRuntimeSpec)
// 		ri, _ := client.GetResource(runtimeSpecURL)
// 		runtimeSpec := &managementv1.APIRuntimeRuleset{}
// 		runtimeSpec.FromInstance(ri)
// 		frequency := runtimeSpec.Spec.Definition.Frequency
// 		if frequency != "" {
// 			duration, _ := time.ParseDuration(frequency)
// 		}
// 	}
// }
