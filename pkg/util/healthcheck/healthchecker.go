package healthcheck

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
)

var globalHealthChecker *healthChecker
var statusConfig corecfg.StatusConfig
var logger log.FieldLogger
var statusCfgMutex = sync.Mutex{}

func init() {
	globalHealthChecker = &healthChecker{
		Checks: make(map[string]*statusCheck, 0),
		Status: FAIL,
	}
	logger = log.NewFieldLogger().
		WithPackage("sdk.util.healthcheck").
		WithComponent("healthChecker")
}

// StartPeriodicHealthCheck - starts a job that runs the periodic health checks
func StartPeriodicHealthCheck() {
	interval := defaultCheckInterval
	if GetStatusConfig() != nil {
		interval = GetStatusConfig().GetHealthCheckInterval()
	}
	periodicHealthCheckJob := &periodicHealthCheck{interval: interval}
	jobs.RegisterDetachedIntervalJobWithName(periodicHealthCheckJob, periodicHealthCheckJob.interval, "Periodic Health Check")
}

// SetNameAndVersion - sets the name and version of the globalHealthChecker
func SetNameAndVersion(name, version string) {
	globalHealthChecker.Name = name
	globalHealthChecker.Version = version
}

// RegisterHealthcheck - register a new dependency with this service
func RegisterHealthcheck(name, endpoint string, check CheckStatus) (string, error) {
	if _, ok := globalHealthChecker.Checks[endpoint]; ok {
		return "", fmt.Errorf("a check with the endpoint of %s already exists", endpoint)
	}

	newID, _ := uuid.NewUUID()
	newChecker := &statusCheck{
		Name:        name,
		ID:          newID.String(),
		Endpoint:    endpoint,
		Status:      &Status{},
		checker:     check,
		statusMutex: &sync.Mutex{},
	}

	globalHealthChecker.Checks[endpoint] = newChecker
	statusServer := globalHealthChecker.statusServer
	if statusServer != nil {
		statusServer.registerHandler(fmt.Sprintf("/status/%s", endpoint), statusServer.checkHandler)
	}

	if util.IsNotTest() {
		newChecker.executeCheck()
	}

	return newID.String(), nil
}

// SetStatusConfig - Set the status config globally.
func SetStatusConfig(statusCfg corecfg.StatusConfig) {
	statusCfgMutex.Lock()
	defer statusCfgMutex.Unlock()
	statusConfig = statusCfg
}

// GetStatusConfig - Set the status config globally
func GetStatusConfig() corecfg.StatusConfig {
	statusCfgMutex.Lock()
	defer statusCfgMutex.Unlock()
	return statusConfig
}

// GetStatus - returns the current status for specified service
func GetStatus(endpoint string) StatusLevel {
	statusCheck, ok := globalHealthChecker.Checks[endpoint]
	if !ok {
		logger.Debugf("health check endpoint for %s not found in global health checker. Returning %s status", endpoint, FAIL)
		return FAIL
	}
	if statusCheck.Status.Result != OK {
		logger.
			WithField("details", statusCheck.Status.Details).
			WithField("result", statusCheck.Status.Result).
			WithField("endpoint", endpoint).
			Error("health check is not OK")
	}
	return statusCheck.Status.Result
}

// RunChecks - loop through all
func RunChecks() StatusLevel {
	status := Status{Result: OK}
	for _, check := range globalHealthChecker.Checks {
		check.executeCheck()
		if check.Status.Result == FAIL && status.Result == OK {
			status = *check.Status
		}
	}

	globalHealthChecker.Status = status.Result
	globalHealthChecker.StatusDetail = status.Details
	return globalHealthChecker.Status
}

func (check *statusCheck) setStatus(s *Status) {
	check.Status = s
}

func (check *statusCheck) executeCheck() {
	s := check.checker(check.Name)
	check.setStatus(s)

	if check.Status.Result == OK {
		logger.
			WithField("check-name", check.Name).
			WithField("result", check.Status.Result).
			Trace("health check is OK")
	} else {
		logger.
			WithField("check-name", check.Name).
			WithField("result", check.Status.Result).
			WithField("details", check.Status.Details).
			Error("health check failed")
	}
}

