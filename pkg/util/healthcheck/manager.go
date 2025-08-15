package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/google/uuid"
)

// StatusLevel - the level of the status of the healthcheck
type StatusLevel string

const (
	// OK - healthcheck is running properly
	OK StatusLevel = "OK"
	// FAIL - healthcheck is failing
	FAIL StatusLevel = "FAIL"
)

type Manager struct {
	logger          log.FieldLogger
	Name            string                  `json:"name"`
	Version         string                  `json:"version,omitempty"`
	HCStatus        StatusLevel             `json:"status"`
	HCStatusDetail  string                  `json:"statusDetail,omitempty"`
	Checks          map[string]*statusCheck `json:"statusChecks"`
	statusMutex     *sync.RWMutex
	checksMutex     *sync.Mutex
	statusServer    *server
	port            int
	period          time.Duration
	interval        time.Duration
	initialInterval time.Duration
	jobID           string
	unittest        bool
	pprof           bool
}

type Option func(*Manager)

func IsUnitTest() Option {
	return func(o *Manager) {
		o.unittest = true
	}
}

func WithPort(port int) Option {
	return func(o *Manager) {
		o.port = port
	}
}

func WithPeriod(period time.Duration) Option {
	return func(o *Manager) {
		o.period = period
	}
}

func WithInitialInterval(initialInterval time.Duration) Option {
	return func(o *Manager) {
		o.initialInterval = initialInterval
	}
}

func WithInterval(interval time.Duration) Option {
	return func(o *Manager) {
		o.interval = interval
	}
}

func WithPprof() Option {
	return func(o *Manager) {
		o.pprof = true
	}
}

func WithName(name string) Option {
	return func(o *Manager) {
		o.Name = name
	}
}

func WithVersion(version string) Option {
	return func(o *Manager) {
		o.Version = version
	}
}

func NewManager(opts ...Option) *Manager {
	healthCheckManager := &Manager{
		logger: log.NewFieldLogger().
			WithComponent("manager").
			WithPackage("sdk.util.healthcheck"),
		Name:            "agent",
		Checks:          map[string]*statusCheck{},
		HCStatus:        OK,
		port:            8989,
		period:          5 * time.Minute,
		interval:        30 * time.Second,
		initialInterval: 5 * time.Second,
		statusMutex:     &sync.RWMutex{},
		checksMutex:     &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(healthCheckManager)
	}

	if !healthCheckManager.unittest {
		// register the periodic health check job
		healthCheckManager.jobID, _ = jobs.RegisterDetachedIntervalJobWithName(healthCheckManager, healthCheckManager.interval, "Periodic Health Check")
		// start the health check server
		healthCheckManager.statusServer = newStartNewServer(healthCheckManager)
	}

	return healthCheckManager
}

func (m *Manager) getCheck(endpoint string) *statusCheck {
	m.checksMutex.Lock()
	defer m.checksMutex.Unlock()
	if check, ok := m.Checks[endpoint]; ok {
		return check
	}
	return nil
}

func (m *Manager) getChecks() map[string]*statusCheck {
	m.checksMutex.Lock()
	defer m.checksMutex.Unlock()
	return m.Checks
}

func (m *Manager) addCheck(name string, check *statusCheck) {
	m.checksMutex.Lock()
	defer m.checksMutex.Unlock()
	m.Checks[name] = check
}

func (m *Manager) getStatusAndDetail() (StatusLevel, string) {
	m.statusMutex.RLock()
	defer m.statusMutex.RUnlock()
	return m.HCStatus, m.HCStatusDetail
}

func (m *Manager) setStatusAndDetail(status StatusLevel, detail string) {
	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()
	m.HCStatus = status
	m.HCStatusDetail = detail
}

func (m *Manager) InitialHealthCheck() error {
	if m.unittest {
		// m.logger.Trace("skipping health check ticker in test mode")
		m.logger.Info("skipping health check ticker in test mode")
		return nil
	}
	// m.logger.Trace("run health checker ticker to check health status on RunChecks")
	m.logger.Info("run health checker ticker to check health status on RunChecks")
	ticker := time.NewTicker(m.initialInterval)
	tickerTimeout := time.NewTicker(m.period)

	defer ticker.Stop()
	defer tickerTimeout.Stop()

	for {
		select {
		case <-tickerTimeout.C:
			err := errors.New("start up health check failing")
			m.logger.WithError(err).Error("could not start agent")
			return (err)
		case <-ticker.C:
			m.runChecks()
			status, _ := m.getStatusAndDetail()
			if status == OK {
				// m.logger.Debug("start up health check on successful")
				m.logger.Info("start up health check on successful")
				return nil
			}
			// m.logger.Trace("start up health checks still failing, will retry")
			m.logger.Info("start up health checks still failing, will retry")
		}
	}
}

