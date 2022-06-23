package provisioning

import "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

type Quota interface {
	// GetInterval returns the quota interval from within the access request
	GetInterval() QuotaInterval
	// GetIntervalString returns the string representation of the quota interval from within the access request
	GetIntervalString() string
	// GetLimit returns the quota limit from within the access request
	GetLimit() int64
}

// QuotaInterval is the quota limit
type QuotaInterval int

const (
	// The supported limits
	Unsupported QuotaInterval = iota + 1
	Daily
	Weekly
	Monthly
	Annually
)

// String returns the string value of the State
func (q QuotaInterval) String() string {
	return map[QuotaInterval]string{
		Daily:       "daily",
		Weekly:      "weekly",
		Monthly:     "monthly",
		Annually:    "annually",
		Unsupported: "",
	}[q]
}

// quotaLimitFromString returns the quota limit represented by the string sent in
func quotaLimitFromString(limit string) QuotaInterval {
	if q, ok := map[string]QuotaInterval{
		"daily":    Daily,
		"weekly":   Weekly,
		"monthly":  Monthly,
		"annually": Annually,
	}[limit]; ok {
		return q
	}
	return Unsupported
}

type quota struct {
	interval QuotaInterval
	limit    int64
}

func NewQuotaFromAccessRequest(ar *v1alpha1.AccessRequest) Quota {
	if ar.Spec.Quota == nil {
		return nil
	}
	return &quota{
		limit:    int64(ar.Spec.Quota.Limit),
		interval: quotaLimitFromString(ar.Spec.Quota.Interval),
	}
}

func (q *quota) GetInterval() QuotaInterval {
	return q.interval
}

func (q *quota) GetIntervalString() string {
	return q.interval.String()
}

func (q *quota) GetLimit() int64 {
	return q.limit
}
