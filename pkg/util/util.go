package util

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

const (
	// AmplifyCentral amplify central
	AmplifyCentral             = "Amplify Central"
	CentralHealthCheckEndpoint = "central"
)

// Remarshal - remarshal the bytes to remove any extra spaces and consistent key order
func Remarshal(data []byte) ([]byte, error) {
	var temp interface{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return nil, err
	}
	return json.Marshal(temp)
}

// ComputeHash - get the hash of the byte array sent in
func ComputeHash(data interface{}) (uint64, error) {
	// marshall the data in and out
	dataB, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	dataB, err = Remarshal(dataB)
	if err != nil {
		return 0, err
	}

	h := fnv.New64a()
	h.Write(dataB)
	return h.Sum64(), nil
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

// GetURLHostName - return the host name of the passed in URL
func GetURLHostName(urlString string) string {
	host, err := url.Parse(urlString)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return host.Hostname()
}

// ParsePort - parse port from URL
func ParsePort(url *url.URL) int {
	port := 0
	if url == nil {
		return port
	}

	if url.Port() == "" {
		port, _ = net.LookupPort("tcp", url.Scheme)
	} else {
		port, _ = strconv.Atoi(url.Port())
	}
	return port
}

// ParseAddr - parse host:port from URL
func ParseAddr(url *url.URL) string {
	if url == nil {
		return ""
	}

	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return fmt.Sprintf("%s:%d", url.Host, ParsePort(url))
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// StringSliceContains - does the given string slice contain the specified string?
func StringSliceContains(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveDuplicateValuesFromStringSlice - remove duplicate values from a string slice
func RemoveDuplicateValuesFromStringSlice(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// IsItemInSlice - Returns true if the given item is in the string slice, strSlice should be sorted
func IsItemInSlice(strSlice []string, item string) bool {
	if len(strSlice) == 0 {
		return false
	}
	if len(strSlice) == 1 {
		return strSlice[0] == item
	}
	midPoint := len(strSlice) / 2
	if item == strSlice[midPoint] {
		return true
	}
	if item < strSlice[midPoint] {
		return IsItemInSlice(strSlice[:midPoint], item)
	}
	return IsItemInSlice(strSlice[midPoint:], item)
}

// ConvertTimeToMillis - convert to milliseconds
func ConvertTimeToMillis(tm time.Time) int64 {
	return tm.UnixNano() / 1e6
}

// IsNotTest determines if a test is running or not
func IsNotTest() bool {
	return flag.Lookup("test.v") == nil
}

// RemoveUnquotedSpaces - Remove all whitespace not between matching quotes
func RemoveUnquotedSpaces(s string) (string, error) {
	rs := make([]rune, 0, len(s))
	const out = rune(0)
	var quote rune = out
	var escape = false
	for _, r := range s {
		if !escape {
			if r == '`' || r == '"' || r == '\'' {
				if quote == out {
					// start unescaped quote
					quote = r
				} else if quote == r {
					// end unescaped quote
					quote = out
				}
			}
		}
		// backslash (\) is the escape character
		// except when it is the second backslash of a pair
		escape = !escape && r == '\\'
		if quote != out || !unicode.IsSpace(r) {
			// between matching unescaped quotes
			// or not whitespace
			rs = append(rs, r)
		}
	}
	if quote != out {
		err := fmt.Errorf("unmatched unescaped quote: %q", quote)
		return "", err
	}
	return string(rs), nil
}

// CreateDirIfNotExist - Creates the directory with same permission as parent
func CreateDirIfNotExist(dirPath string) {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		dataInfo := getParentDirInfo(dirPath)
		os.MkdirAll(dirPath, dataInfo.Mode().Perm())
	}
}

func getParentDirInfo(dirPath string) fs.FileInfo {
	parent := filepath.Dir(dirPath)
	dataInfo, err := os.Stat(parent)
	if os.IsNotExist(err) {
		return getParentDirInfo(parent)
	}
	return dataInfo
}

// MergeMapStringInterface - merges the provided maps.
// If duplicate keys are found across the maps, then the keys in map n will be overwritten in keys in map n+1
func MergeMapStringInterface(m ...map[string]interface{}) map[string]interface{} {
	attrs := make(map[string]interface{})

	for _, item := range m {
		for k, v := range item {
			attrs[k] = v
		}
	}

	return attrs
}

// MergeMapStringString - merges the provided maps.
// If duplicate keys are found across the maps, then the keys in map n will be overwritten in keys in map n+1.
func MergeMapStringString(m ...map[string]string) map[string]string {
	attrs := make(map[string]string)

	for _, item := range m {
		for k, v := range item {
			attrs[k] = v
		}
	}

	return attrs
}

// CheckEmptyMapStringString creates a new empty map if the provided map is nil
func CheckEmptyMapStringString(m map[string]string) map[string]string {
	if m == nil {
		return make(map[string]string)
	}

	return m
}

// MapStringStringToMapStringInterface converts a map[string]string to map[string]interface{}
func MapStringStringToMapStringInterface(m map[string]string) map[string]interface{} {
	newMap := make(map[string]interface{})

	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

// ToString converts an interface{} to a string
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// IsNil checks a value, or a pointer for nil
func IsNil(v interface{}) bool {
	return v == nil || reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil()
}

// MapStringInterfaceToStringString - convert map[string]interface{} to map[string]string given the item can be a string
func MapStringInterfaceToStringString(data map[string]interface{}) map[string]string {
	newData := make(map[string]string)

	for k, v := range data {
		newData[k] = ""
		if v == nil {
			continue
		} else {
			newData[k] = fmt.Sprintf("%+v", v)
		}
	}
	return newData
}

// ConvertStringToUint -
func ConvertStringToUint(val string) uint64 {
	ret, _ := strconv.ParseUint(val, 10, 64)
	return ret
}

// ConvertUnitToString -
func ConvertUnitToString(val uint64) string {
	return strconv.FormatUint(val, 10)
}

// ReadPrivateKeyFile - reads and parses the private key content
func ReadPrivateKeyFile(privateKeyFile, passwordFile string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	// cleanup private key read bytes
	defer func() {
		for i := range keyBytes {
			keyBytes[i] = 0
		}
	}()

	if passwordFile != "" {
		var passwordBuf []byte
		var err error
		// cleanup password bytes
		defer func() {
			for i := range passwordBuf {
				passwordBuf[i] = 0
			}
		}()

		passwordBuf, err = readPassword(passwordFile)
		if err != nil {
			return nil, err
		}

		if len(passwordBuf) > 0 {
			key, err := parseRSAPrivateKeyFromPEMWithBytePassword(keyBytes, passwordBuf)
			if err != nil {
				return nil, err
			}

			return key, nil

		}
		log.Debug("password file empty, assuming unencrypted key")
		return jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	}

	log.Debug("no password, assuming unencrypted key")
	return jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
}

func readPassword(passwordFile string) ([]byte, error) {
	return os.ReadFile(passwordFile)
}

// ReadPublicKeyBytes - reads the public key bytes from file
func ReadPublicKeyBytes(publicKeyFile string) ([]byte, error) {
	keyBytes, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}

// parseRSAPrivateKeyFromPEMWithBytePassword - tries to parse an rsa private key using password as bytes
// inspired from jwt.ParseRSAPrivateKeyFromPEMWithPassword
func parseRSAPrivateKeyFromPEMWithBytePassword(key []byte, password []byte) (*rsa.PrivateKey, error) {
	var err error

	// trim any spaces from the password
	password = bytes.TrimSpace(password)

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, fmt.Errorf("key must be pem encoded")
	}

	var parsedKey interface{}

	var blockDecrypted []byte
	if blockDecrypted, err = x509.DecryptPEMBlock(block, password); err != nil {
		return nil, err
	}

	if parsedKey, err = x509.ParsePKCS1PrivateKey(blockDecrypted); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(blockDecrypted); err != nil {
			return nil, err
		}
	}

	var pkey *rsa.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(*rsa.PrivateKey); !ok {
		return nil, fmt.Errorf("[apicauth] not a private key")
	}

	return pkey, nil
}

// ParsePublicKey - parses the public key content
func ParsePublicKey(publicKey []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err)
	}

	p, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected public key type to be *rsa.PublicKey but received %T", pub)
	}
	return p, nil
}

