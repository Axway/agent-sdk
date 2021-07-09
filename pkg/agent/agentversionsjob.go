package agent

import (
	"bytes"
	"encoding/xml"

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

// agentVersionCheckJob - polls for agent versions
type agentVersionCheckJob struct {
	jobs.Job
	allVersions   []string
	latestVersion string
	buildVersion  string
	dataPlaneType string
	urlName       string
}

// Ready -
func (avj *agentVersionCheckJob) Ready() bool {
	avj.setURLName()
	return true
}

// Status -
func (avj *agentVersionCheckJob) Status() error {
	return nil
}

// Execute - run agent version check job one time
func (avj *agentVersionCheckJob) Execute() error {
	if avj.urlName == "AgentSDK" || avj.urlName == "" {
		err := errors.ErrStartingVersionChecker.FormatError("empty or generic data plane type name")
		log.Error(err)
		return err
	}
	err := avj.getBuildVersion()
	if err != nil {
		log.Error(err)
		return err
	}
	err = avj.getJFrogVersions(avj.urlName)
	if err != nil {
		log.Error(err)
		return err
	}

	// compare build to latest version
	if isVersionOlder(avj.buildVersion, avj.latestVersion) {
		log.Infof("Running older version of %s. Please consider upgrading from version %s to version %s", avj.dataPlaneType, avj.buildVersion, avj.latestVersion)
	}
	return nil
}

// isVersionOlder - return true if version of arg1 is older than arg2
func isVersionOlder(build string, latest string) bool {
	vB := getSemVer(build)
	vL := getSemVer(latest)

	if vB.major < vL.major {
		return true
	} else if vB.major == vL.major && vB.minor < vL.minor {
		return true
	} else if vB.major == vL.major && vB.minor == vL.minor && vB.patch < vL.patch {
		return true
	}

	return false
}

func (avj *agentVersionCheckJob) setURLName() {
	avj.dataPlaneType = agent.cfg.GetBuildDataPlaneType()
	avj.urlName = agentURL[avj.dataPlaneType]
}

func (avj *agentVersionCheckJob) getBuildVersion() error {
	avj.buildVersion = agent.cfg.GetBuildVersion()
	//remove -SHA from build version
	noSHA := strings.Split(avj.buildVersion, "-")
	avj.buildVersion = noSHA[0]

	//regex check for semantic versioning
	semVerRegexp := regexp.MustCompile(`\d.\d.\d`)
	if avj.buildVersion == "" || !semVerRegexp.MatchString(avj.buildVersion) {
		return errors.ErrStartingVersionChecker.FormatError("missing or non compliant build version")
	}
	return nil
}

func (avj *agentVersionCheckJob) getJFrogVersions(name string) error {
	b := loadPage(name)

	hAnchors := htmlAnchors{}
	err := xml.NewDecoder(bytes.NewBuffer(b)).Decode(&hAnchors)
	if err != nil {
		return err
	}

	avj.allVersions = hAnchors.VersionList
	avj.latestVersion = avj.setLatestFromJFrog()
	return nil
}

func (avj *agentVersionCheckJob) setLatestFromJFrog() string {
	maxTempVersion := version{
		major: 0,
		minor: 0,
		patch: 0,
		val:   "",
	}
	re := regexp.MustCompile(`\d{8}`)

	for _, v := range avj.allVersions {
		trimmed := strings.TrimSuffix(v, "/")
		if trimmed != ".." && trimmed != "latest" && trimmed != "" {
			v := getSemVer(trimmed)
			// avoid a version with an 8 digit date as the patch number: 1.0.20210421
			if !re.MatchString(strconv.Itoa(v.patch)) {
				if maxTempVersion.major < v.major {
					maxTempVersion.major = v.major
					maxTempVersion.minor = v.minor
					maxTempVersion.patch = v.patch
					maxTempVersion.val = v.val
				} else if maxTempVersion.major == v.major && maxTempVersion.minor < v.minor {
					maxTempVersion.major = v.major
					maxTempVersion.minor = v.minor
					maxTempVersion.patch = v.patch
					maxTempVersion.val = v.val
				} else if maxTempVersion.major == v.major && maxTempVersion.minor == v.minor && maxTempVersion.patch < v.patch {
					maxTempVersion.major = v.major
					maxTempVersion.minor = v.minor
					maxTempVersion.patch = v.patch
					maxTempVersion.val = v.val
				}
			}
		}
	}
	return maxTempVersion.val
}

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

func loadPage(name string) []byte {
	page := fmt.Sprintf("https://axway.jfrog.io/artifactory/ampc-public-docker-release/agent/%v/", name)
	resp, err := http.Get(page)
	if err != nil {
		log.Errorf("Unable to poll jfrog for agent versions. %s", err.Error())
	}
	defer resp.Body.Close()
	// reads html as a slice of bytes
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	return html
}
