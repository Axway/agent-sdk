package healthcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/google/uuid"
)

var globalHealthChecker *healthChecker
var statusConfig corecfg.StatusConfig

func init() {
	globalHealthChecker = &healthChecker{
		Checks: make(map[string]*statusCheck, 0),
		Status: FAIL,
	}
}

// SetNameAndVersion - sets the name and version of the globalHealthChecker
func SetNameAndVersion(name, version string) {
	globalHealthChecker.Name = name
	globalHealthChecker.Version = version
}

// RegisterHealthcheck - register a new dependency with this service
func RegisterHealthcheck(name, endpoint string, check CheckStatus) (string, error) {
	if _, ok := globalHealthChecker.Checks[endpoint]; ok {
		return "", fmt.Errorf("A check with the endpoint of %s already exists", endpoint)
	}

	newID, _ := uuid.NewUUID()
	newChecker := &statusCheck{
		Name:     name,
		ID:       newID.String(),
		Endpoint: endpoint,
		Status:   &Status{},
		checker:  check,
	}

	globalHealthChecker.Checks[endpoint] = newChecker

	http.HandleFunc(fmt.Sprintf("/status/%s", endpoint), checkHandler)

	return newID.String(), nil
}

// SetStatusConfig - Set the status config globally
func SetStatusConfig(statusCfg corecfg.StatusConfig) {
	statusConfig = statusCfg
}

// GetStatusConfig - Set the status config globally
func GetStatusConfig() corecfg.StatusConfig {
	return statusConfig
}

// WaitForReady - Iterate through all healthchecks, returns OK once ready or returns Fail if timeout is reached
func WaitForReady() error {
	timeout := time.After(statusConfig.GetHealthCheckPeriod())
	tick := time.Tick(500 * time.Millisecond)
	// Keep trying until we have timed out or got a good result
	for {
		select {
		// Got a timeout! Fail with a timeout error
		case <-timeout:
			return errors.ErrTimeoutServicesNotReady
		// Got a tick, we should RunChecks
		case <-tick:
			if RunChecks() == OK {
				log.Info("Services are Ready")
				return nil
			}
		}
	}
}

//RunChecks - loop through all
func RunChecks() StatusLevel {
	globalHealthChecker.Status = OK
	for _, check := range globalHealthChecker.Checks {
		executeCheck(check)
		if check.Status.Result == FAIL {
			globalHealthChecker.Status = FAIL
		}
	}
	return globalHealthChecker.Status
}

// executeCheck  - executes the specified status check and logs the result
func executeCheck(check *statusCheck) {
	// Run the check
	check.Status = check.checker(check.Name)
	if check.Status.Result == OK {
		log.Debugf("%s - %s", check.Name, check.Status.Result)
	} else {
		log.Errorf("%s - %s (%s)", check.Name, check.Status.Result, check.Status.Details)
	}
}

//HandleRequests - starts the http server
func HandleRequests() {
	if !globalHealthChecker.registered {
		http.HandleFunc("/status", statusHandler)
		http.HandleFunc("/status/", statusHandler)
		globalHealthChecker.registered = true
	}

	if statusConfig.GetPort() > 0 {
		go http.ListenAndServe(fmt.Sprintf(":%d", statusConfig.GetPort()), nil)
	}
}

//GetHealthcheckOutput - query the http endpoint and return the body
func GetHealthcheckOutput(url string) (string, error) {
	client := http.DefaultClient

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("Could not query for the status")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Could not read the body of the response")
	}

	// Marshall the body to the interface sent in
	var statusResp healthChecker
	err = json.Unmarshal(body, &statusResp)
	if err != nil {
		return "", fmt.Errorf("Could not marshall into the expected type")
	}
	// Close the response body and the server
	resp.Body.Close()

	output, err := json.MarshalIndent(statusResp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("Error formatting the Status Check into Indented JSON")
	}

	return string(output), nil
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// Run the checks to get the latest results
	RunChecks()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Return the data
	data, err := json.Marshal(globalHealthChecker)
	if err != nil {
		log.Errorf("Error hit marshalling the health check data to json: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// If any of the checks failed change the return code to 500
		if globalHealthChecker.Status == FAIL {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}

	io.WriteString(w, string(data))
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	// Run the checks to get the latest results
	path := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	log.Infof("%v", path)
	if len(path) != 2 || path[0] != "status" {
		log.Errorf("Error getting status for path %s, expected /status/[endpoint]", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get the check object
	endpoint := path[1]
	thisCheck, ok := globalHealthChecker.Checks[endpoint]
	if !ok {
		log.Errorf("Check with endpoint of %s is not known", endpoint)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	executeCheck(thisCheck)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// If check failed change return code to 500
	if thisCheck.Status.Result == FAIL {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Return data
	data, _ := json.Marshal(globalHealthChecker.Checks[endpoint].Status)
	io.WriteString(w, string(data))
}

//QueryForStatus - create a URL string and call teh GetHealthcheckOutput func
func QueryForStatus(port int) (statusOut string) {
	var err error
	urlObj := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
		Path:   "status",
	}
	statusOut, err = GetHealthcheckOutput(urlObj.String())
	if err != nil {
		statusOut = fmt.Sprintf("Error querying for the status: %v", err.Error())
	}
	return
}
