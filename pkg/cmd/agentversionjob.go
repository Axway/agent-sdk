package cmd

import (
	"bytes"
	"encoding/xml"

	"github.com/Axway/agent-sdk/pkg/config"

	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"net/http"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

var agentURL = map[string]string{
	"AWSDiscoveryAgent":                      "aws-apigw-discovery-agent",
	"AWSTraceabilityAgent":                   "aws-apigw-traceability-agent",
	"AzureDiscoveryAgent":                    "azure-discovery-agent",
	"AzureTraceabilityAgent":                 "azure-traceability-agent",
	"EnterpriseEdgeGatewayDiscoveryAgent":    "v7-discovery-agent",
	"EnterpriseEdgeGatewayTraceabilityAgent": "v7-traceability-agent",
}

type htmlAnchors struct {
	VersionList []string `xml:"body>pre>a"`
}

type version struct {
	major, minor, patch int
	val                 string
}

// AgentVersionCheckJob - polls for agent versions
type AgentVersionCheckJob struct {
	jobs.Job
	allVersions   []string
	buildVersion  string
	dataPlaneType string
	urlName       string
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
	avj.dataPlaneType = BuildDataPlaneType
	avj.urlName = agentURL[avj.dataPlaneType]
	if avj.urlName == "AgentSDK" || avj.urlName == "" {
		err := errors.ErrStartingVersionChecker.FormatError("empty or generic data plane type name")
		log.Trace(err)
		return err
	}
	err := avj.getBuildVersion()
	if err != nil {
		log.Trace(err)
		return err
	}
	err = avj.getJFrogVersions(avj.urlName)
	if err != nil {
		log.Trace(err)
		return err
	}
	// compare build to latest version
	if isVersionStringOlder(avj.buildVersion, config.AgentLatestVersion) {
		log.Warnf("New version available. Please consider upgrading from version %s to version %s", avj.buildVersion, config.AgentLatestVersion)
	}
	return nil
}

func (avj *AgentVersionCheckJob) getBuildVersion() error {
	avj.buildVersion = BuildVersion
	//remove -SHA from build version
	noSHA := strings.Split(avj.buildVersion, "-")
	avj.buildVersion = noSHA[0]

	//regex check for semantic versioning
	semVerRegexp := regexp.MustCompile(`\d.\d.\d`)
	if avj.buildVersion == "" || !semVerRegexp.MatchString(avj.buildVersion) {
		return errors.ErrStartingVersionChecker.FormatError("build version is missing or of noncompliant semantic versioning")
	}
	return nil
}

// getJFrogVersions - obtaining the versions from JFrog website
// **Note** polling the jfrog website is the current solution to obtaining the list of versions
// In the future, adding a (Generic) resource for grouping versions together under the same scope is a possible solution
// ie: a new unscoped resource that represents the platform services, so that other products can plug in their releases.
func (avj *AgentVersionCheckJob) getJFrogVersions(name string) error {
	b := loadPage(name)

	hAnchors := htmlAnchors{}
	err := xml.NewDecoder(bytes.NewBuffer(b)).Decode(&hAnchors)
	if err != nil {
		return err
	}

	avj.allVersions = hAnchors.VersionList
	config.AgentLatestVersion = avj.getLatestVersionFromJFrog()
	return nil
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

func (avj *AgentVersionCheckJob) getLatestVersionFromJFrog() string {
	tempMaxVersion := version{
		major: 0,
		minor: 0,
		patch: 0,
		val:   "",
	}
	re := regexp.MustCompile(`\d{8}`)

	for _, v := range avj.allVersions {
		//trimming version of / that comes from jfrog webpage
		trimmed := strings.TrimSuffix(v, "/")
		if trimmed != ".." && trimmed != "latest" && trimmed != "" {
			v := getSemVer(trimmed)
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

func loadPage(name string) []byte {
	page := fmt.Sprintf("https://axway.jfrog.io/artifactory/ampc-public-docker-release/agent/%v/", name)
	resp, err := http.Get(page)
	if err != nil {
		log.Tracef("Unable to poll jfrog for agent versions. %s", err.Error())
	}
	defer resp.Body.Close()
	// reads html as a slice of bytes
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Trace(err)
	}
	return html
}

// startAgentVersionChecker - single run job to check for a newer agent version on jfrog
func startAgentVersionChecker() {
	// register the agent version checker single run job
	id, err := jobs.RegisterSingleRunJobWithName(&AgentVersionCheckJob{}, "Version Check")
	if err != nil {
		log.Errorf("could not start the agent version checker job: %v", err.Error())
		return
	}
	log.Tracef("registered agent version checker job: %s", id)
}