// ParsePublicKeyDER - parse DER block from public key
func ParsePublicKeyDER(publicKey []byte) ([]byte, error) {
	if b64key, err := base64.StdEncoding.DecodeString(string(publicKey)); err == nil {
		return b64key, nil
	}

	_, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		pemBlock, _ := pem.Decode(publicKey)
		if pemBlock == nil {
			return nil, errors.New("data in key was not valid")
		}
		if pemBlock.Type != "PUBLIC KEY" {
			return nil, errors.New("unsupported key type")
		}
		return pemBlock.Bytes, nil
	}
	return publicKey, nil
}

// ComputeKIDFromDER - compute key ID for public key
func ComputeKIDFromDER(publicKey []byte) (kid string, err error) {
	b64key, err := ParsePublicKeyDER(publicKey)
	if err != nil {
		return "", err
	}
	h := sha256.New() // create new hash with sha256 checksum
	/* #nosec G104 */
	if _, err := h.Write(b64key); err != nil { // add b64key to hash
		return "", err
	}
	e := base64.StdEncoding.EncodeToString(h.Sum(nil)) // return string of base64 encoded hash
	kid = strings.Split(e, "=")[0]
	kid = strings.Replace(kid, "+", "-", -1)
	kid = strings.Replace(kid, "/", "_", -1)
	return
}

