package util

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/sirupsen/logrus"
	"github.com/subosito/gotenv"
)

// ComputeHash - get the hash of the byte array sent in
func ComputeHash(data interface{}) (uint64, error) {
	dataB, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("Could not marshal data to bytes")
	}

	h := fnv.New64a()
	h.Write(dataB)
	return h.Sum64(), nil
}

// LoadEnvFromFile - Loads the environment variables from a file
func LoadEnvFromFile(envFile string) error {
	if envFile != "" {
		err := gotenv.Load(envFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// MaskValue - mask sensitive information with * (asterisk).  Length of sensitiveData to match returning maskedValue
func MaskValue(sensitiveData string) string {
	var maskedValue string
	for i := 0; i < len(sensitiveData); i++ {
		maskedValue += "*"
	}
	return maskedValue
}

// PrintDataInterface - prints contents of the interface only if in debug mode
func PrintDataInterface(data interface{}) {
	if log.GetLevel() == logrus.DebugLevel {
		PrettyPrint(data)
	}
}

// PrettyPrint - print the contents of the obj
func PrettyPrint(data interface{}) {
	var p []byte
	//    var err := error
	p, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s \n", p)
}

// GetProxyURL - need to provide my own function (instead of http.ProxyURL()) to handle empty url. Returning nil
// means "no proxy"
func GetProxyURL(fixedURL *url.URL) func(*http.Request) (*url.URL, error) {
	return func(*http.Request) (*url.URL, error) {
		if fixedURL == nil || fixedURL.Host == "" {
			return nil, nil
		}
		return fixedURL, nil
	}
}

// StringArrayContains - does the given array of strings contain the specified string?
func StringArrayContains(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}
