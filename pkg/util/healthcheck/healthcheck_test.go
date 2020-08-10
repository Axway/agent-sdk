package healthcheck

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func resetGlobalHealthChecker() {
	globalHealthChecker = &healthChecker{
		Checks: make(map[string]*statusCheck, 0),
		Status: FAIL,
	}
}

func TestRegisterHealthcheck(t *testing.T) {
	resetGlobalHealthChecker()

	// assert that the number of Checks is 0
	assert.Equal(t, 0, len(globalHealthChecker.Checks), "The initial number of checks was not 0")

	// Register a healthcheck
	newID, err := RegisterHealthcheck("dummy", "dummy", func(name string) *Status { return &Status{Result: OK} })

	assert.Nil(t, err, "There was an unexpected error while registering a check")
	assert.NotEqual(t, "", newID, "Expected a value for newID")
	assert.Equal(t, 1, len(globalHealthChecker.Checks), "Expected 1 check to have been registered")
	assert.Equal(t, "dummy", globalHealthChecker.Checks["dummy"].Endpoint, "Expected the dummy check endpoint to be dummy")
	assert.Equal(t, "dummy", globalHealthChecker.Checks["dummy"].Name, "Expected the dummy check endpoint to be dummy")

	// Register a duplicate healthcheck
	newID, err = RegisterHealthcheck("dummy", "dummy", func(name string) *Status { return &Status{Result: OK} })

	assert.NotNil(t, err, "There was no error thrown when expected")
	assert.Equal(t, "", newID, "Expected a blank for newID")
	assert.Equal(t, 1, len(globalHealthChecker.Checks), "Expected to still have only 1 check registered")
}

func TestRunChecks(t *testing.T) {
	resetGlobalHealthChecker()

	// assert that the number of Checks is 0
	assert.Equal(t, 0, len(globalHealthChecker.Checks), "The initial number of checks was not 0")

	hcValues := map[string]bool{
		"healthcheck1": false,
		"healthcheck2": false,
	}

	hcFunc := func(name string) *Status {
		if hcValues[name] {
			// return pass
			return &Status{
				Result: OK,
			}
		}
		// return fail
		return &Status{
			Details: fmt.Sprintf("%s set to false", name),
			Result:  FAIL,
		}
	}

	isReady := false
	cfg := corecfg.NewStatusConfig()
	SetStatusConfig(cfg)
	// Start a go func to watch WaitForReady
	go func() {
		// Set isReady to true on return
		err := WaitForReady()
		if err == nil {
			isReady = true
		}

	}()

	// Register healthchecks
	_, err1 := RegisterHealthcheck("healthcheck1", "healthcheck1", hcFunc)
	_, err2 := RegisterHealthcheck("healthcheck2", "healthcheck2", hcFunc)
	assert.Nil(t, err1, "There was an unexpected error while registering a healthcheck1")
	assert.Nil(t, err2, "There was an unexpected error while registering a healthcheck2")

	// Run the checks, al fail
	res := RunChecks()
	assert.Equal(t, FAIL, res, "The overall healthcheck should have failed")
	assert.False(t, isReady, "isReady should have been false")

	// only hc1 pass
	hcValues["healthcheck1"] = true
	hcValues["healthcheck2"] = false
	res = RunChecks()
	assert.Equal(t, FAIL, res, "The overall healthcheck should have failed")
	assert.Equal(t, OK, globalHealthChecker.Checks["healthcheck1"].Status.Result, "healthcheck1 should have passed")
	assert.Equal(t, FAIL, globalHealthChecker.Checks["healthcheck2"].Status.Result, "healthcheck2 should have failed")
	assert.False(t, isReady, "isReady should have been false")

	// only hc2 pass
	hcValues["healthcheck1"] = false
	hcValues["healthcheck2"] = true
	res = RunChecks()
	assert.Equal(t, FAIL, res, "The overall healthcheck should have failed")
	assert.Equal(t, FAIL, globalHealthChecker.Checks["healthcheck1"].Status.Result, "healthcheck1 should have failed")
	assert.Equal(t, OK, globalHealthChecker.Checks["healthcheck2"].Status.Result, "healthcheck2 should have passed")
	assert.False(t, isReady, "isReady should have been false")

	// hall hc pass
	hcValues["healthcheck1"] = true
	hcValues["healthcheck2"] = true
	res = RunChecks()
	assert.Equal(t, OK, res, "The overall healthcheck should have passed")
	assert.Equal(t, OK, globalHealthChecker.Checks["healthcheck1"].Status.Result, "healthcheck1 should have passed")
	assert.Equal(t, OK, globalHealthChecker.Checks["healthcheck2"].Status.Result, "healthcheck2 should have passed")
	// Give the WaitForReady check a second to pass
	time.Sleep(time.Second)
	assert.True(t, isReady, "isReady should have been true")
}