// GetStringFromMapInterface - returns the validated string for the map element
func GetStringFromMapInterface(key string, data map[string]interface{}) string {
	if e, ok := data[key]; ok && e != nil {
		if value, ok := e.(string); ok {
			return value
		}
	}
	return ""
}

// GetStringArrayFromMapInterface - returns the validated string array for the map element
func GetStringArrayFromMapInterface(key string, data map[string]interface{}) []string {
	val := []string{}
	if e, ok := data[key]; ok && e != nil {
		if i, ok := e.([]interface{}); ok {
			for _, u := range i {
				if s, ok := u.(string); ok {
					val = append(val, s)
				}
			}
		}
		if sa, ok := e.([]string); ok {
			val = append(val, sa...)
		}
	}
	return val
}

// ConvertToDomainNameCompliant - converts string to be domain name complaint
func ConvertToDomainNameCompliant(str string) string {
	// convert all letters to lower first
	newName := strings.ToLower(str)

	// parse name out. All valid parts must be '-', '.', a-z, or 0-9
	re := regexp.MustCompile(`[-\.a-z0-9]*`)
	matches := re.FindAllString(newName, -1)

	// join all of the parts, separated with '-'. This in effect is substituting all illegal chars with a '-'
	newName = strings.Join(matches, "-")

	// The regex rule says that the name must not begin or end with a '-' or '.', so trim them off
	newName = strings.TrimLeft(strings.TrimRight(newName, "-."), "-.")

	// The regex rule also says that the name must not have a sequence of ".-", "-.", or "..", so replace them
	r1 := strings.ReplaceAll(newName, "-.", "--")
	r2 := strings.ReplaceAll(r1, ".-", "--")
	return strings.ReplaceAll(r2, "..", "--")
}

func OrderStringsInMap[T any](input map[string]T) map[string]T {
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	output := map[string]T{}
	for _, k := range keys {
		output[k] = input[k]
	}
	return output
}

func OrderedKeys[T any](input map[string]T) []string {
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// SanitizeServiceVersion - apiserver limits version size to 30.  If its greater than 30, compute hash for the version and use that for the displayed value
func SanitizeServiceVersion(version string) string {
	sanitizedVersion := version
	if len(version) > 30 {
		hashInt, err := ComputeHash(version)
		if err != nil {
			// if hash computation fails, take the substring of version up to 30 chars (apiserver limit). That's the least that can be done
			sanitizedVersion = version[:30] // Gets first 30 bytes
		} else {
			sanitizedVersion = strconv.FormatUint(hashInt, 10)
		}
	}
	return sanitizedVersion
}

// EnsureStringIsNotFloat - ensures the string is not a float representation but a whole number
func EnsureStringIsNotFloat(in string) string {
	if fResult, err := strconv.ParseFloat(in, 64); err == nil {
		return fmt.Sprintf("%.0f", fResult)
	}
	return in
}
