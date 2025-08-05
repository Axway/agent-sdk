package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStartNewServer(t *testing.T) {
	hcManager := &Manager{
		logger:      log.NewFieldLogger().WithPackage("test"),
		Name:        "test-service",
		Version:     "1.0.0",
		HCStatus:    OK,
		Checks:      make(map[string]*statusCheck),
		statusMutex: &sync.RWMutex{},
		checksMutex: &sync.Mutex{},
		port:        8080,
		unittest:    true,
	}

	server := newStartNewServer(hcManager)

	assert.NotNil(t, server)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.router)
	assert.Equal(t, hcManager, server.hc)
	assert.False(t, server.registered)
}

func TestServerRegisterHandler(t *testing.T) {
	hcManager := &Manager{
		logger:      log.NewFieldLogger().WithPackage("test"),
		Name:        "test-service",
		Checks:      make(map[string]*statusCheck),
		statusMutex: &sync.RWMutex{},
		checksMutex: &sync.Mutex{},
		unittest:    true,
	}

	server := newStartNewServer(hcManager)

	// Test handler registration
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}

	server.registerHandler("/test", testHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test", w.Body.String())
}

func TestServerHandleRequests(t *testing.T) {
	hcManager := &Manager{
		logger:      log.NewFieldLogger().WithPackage("test"),
		Name:        "test-service",
		Version:     "1.0.0",
		HCStatus:    OK,
		Checks:      make(map[string]*statusCheck),
		statusMutex: &sync.RWMutex{},
		checksMutex: &sync.Mutex{},
		port:        8080,
		unittest:    true,
		pprof:       false,
	}

	// Add a test check
	testCheck := &statusCheck{
		ID:       "test-id",
		Name:     "test-check",
		Endpoint: "test-endpoint",
		Status: &Status{
			Result:  OK,
			Details: "Test check is healthy",
		},
		logger: log.NewFieldLogger().WithPackage("test"),
		checker: func(name string) *Status {
			return &Status{Result: OK, Details: "Test check is healthy"}
		},
	}
	hcManager.Checks["test-endpoint"] = testCheck

	server := newStartNewServer(hcManager)
	server.handleRequests()

	assert.True(t, server.registered)

	// Test that status handler is registered
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test that check handler is registered
	req = httptest.NewRequest("GET", "/status/test-endpoint", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test duplicate registration
	server.handleRequests()
	assert.True(t, server.registered)
}

func TestStatusHandler(t *testing.T) {
	tests := []struct {
		name           string
		hcStatus       StatusLevel
		expectedStatus int
	}{
		{
			name:           "healthy status",
			hcStatus:       OK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "failed status",
			hcStatus:       FAIL,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcManager := &Manager{
				logger:         log.NewFieldLogger().WithPackage("test"),
				Name:           "test-service",
				Version:        "1.0.0",
				HCStatus:       tt.hcStatus,
				HCStatusDetail: "Test status detail",
				Checks:         make(map[string]*statusCheck),
				statusMutex:    &sync.RWMutex{},
				checksMutex:    &sync.Mutex{},
				unittest:       true,
			}

			server := newStartNewServer(hcManager)

			req := httptest.NewRequest("GET", "/status", nil)
			w := httptest.NewRecorder()

			server.statusHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			// Verify response body
			var response Manager
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, hcManager.Name, response.Name)
			assert.Equal(t, hcManager.Version, response.Version)
			assert.Equal(t, hcManager.HCStatus, response.HCStatus)
			assert.Equal(t, hcManager.HCStatusDetail, response.HCStatusDetail)
		})
	}
}

func TestCheckHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		checkExists    bool
		checkStatus    StatusLevel
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid endpoint with OK status",
			path:           "/status/test-endpoint",
			checkExists:    true,
			checkStatus:    OK,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "valid endpoint with FAIL status",
			path:           "/status/test-endpoint",
			checkExists:    true,
			checkStatus:    FAIL,
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  false,
		},
		{
			name:           "unknown endpoint",
			path:           "/status/unknown-endpoint",
			checkExists:    false,
			checkStatus:    OK,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name:           "invalid path format",
			path:           "/invalid/path",
			checkExists:    false,
			checkStatus:    OK,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name:           "malformed path",
			path:           "/status",
			checkExists:    false,
			checkStatus:    OK,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hcManager := &Manager{
				logger:      log.NewFieldLogger().WithPackage("test"),
				Name:        "test-service",
				Checks:      make(map[string]*statusCheck),
				statusMutex: &sync.RWMutex{},
				checksMutex: &sync.Mutex{},
				unittest:    true,
			}

			if tt.checkExists {
				testCheck := &statusCheck{
					ID:       "test-id",
					Name:     "test-check",
					Endpoint: "test-endpoint",
					Status: &Status{
						Result:  tt.checkStatus,
						Details: "Test check details",
					},
					logger: log.NewFieldLogger().WithPackage("test"),
					checker: func(name string) *Status {
						return &Status{Result: tt.checkStatus, Details: "Test check details"}
					},
				}
				hcManager.Checks["test-endpoint"] = testCheck
			}

			server := newStartNewServer(hcManager)

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			server.checkHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectedError {
				assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

				// Verify response body for successful cases
				var response Status
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.checkStatus, response.Result)
				assert.Equal(t, "Test check details", response.Details)
			}
		})
	}
}

// Integration test for the complete server functionality
func TestServerIntegration(t *testing.T) {
	hcManager := &Manager{
		logger:         log.NewFieldLogger().WithPackage("test"),
		Name:           "integration-test-service",
		Version:        "1.0.0",
		HCStatus:       OK,
		HCStatusDetail: "Service is healthy",
		Checks:         make(map[string]*statusCheck),
		statusMutex:    &sync.RWMutex{},
		checksMutex:    &sync.Mutex{},
		port:           8080,
		unittest:       true,
		pprof:          true,
	}

	// Add multiple test checks
	checks := []struct {
		endpoint string
		status   StatusLevel
		details  string
	}{
		{"database", OK, "Database connection is healthy"},
		{"redis", FAIL, "Redis connection timeout"},
		{"external-api", OK, "External API is responding"},
	}

	for _, check := range checks {
		testCheck := &statusCheck{
			ID:       fmt.Sprintf("%s-id", check.endpoint),
			Name:     fmt.Sprintf("%s-check", check.endpoint),
			Endpoint: check.endpoint,
			Status: &Status{
				Result:  check.status,
				Details: check.details,
			},
			logger: log.NewFieldLogger().WithPackage("test"),
			checker: func(name string) *Status {
				return &Status{Result: check.status, Details: check.details}
			},
		}
		hcManager.Checks[check.endpoint] = testCheck
	}

	// Set overall status to FAIL if any check fails
	hcManager.HCStatus = FAIL

	server := newStartNewServer(hcManager)
	server.handleRequests()

	// Test overall status endpoint
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response Manager
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "integration-test-service", response.Name)
	assert.Equal(t, "1.0.0", response.Version)
	assert.Equal(t, FAIL, response.HCStatus)

	// Test individual check endpoints
	for _, check := range checks {
		req := httptest.NewRequest("GET", fmt.Sprintf("/status/%s", check.endpoint), nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		expectedStatus := http.StatusOK
		if check.status == FAIL {
			expectedStatus = http.StatusServiceUnavailable
		}

		assert.Equal(t, expectedStatus, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var statusResponse Status
		err := json.Unmarshal(w.Body.Bytes(), &statusResponse)
		require.NoError(t, err)
		assert.Equal(t, check.status, statusResponse.Result)
		assert.Equal(t, check.details, statusResponse.Details)
	}

	// Test 404 for unknown endpoint (no handler registered for this path)
	req = httptest.NewRequest("GET", "/status/unknown", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test pprof endpoints
	req = httptest.NewRequest("GET", "/debug/pprof/", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