// Server contains an http server for health checks.
type Server struct {
	logger   log.FieldLogger
	router   *http.ServeMux
	httpprof bool
}

func StartNewServer(httpprof bool) {
	globalHealthChecker.statusServer = &Server{
		logger: log.NewFieldLogger().
			WithPackage("sdk.util.healthcheck").
			WithComponent("healthChecker"),
		router:   http.NewServeMux(),
		httpprof: httpprof,
	}
	globalHealthChecker.statusServer.handleRequests()
}

func (s *Server) registerHandler(path string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(path, handler)
}

// HandleRequests - starts the http server
func (s *Server) handleRequests() {
	if !globalHealthChecker.registered {
		s.registerHandler("/status", s.statusHandler)
		for _, statusChecks := range globalHealthChecker.Checks {
			s.registerHandler(fmt.Sprintf("/status/%s", statusChecks.Endpoint), s.checkHandler)
		}
		globalHealthChecker.registered = true
	}

	if s.httpprof {
		s.router.HandleFunc("/debug/pprof/", pprof.Index)
		s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	s.startHealthCheckServer()
}

func (s *Server) startHealthCheckServer() {
	if statusConfig == nil {
		s.logger.Error("status config is not set, cannot start health check server")
		return
	}
	if statusConfig.GetPort() <= 0 {
		s.logger.Error("status port is not set or invalid, cannot start health check server")
		return
	}

	go func() {
		addr := fmt.Sprintf(":%d", statusConfig.GetPort())
		s.logger.WithField("address", addr).Info("starting health check server")
		err := http.ListenAndServe(addr, s.router)
		s.logger.WithError(err).Error("health check server stopped")
	}()
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	s.logger.Trace("checking health status")

	// Return the data
	data, err := json.Marshal(globalHealthChecker)
	if err != nil {
		s.logger.WithError(err).Error("could not marshal the health check data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// If any of the checks failed change the return code to 500
	if globalHealthChecker.Status == FAIL {
		s.logger.Error("health check failed, returning 503")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Write(data)
}

func (s *Server) checkHandler(w http.ResponseWriter, r *http.Request) {
	// Run the checks to get the latest results
	path := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(path) != 2 || path[0] != "status" {
		s.logger.WithField("path", r.URL.Path).Error("could not get status for path", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get the check object
	endpoint := path[1]
	logger := s.logger.WithField("endpoint", endpoint)
	logger.Trace("checking endpoint status")
	thisCheck, ok := globalHealthChecker.Checks[endpoint]
	if !ok {
		logger.Error("unknown endpoint")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// If check failed change return code to 500
	if thisCheck.Status.Result == FAIL {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Return data
	data, err := json.Marshal(globalHealthChecker.Checks[endpoint].Status)
	if err != nil {
		logger.WithError(err).Error("could not marshal the health check data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// CheckIsRunning - Checks if another instance is already running by looking at the healthcheck.
func CheckIsRunning() error {
	if statusConfig != nil && statusConfig.GetPort() > 0 {
		apiClient := api.NewClient(nil, "", api.WithTimeout(5*time.Second))
		req := api.Request{
			Method: "GET",
			URL:    "http://0.0.0.0:" + strconv.Itoa(statusConfig.GetPort()) + "/status",
		}
		res, err := apiClient.Send(req)
		if res == nil || err != nil {
			return nil
		}

		if err == nil && res.Code == 200 {
			return ErrAlreadyRunning
		}
	}
	return nil
}

// GetGlobalStatus - return the status of the global health checker
func GetGlobalStatus() (string, string) {
	return string(globalHealthChecker.Status), globalHealthChecker.StatusDetail
}

// GetHealthcheckOutput - query the http endpoint and return the body
func GetHealthcheckOutput(url string) (string, error) {
	client := http.DefaultClient

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("could not query for the status")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read the body of the response")
	}

	// Marshall the body to the interface sent in
	var statusResp healthChecker
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