func (m *Manager) StartServer() {
	if m.unittest {
		return
	}
	m.statusServer.handleRequests()
}

// RegisterHealthcheck - register a new dependency with this service
func (m *Manager) RegisterHealthcheck(name, endpoint string, check CheckStatus) (string, error) {
	logger := m.logger.WithField("name", name).WithField("endpoint", endpoint)
	// logger.Debug("registering health check")
	logger.Info("registering health check")
	if m.getCheck(endpoint) != nil {
		logger.Error("a check with the endpoint already exists")
		return "", fmt.Errorf("a check with the endpoint of %s already exists", endpoint)
	}

	newID := uuid.NewString()

	logger = logger.WithField("id", newID)
	newChecker := &statusCheck{
		Name:     name,
		ID:       newID,
		Endpoint: endpoint,
		Status:   &Status{},
		logger:   logger.WithComponent("statusCheck"),
		checker:  check,
	}

	m.addCheck(endpoint, newChecker)
	if m.statusServer != nil {
		// logger.Debug("registering endpoint with health check server")
		logger.Info("registering endpoint with health check server")
		m.statusServer.registerHandler(fmt.Sprintf("/status/%s", endpoint), m.statusServer.checkHandler)
	}

	if util.IsNotTest() {
		newChecker.executeCheck()
	}

	logger.Info("health check registered")
	return newID, nil
}

func (m *Manager) runChecks() {
	m.logger.Trace("running health checks")
	status := Status{Result: OK}
	statusMutex := &sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(m.getChecks()))
	for _, check := range m.getChecks() {
		go func(c *statusCheck) {
			defer wg.Done()
			check.executeCheck()
			statusMutex.Lock()
			defer statusMutex.Unlock()
			if check.Status.Result == FAIL && status.Result == OK {
				status.Result = FAIL
				status.Details = check.Status.Details
			}
		}(check)
	}
	wg.Wait()
	m.setStatusAndDetail(status.Result, status.Details)
}

func (m *Manager) RunChecks() StatusLevel {
	m.runChecks()
	status, _ := m.getStatusAndDetail()
	return status
}

// GetCheckStatus - returns the current status for specified service
func (m *Manager) GetCheckStatus(endpoint string) StatusLevel {
	statusCheck := m.getCheck(endpoint)
	logger := m.logger.WithField("endpoint", endpoint)
	if statusCheck == nil {
		// logger.Debug("health check endpoint not found in global health checker")
		logger.Info("health check endpoint not found in global health checker")
		return FAIL
	}
	if statusCheck.Status.Result != OK {
		logger.
			WithField("check", statusCheck.Name).
			WithField("result", statusCheck.Status.Result).
			WithField("details", statusCheck.Status.Details).
			Error("health check is not OK")
	}
	return statusCheck.Status.Result
}

func (m *Manager) CheckIsRunning() error {
	if m.port > 0 {
		apiClient := api.NewClient(nil, "", api.WithTimeout(5*time.Second))
		req := api.Request{
			Method: "GET",
			URL:    "http://0.0.0.0:" + strconv.Itoa(m.port) + "/status",
		}
		res, err := apiClient.Send(req)
		if res == nil || err != nil {
			return nil
		}

		if res.Code == 200 {
			m.logger.WithField("port", m.port).Info("healthcheck port conflict detected")
			// m.logger.WithField("port", m.port).Trace("healthcheck port conflict detected")
			return fmt.Errorf("healthcheck port, %d, conflict detected", m.port)
		}
	}
	return nil
}

// GetAgentStatus - return the status of the agent health checker
func (m *Manager) GetAgentStatus() (string, string) {
	status, detail := m.getStatusAndDetail()
	return string(status), detail
}

func (m *Manager) Ready() bool {
	return true
}

func (m *Manager) Status() error {
	return nil
}

func (m *Manager) Execute() error {
	m.runChecks()
	status, _ := m.getStatusAndDetail()
	if status != OK {
		m.logger.WithField("status", status).Warn("periodicHealthCheck status is not OK")
	}
	return nil
}

// GetHealthcheckOutput - query the http endpoint and return the body
func GetHealthcheckOutput(url string) (string, error) {
	client := http.DefaultClient

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("could not query for the status")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read the body of the response")
	}

	// Marshall the body to the interface sent in
	var statusResp Manager
	err = json.Unmarshal(body, &statusResp)
	if err != nil {
		return "", fmt.Errorf("could not marshall into the expected type")
	}
	// Close the response body and the server
	resp.Body.Close()

	output, err := json.MarshalIndent(statusResp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error formatting the Status Check into Indented JSON")
	}

	if resp.StatusCode != http.StatusOK {
		return string(output), fmt.Errorf("healthcheck failed %s", resp.Status)
	}
	return string(output), nil
}
