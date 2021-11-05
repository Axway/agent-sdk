package cmd

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"

	"regexp"
	"strconv"
	"strings"

	"net/http"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	avcCronSchedule = "@daily"
	jfrogURL        = "https://axway.jfrog.io/ui/api/v1/ui/treebrowser"
)

var agentNameToRepoPath = map[string]string{
	"AWSDiscoveryAgent":                      "aws-apigw-discovery-agent",
	"AWSTraceabilityAgent":                   "aws-apigw-traceability-agent",
	"AzureDiscoveryAgent":                    "azure-discovery-agent",
	"AzureTraceabilityAgent":                 "azure-traceability-agent",
	"EnterpriseEdgeGatewayDiscoveryAgent":    "v7-discovery-agent",
	"EnterpriseEdgeGatewayTraceabilityAgent": "v7-traceability-agent",
}

type version struct {
	major, minor, patch int
	val                 string
}

type jfrogRequest struct {
	Type     string `json:"type"`
	RepoType string `json:"repoType"`
	RepoKey  string `json:"repoKey"`
	Path     string `json:"path"`
	Text     string `json:"text"`
}

type jfrogData struct {
	Items []jfrogItem `json:"data"`
}

type jfrogItem struct {
	RepoKey      string `json:"repoKey,omitempty"`
	Path         string `json:"path,omitempty"`
	Version      string `json:"text"`
	RepoType     string `json:"repoType,omitempty"`
	HasChild     bool   `json:"hasChild,omitempty"`
	Local        bool   `json:"local,omitempty"`
	Type         string `json:"type,omitempty"`
	Compacted    bool   `json:"compacted,omitempty"`
	Cached       bool   `json:"cached,omitempty"`
	Trash        bool   `json:"trash,omitempty"`
	Distribution bool   `json:"distribution,omitempty"`
}

// AgentVersionCheckJob - polls for agent versions
type AgentVersionCheckJob struct {
	jobs.Job
	apiClient    coreapi.Client
	requestBytes []byte
	buildVersion string
	headers      map[string]string
}

// NewAgentVersionCheckJob - creates a new agent version check job structure
func NewAgentVersionCheckJob(cfg config.CentralConfig) (*AgentVersionCheckJob, error) {
	// get current build version
	buildVersion, err := getBuildVersion()
	if err != nil {
		log.Trace(err)
		return nil, err
	}

	if _, found := agentNameToRepoPath[BuildAgentName]; !found {
		err := errors.ErrStartingVersionChecker.FormatError("empty or generic data plane type name")
		log.Trace(err)
		return nil, err
	}

	// create the request body for each check
	requestBody := jfrogRequest{
		Type:     "junction",
		RepoType: "virtual",
		RepoKey:  "ampc-public-docker-release",
		Path:     "agent/" + agentNameToRepoPath[BuildAgentName],
		Text:     agentNameToRepoPath[BuildAgentName],
	}
	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		log.Trace(err)
		return nil, err
	}

	return &AgentVersionCheckJob{
		apiClient:    coreapi.NewClientWithTimeout(cfg.GetTLSConfig(), cfg.GetProxyURL(), cfg.GetClientTimeout()),
		buildVersion: buildVersion,
		requestBytes: requestBytes,
		headers: map[string]string{
			"X-Requested-With": "XMLHttpRequest",
			"Host":             "axway.jfrog.io",
			"Content-Length":   strconv.Itoa(len(requestBytes)),
			"Content-Type":     "application/json",
		},
	}, nil
}

// Ready -
func (avj *AgentVersionCheckJob) Ready() bool {
	return true
}

// Status -
func (avj *AgentVersionCheckJob) Status() error {
	return nil
}

// Execute - run agent version check job one time
func (avj *AgentVersionCheckJob) Execute() error {
	err := avj.getJFrogVersions()
	if err != nil {
		log.Trace(err)
		// Could not get update from jfrog.  Warn that we could not determine version and continue processing
		log.Warn("Agent cannot determine the next available release. Be aware that your agent could be outdated.")
	} else {
		// Successfully got jfrog version.  Now compare build to latest version
		if isVersionStringOlder(avj.buildVersion, config.AgentLatestVersion) {
			log.Warnf("New version available. Please consider upgrading from version %s to version %s", avj.buildVersion, config.AgentLatestVersion)
		}
	}
	return nil
}