func TestHTTPRequests(t *testing.T) {
	resetGlobalHealthChecker()

	// assert that the number of Checks is 0
	assert.Equal(t, 0, len(globalHealthChecker.Checks), "The initial number of checks was not 0")

	hcValues := map[string]bool{
		"hc1": false,
		"hc2": false,
	}

	hcFunc := func(name string) *Status {
		if hcValues[name] {
			// return pass
			return &Status{
				Result: OK,
			}
		}
		// return fail
		return &Status{
			Details: fmt.Sprintf("%s set to false", name),
			Result:  FAIL,
		}
	}

	// Register healthchecks
	RegisterHealthcheck("hc1", "hc1", hcFunc)
	RegisterHealthcheck("hc2", "hc2", hcFunc)

	var server *httptest.Server

	// http Client
	client := http.DefaultClient
	getRequest := func(path string, marshObj interface{}, unmarshallErr bool) *http.Response {
		// Call the status endpoint
		resp, err := client.Get(server.URL + path)
		assert.Nil(t, err)

		// Read the return body
		body, err := ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		// Marshall the body to the interface sent in
		err = json.Unmarshal(body, &marshObj)
		if unmarshallErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		// Close the response body and the server
		resp.Body.Close()

		return resp
	}

	//########################## statusHandler ################################
	server = httptest.NewServer(http.HandlerFunc(statusHandler))

	hcValues["hc1"] = false
	hcValues["hc2"] = false

	// Marshall the body to the healthChecker struct
	var result healthChecker
	resp := getRequest("/status", &result, false)

	// assert response values
	assert.Equal(t, FAIL, result.Status, "Expected FAIL to be the overall result")
	assert.Equal(t, FAIL, result.Checks["hc1"].Status.Result, "hc1 should have failed")
	assert.Equal(t, FAIL, result.Checks["hc2"].Status.Result, "hc2 should have failed")
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// set all healthchecks to pass
	hcValues["hc1"] = true
	hcValues["hc2"] = true

	// Execute another request
	resp = getRequest("/status", &result, false)

	// assert response values
	assert.Equal(t, OK, result.Status, "Expected PASS to be the overall result")
	assert.Equal(t, OK, result.Checks["hc1"].Status.Result, "hc1 should have passed")
	assert.Equal(t, OK, result.Checks["hc2"].Status.Result, "hc2 should have passed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	//########################## GetHealthcheckOutput ################################

	// Marshall the previous response to the same marshall indented expected from GetHealthcheckOutput
	indentResult, _ := json.MarshalIndent(result, "", "  ")

	output, err := GetHealthcheckOutput(server.URL)
	assert.Nil(t, err)
	assert.Equal(t, string(indentResult), output)

	//########################## checkHandler ################################
	server = httptest.NewServer(http.HandlerFunc(checkHandler))
	var checkRes Status

	// Bad path
	resp = getRequest("/stats/hc1", &checkRes, true)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Bad check
	resp = getRequest("/status/badHC", &checkRes, true)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Set hc1 to fail
	hcValues["hc1"] = false
	resp = getRequest("/status/hc1", &checkRes, false)

	// assert response values
	assert.Equal(t, FAIL, checkRes.Result, "hc1 should have failed")
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Set hc1 to fail
	hcValues["hc1"] = true
	resp = getRequest("/status/hc1", &checkRes, false)

	// assert response values
	assert.Equal(t, OK, checkRes.Result, "hc1 should have passed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	server.Close()
}
