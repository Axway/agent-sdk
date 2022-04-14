package definitions

import (
	"testing"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetSubscriptionNameFromAccessReq(t *testing.T) {
	subscriptionName := GetSubscriptionNameFromAccessRequest(nil)
	assert.Equal(t, "", subscriptionName)

	// Reference from group other than catalog
	ar := &mv1.AccessRequest{
		References: []interface{}{
			mv1.AccessRequestReferencesSubscription{
				Kind: "Subscription",
				Name: "management/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)

	// Reference from catalog group
	ar = &mv1.AccessRequest{
		References: []interface{}{
			mv1.AccessRequestReferencesSubscription{
				Kind: "Subscription",
				Name: "catalog/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ar = &mv1.AccessRequest{
		References: []interface{}{
			&mv1.AccessRequestReferencesSubscription{
				Kind: "Subscription",
				Name: "catalog/test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ri, _ := ar.AsInstance()
	ar = &mv1.AccessRequest{}
	ar.FromInstance(ri)

	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "test", subscriptionName)

	ar = &mv1.AccessRequest{
		References: []interface{}{
			&mv1.AccessRequestReferencesSubscription{
				Name: "test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)

	ar = &mv1.AccessRequest{
		References: []interface{}{
			&mv1.AccessRequestReferencesApplication{
				Kind: "Application",
				Name: "test",
			},
		},
	}
	subscriptionName = GetSubscriptionNameFromAccessRequest(ar)
	assert.Equal(t, "", subscriptionName)
}
