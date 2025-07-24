package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Server contains an http server for health checks.
type server struct {
	logger     log.FieldLogger
	router     *http.ServeMux
	hc         *Manager
	registered bool
}

// 1
func newStartNewServer(hc *Manager) *server {
	return &server{
		logger: log.NewFieldLogger().
			WithPackage("sdk.util.healthcheck").
			WithComponent("server"),
		router: http.NewServeMux(),
		hc:     hc,
	}
}

func (s *server) registerHandler(path string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(path, handler)
}

// HandleRequests - starts the http server
func (s *server) handleRequests() {
	if !s.registered {
		s.registerHandler("/status", s.statusHandler)
		for _, statusChecks := range s.hc.Checks {
			s.registerHandler(fmt.Sprintf("/status/%s", statusChecks.Endpoint), s.checkHandler)
		}
		s.registered = true
	}

	if s.hc.pprof {
		s.router.HandleFunc("/debug/pprof/", pprof.Index)
		s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	s.startHealthCheckServer()
}

func (s *server) startHealthCheckServer() {
	go func() {
		addr := fmt.Sprintf(":%d", s.hc.port)
		s.logger.WithField("address", addr).Info("starting health check server")
		err := http.ListenAndServe(addr, s.router)
		s.logger.WithError(err).Error("health check server stopped")
	}()
}

func (s *server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	s.logger.Trace("checking health status")

	// Return the data
	data, err := json.Marshal(s.hc)
	if err != nil {
		s.logger.WithError(err).Error("could not marshal the health check data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// If any of the checks failed change the return code to 500
	if s.hc.HCStatus == FAIL {
		s.logger.Error("health check failed, returning 503")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Write(data)
}

func (s *server) checkHandler(w http.ResponseWriter, r *http.Request) {
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
	thisCheck, ok := s.hc.Checks[endpoint]
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
	data, err := json.Marshal(s.hc.Checks[endpoint].Status)
	if err != nil {
		logger.WithError(err).Error("could not marshal the health check data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