// getJFrogVersions - obtaining the versions from JFrog website
// **Note** polling the jfrog website is the current solution to obtaining the list of versions
// In the future, adding a (Generic) resource for grouping versions together under the same scope is a possible solution
// ie: a new unscoped resource that represents the platform services, so that other products can plug in their releases.
func (avj *AgentVersionCheckJob) getJFrogVersions() error {
	request := coreapi.Request{
		Method:  http.MethodPost,
		URL:     jfrogURL,
		Headers: avj.headers,
		Body:    avj.requestBytes,
	}
	response, err := avj.apiClient.Send(request)
	if err != nil {
		return err
	}

	jfrogResponse := jfrogData{}
	err = json.Unmarshal(response.Body, &jfrogResponse)
	if err != nil {
		return err
	}

	config.AgentLatestVersion = avj.getLatestVersionFromJFrog(jfrogResponse.Items)
	return nil
}

func getBuildVersion() (string, error) {
	//remove -SHA from build version
	versionNoSHA := strings.Split(BuildVersion, "-")[0]

	//regex check for semantic versioning
	semVerRegexp := regexp.MustCompile(`\d.\d.\d`)
	if versionNoSHA == "" || !semVerRegexp.MatchString(versionNoSHA) {
		return "", errors.ErrStartingVersionChecker.FormatError("build version is missing or of noncompliant semantic versioning")
	}
	return versionNoSHA, nil
}

// isVersionStringOlder - return true if version of str1 is older than str2
func isVersionStringOlder(build string, latest string) bool {
	vB := getSemVer(build)
	vL := getSemVer(latest)

	return isVersionSmaller(vB, vL)
}

// isVersionSmaller - return true if version1 smaller than version2
func isVersionSmaller(v1 version, v2 version) bool {
	if v1.major < v2.major {
		return true
	}
	if v1.major == v2.major {
		if v1.minor < v2.minor {
			return true
		}
		if v1.minor == v2.minor && v1.patch < v2.patch {
			return true
		}
	}
	return false
}

func (avj *AgentVersionCheckJob) getLatestVersionFromJFrog(jfrogItems []jfrogItem) string {
	tempMaxVersion := version{
		major: 0,
		minor: 0,
		patch: 0,
		val:   "",
	}
	re := regexp.MustCompile(`\d{8}`)

	for _, item := range jfrogItems {
		//trimming version from jfrog webpage
		if item.Version != "latest" && item.Version != "" {
			v := getSemVer(item.Version)
			// avoid a version with an 8 digit date as the patch number: 1.0.20210421
			if !re.MatchString(strconv.Itoa(v.patch)) && isVersionSmaller(tempMaxVersion, v) {
				copyVersionStruct(&tempMaxVersion, v)
			}
		}
	}
	return tempMaxVersion.val
}

// getSemVer - getting a semantic version struct from version string
// pre-req is that string is already in semantic versioning with major, minor, and patch
func getSemVer(str string) version {
	s := strings.Split(str, ".")
	maj, err := strconv.Atoi(s[0])
	min, err2 := strconv.Atoi(s[1])
	pat, err3 := strconv.Atoi(s[2])
	if err == nil && err2 == nil && err3 == nil {
		v := version{
			major: maj,
			minor: min,
			patch: pat,
			val:   str,
		}
		return v
	}
	return version{}
}

// copyVersionStruct - copying version2 into version1 struct by value
func copyVersionStruct(v1 *version, v2 version) {
	v1.major = v2.major
	v1.minor = v2.minor
	v1.patch = v2.patch
	v1.val = v2.val
}

// startVersionCheckJobs - starts both a single run and continuous checks
func startVersionCheckJobs(cfg config.CentralConfig) {
	if !util.IsNotTest() || !cfg.IsVersionCheckerEnabled() {
		return
	}
	// register the agent version checker single run job
	checkJob, err := NewAgentVersionCheckJob(cfg)
	if err != nil {
		log.Errorf("could not create the agent version checker: %v", err.Error())
		return
	}
	if id, err := jobs.RegisterSingleRunJobWithName(checkJob, "Version Check"); err == nil {
		log.Tracef("registered agent version checker job: %s", id)
	}
	if id, err := jobs.RegisterScheduledJobWithName(checkJob, avcCronSchedule, "Version Check Schedule"); err == nil {
		log.Tracef("registered agent version checker cronjob: %s", id)
	}
}
