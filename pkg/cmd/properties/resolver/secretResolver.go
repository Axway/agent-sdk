package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	secretConfigPrefix  = "@Secret."
	secretMapItemPrefix = "SecretResource_"
)

// SecretResolver - Interface to resolve secret reference
type SecretResolver interface {
	properties.SecretPropertyResolver
	ResetResolver()
}

type secretResolver struct {
	SecretResolver
	secretsCache cache.Cache
}

// NewSecretResolver - create a new secret resolver
func NewSecretResolver() SecretResolver {
	return &secretResolver{
		secretsCache: cache.New(),
	}
}

// parseSecretRef - parses the secret reference with prefixed secret name ane key
func (s *secretResolver) parseSecretRef(secretRef string) (string, string) {
	// parse secret and key from @Secret.secretName.key
	secretRef = strings.TrimSpace(secretRef)
	if strings.HasPrefix(secretRef, secretConfigPrefix) {
		secretRef = secretRef[len(secretConfigPrefix):]
		secretRefElements := strings.Split(secretRef, ".")
		if len(secretRefElements) > 1 {
			return secretRefElements[0], strings.Join(secretRefElements[1:], ".")
		}
	}
	return "", ""
}

func (s *secretResolver) getSecret(secretName string) (*v1alpha1.Secret, error) {
	secretResourceURL := agent.GetCentralConfig().GetEnvironmentURL() + "/secrets/" + secretName

	response, err := agent.GetCentralClient().ExecuteAPI(coreapi.GET, secretResourceURL, nil, nil)
	if err != nil {
		return nil, err
	}
	secret := &v1alpha1.Secret{}
	err = json.Unmarshal(response, secret)
	return secret, err
}

func (s *secretResolver) parseKeyValueFromSecretSpec(secret *v1alpha1.Secret, key string) (string, error) {
	// Return empty string if secret key not found
	keyVal, ok := secret.Spec.Data[key]
	if !ok {
		msg := fmt.Sprintf("key %s not found in secret %s", key, secret.Name)
		return "", errors.New(msg)
	}
	return keyVal, nil
}

func (s *secretResolver) ResolveSecret(secretRef string) (string, error) {
	// Do not parse secret reference until central config is parsed and initialized
	// secret ref be applied to agent config only and not central config
	cfg := agent.GetCentralConfig()
	if cfg == nil || reflect.ValueOf(cfg).IsNil() {
		return secretRef, nil
	}

	// If usage reporting is offline, do not resolve secretclear
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() && strings.HasPrefix(secretRef, secretConfigPrefix) {
		msg := "Securing password with @Secret resource is not possible when running agent in offline mode."
		log.Debugf("@Secret reference %s is not valid when offline mode is true.", secretRef)
		return "", errors.New(msg)
	}

	secretName, key := s.parseSecretRef(secretRef)
	if secretName != "" && key != "" {
		var secret *v1alpha1.Secret
		// Get cached secret to resolve key
		cachedSecret, err := s.secretsCache.Get(secretMapItemPrefix + secretName)
		if err != nil {
			// Secret not cached, get the secret from API server
			secret, err = s.getSecret(secretName)
			if err != nil {
				log.Trace(err.Error())
				msg := fmt.Sprintf("unable to resolve secret %s", secretName)
				return "", errors.New(msg)
			}
			s.secretsCache.Set(secretMapItemPrefix+secret.GetName(), secret)
		} else {
			secret, _ = cachedSecret.(*v1alpha1.Secret)
		}
		keyVal, err := s.parseKeyValueFromSecretSpec(secret, key)
		if err != nil {
			return "", err
		}
		return keyVal, nil
	}
	// Not a secret ref, return it as value
	return secretRef, nil
}

func (s *secretResolver) ResetResolver() {
	s.secretsCache = cache.New()
}
