package definitions

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetSubscriptionNameFromAccessReq(t *testing.T) {
	subscriptionName := GetSubscriptionNameFromAccessRequest(nil)
	assert.Equal(t, "", subscriptionName)

	// Reference from group other than catalog
	ar := &management.AccessRequest{
		References: []interface{}{
			management.AccessRequestReferencesSubscription{
				Kind: Subscription,
				Name: "management/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)

	// Reference from catalog group
	ar = &management.AccessRequest{
		References: []interface{}{
			management.AccessRequestReferencesSubscription{
				Kind: Subscription,
				Name: "catalog/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ar = &management.AccessRequest{
		References: []interface{}{
			&management.AccessRequestReferencesSubscription{
				Kind: Subscription,
				Name: "catalog/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ri, _ := ar.AsInstance()
	ar = &management.AccessRequest{}
	ar.FromInstance(ri)

	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ar = &management.AccessRequest{
		References: []interface{}{
			&management.AccessRequestReferencesSubscription{
				Name: "test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)

	ar = &management.AccessRequest{
		References: []interface{}{
			&management.AccessRequestReferencesApplication{
				Kind: "Application",
				Name: "test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)
}
