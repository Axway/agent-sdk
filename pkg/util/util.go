package util

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

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
