package healthcheck

import (
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckManager(t *testing.T) {
	// create new health check manager
	testManager := NewManager(
		WithName("test-manager"),
		WithInitialInterval(1000),
		WithInterval(1000),
		WithVersion("1.0.0"),
		WithPprof(),
		WithPeriod(100000),
		WithPort(8080),
	)

	jobs.UnregisterJob(testManager.jobID)
	assert.NotNil(t, testManager, "HealthCheckManager should not be nil")

	err := testManager.InitialHealthCheck()
	assert.NoError(t, err, "Initial health check should not return an error")

	// execute the initial health check again after setting unittest to false
	testManager.unittest = false
	err = testManager.InitialHealthCheck()
	assert.NoError(t, err, "Initial health check should not return an error after setting unittest to false")

	testManager.StartServer()

	t1Mutex := &sync.Mutex{}
	t2Mutex := &sync.Mutex{}
	t1Mutex.Lock()
	test1Status := OK
	t1Mutex.Unlock()

	// register a new health check
	testManager.RegisterHealthcheck("test-1", "test1",
		func(name string) *Status {
			t1Mutex.Lock()
			defer t1Mutex.Unlock()
			return &Status{Result: test1Status, Details: "test-1 passed"}
		},
	)
	// register a second health check
	testManager.RegisterHealthcheck("test-2", "test2",
		func(name string) *Status {
			t2Mutex.Lock()
			defer t2Mutex.Unlock()
			return &Status{Result: OK, Details: "test-2 passed"}
		},
	)

	// run the health checks
	status := testManager.RunChecks()
	assert.Equal(t, status, OK, "Health check status should be OK")

	// update the status of test-1 to FAIL
	t1Mutex.Lock()
	test1Status = FAIL
	t1Mutex.Unlock()
	status = testManager.RunChecks()
	assert.Equal(t, status, FAIL, "Health check status should be FAIL after test-1 fails")

	// get the status of test-1
	test1StatusResult := testManager.GetCheckStatus("test3")
	assert.Equal(t, test1StatusResult, FAIL, "Health check status should be FAIL")

	// get the status of test-2
	test2StatusResult := testManager.GetCheckStatus("test2")
	assert.Equal(t, test2StatusResult, OK, "Health check status should be OK")

	// expect FAIL status for an endpoint that is not registered
	unknownCheck := testManager.GetCheckStatus("unknown")
	assert.Equal(t, unknownCheck, FAIL, "Health check status should be FAIL for unknown checks")

	// test the check is running method
	err = testManager.CheckIsRunning()
	assert.NoError(t, err, "Health check should be running")

	// validate that get healthcheck output works
	output, err := GetHealthcheckOutput("http://0.0.0.0:8080/status")
	assert.Error(t, err, "Getting health check output should not return an error")
	assert.NotNil(t, output, "Health check output should not be nil")
	assert.Contains(t, string(output), "test-1", "Health check output should contain test-1")
	assert.Contains(t, string(output), "test-2", "Health check output should contain test-2")
	assert.Contains(t, string(output), "FAIL", "Health check output should contain FAIL")
	assert.Contains(t, string(output), "OK", "Health check output should contain OK")

	// update the status of test-1 to OK
	t1Mutex.Lock()
	test1Status = OK
	t1Mutex.Unlock()
	// run the checks again
	testManager.RunChecks()
	// ensure the GetHealthcheckOutput reflects the change
	output, err = GetHealthcheckOutput("http://0.0.0.0:8080/status")
	assert.NoError(t, err, "Getting health check output should not return an error")
	assert.NotNil(t, output, "Health check output should not be nil")
	assert.Contains(t, string(output), "test-1", "Health check output should contain test-1")
	assert.Contains(t, string(output), "test-2", "Health check output should contain test-2")
	assert.NotContains(t, string(output), "FAIL", "Health check output should not contain FAIL")
	assert.Contains(t, string(output), "OK", "Health check output should contain OK")

}
