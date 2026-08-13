package apic

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/stretchr/testify/assert"
)

// itemsJSON builds a JSON array of ResourceInstance-shaped objects, one per name.
func itemsJSON(names ...string) string {
	items := make([]string, len(names))
	for i, n := range names {
		items[i] = fmt.Sprintf(`{"name":%q}`, n)
	}
	return "[" + strings.Join(items, ",") + "]"
}

func namesOf(ris []*apiv1.ResourceInstance) []string {
	names := make([]string, len(ris))
	for i, ri := range ris {
		names[i] = ri.Name
	}
	return names
}

func countResponse(count int) api.MockResponse {
	return api.MockResponse{
		RespCode: http.StatusOK,
		// must be pre-canonicalized: GetAPIV1ResourceCount looks this up via http.CanonicalHeaderKey
		RespHeaders: map[string][]string{http.CanonicalHeaderKey("X-Axway-total-count"): {strconv.Itoa(count)}},
	}
}

func itemsResponse(names ...string) api.MockResponse {
	return api.MockResponse{RespCode: http.StatusOK, RespData: itemsJSON(names...)}
}

// TestGetAPIV1ResourceInstances_MultiPage exercises the worker-pool dispatch/collector path
// end to end. The first page comes back full, which triggers a HEAD count call and concurrent
// fetching of the remaining pages. numberOfWorkers=1 keeps request order deterministic so it
// lines up with the mock HTTP client's strict FIFO response queue (pkg/api/mockhttpclient.go
// serves responses by arrival order only, with no per-request matching).
func TestGetAPIV1ResourceInstances_MultiPage(t *testing.T) {
	const url = "/test"
	client, httpClient := GetTestServiceClient()
	client.numberOfWorkers = 1
	client.setPageSizeIfSmaller(url, 5)

	httpClient.SetResponses([]api.MockResponse{
		itemsResponse("item-0", "item-1", "item-2", "item-3", "item-4"), // first page, full - triggers count+concurrent fetch
		countResponse(12),
		itemsResponse("item-5", "item-6", "item-7", "item-8", "item-9"),
		itemsResponse("item-10", "item-11"),
	})

	ris, err := client.GetAPIV1ResourceInstances(map[string]string{}, url)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"item-0", "item-1", "item-2", "item-3", "item-4",
		"item-5", "item-6", "item-7", "item-8", "item-9",
		"item-10", "item-11",
	}, namesOf(ris))
}

// TestGetAPIV1ResourceInstances_RetryOnTimeout verifies that when the very first fetch times
// out, GetAPIV1ResourceInstances falls back to a HEAD count call and concurrent fetching from
// page 0, and that a worker hitting "context deadline exceeded" on its assigned range retries
// that same range through reducePageParams (instead of dropping it), halving the page size.
// Page 1 is dispatched at the original size 10, but by the time the (single) worker gets to
// it, page 0's retry has already discovered that size 5 works - so fetchPageParams's
// shared-size check skips straight to size 5 for page 1 too, without a doomed attempt at 10.
func TestGetAPIV1ResourceInstances_RetryOnTimeout(t *testing.T) {
	const url = "/test"
	client, httpClient := GetTestServiceClient()
	client.numberOfWorkers = 1
	client.setPageSizeIfSmaller(url, 10)

	httpClient.SetResponses([]api.MockResponse{
		{RespCode: http.StatusRequestTimeout, ErrString: "context deadline exceeded"}, // first fetch, page 0 size 10 - times out
		countResponse(15),
		{RespCode: http.StatusRequestTimeout, ErrString: "context deadline exceeded"}, // worker retries page 0 at size 10 - times out again
		itemsResponse("a0", "a1", "a2", "a3", "a4"),                                   // page 0 reduced sub-range 1, size 5
		itemsResponse("a5", "a6", "a7", "a8", "a9"),                                   // page 0 reduced sub-range 2, size 5
		itemsResponse("a10", "a11", "a12", "a13", "a14"),                              // page 1 reduced sub-range 1, size 5 (skipped straight to size 5)
		itemsResponse(), // page 1 reduced sub-range 2, size 5 - no data left
	})

	ris, err := client.GetAPIV1ResourceInstances(map[string]string{}, url)
	assert.NoError(t, err)
	expected := make([]string, 15)
	for i := range expected {
		expected[i] = fmt.Sprintf("a%d", i)
	}
	assert.ElementsMatch(t, expected, namesOf(ris))

	size, ok := client.getPageSize(url)
	assert.True(t, ok)
	assert.Equal(t, 5, size)
}

// TestGetAPIV1ResourceInstances_ConcurrentSmoke runs with multiple workers to confirm the
// dispatch/collector pipeline doesn't deadlock or panic under real concurrency (run with
// -race). Since the mock HTTP client hands out responses strictly FIFO with no per-request
// matching, only the aggregate item count is checked here, not which page got which response.
func TestGetAPIV1ResourceInstances_ConcurrentSmoke(t *testing.T) {
	const url = "/test"
	client, httpClient := GetTestServiceClient()
	client.numberOfWorkers = 3
	client.setPageSizeIfSmaller(url, 5)

	httpClient.SetResponses([]api.MockResponse{
		itemsResponse("i0", "i1", "i2", "i3", "i4"), // first page, full - triggers count+concurrent fetch
		countResponse(23),
		itemsResponse("i5", "i6", "i7", "i8", "i9"),
		itemsResponse("i10", "i11", "i12", "i13", "i14"),
		itemsResponse("i15", "i16", "i17", "i18", "i19"),
		itemsResponse("i20", "i21", "i22"),
	})

	ris, err := client.GetAPIV1ResourceInstances(map[string]string{}, url)
	assert.NoError(t, err)
	assert.Len(t, ris, 23)
}

// TestGetAPIV1ResourceInstances_CountSmallerThanFirstPage covers a HEAD count that comes
// back smaller than the already-fetched first page - e.g. the X-Axway-total-count header is
// absent (GetAPIV1ResourceCount then returns 0, not an error) or items were deleted between
// the initial fetch and the HEAD request. This used to panic: numPages ended up smaller than
// startPage, and fetchPagesConcurrently tried to make a slice with a negative capacity.
func TestGetAPIV1ResourceInstances_CountSmallerThanFirstPage(t *testing.T) {
	const url = "/test"
	client, httpClient := GetTestServiceClient()
	client.numberOfWorkers = 1
	client.setPageSizeIfSmaller(url, 5)

	httpClient.SetResponses([]api.MockResponse{
		itemsResponse("i0", "i1", "i2", "i3", "i4"), // first page, exactly full
		{RespCode: http.StatusOK},                   // HEAD response with no count header
	})

	ris, err := client.GetAPIV1ResourceInstances(map[string]string{}, url)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"i0", "i1", "i2", "i3", "i4"}, namesOf(ris))
}

// TestGetAPIV1ResourceInstancesWithPageSize covers getAPIV1ResourceInstancesWithPageSize's
// actual contract: a single fetch that trims by skipFirstN/skipLastN and propagates errors
// as-is. It does not retry or track page size itself - that's fetchPageParams/reduceAndRetry's
// job (covered separately below and by the GetAPIV1ResourceInstances_* tests above), reached
// only through the worker pool.
func TestGetAPIV1ResourceInstancesWithPageSize(t *testing.T) {
	const url = "/test"

	t.Run("trims by skipFirstN and skipLastN", func(t *testing.T) {
		client, httpClient := GetTestServiceClient()
		httpClient.SetResponses([]api.MockResponse{itemsResponse("a0", "a1", "a2", "a3", "a4")})

		data, err := client.getAPIV1ResourceInstancesWithPageSize(map[string]string{"key": "value"}, url, pageParams{5, 0, 1, 1})
		assert.NoError(t, err)
		assert.Equal(t, []string{"a1", "a2", "a3"}, namesOf(data))
	})

	t.Run("context deadline exceeded is returned, not retried or tracked", func(t *testing.T) {
		client, httpClient := GetTestServiceClient()
		httpClient.SetResponses([]api.MockResponse{
			{RespCode: http.StatusRequestTimeout, ErrString: "context deadline exceeded"},
		})

		data, err := client.getAPIV1ResourceInstancesWithPageSize(map[string]string{"key": "value"}, url, pageParams{100, 0, 0, 0})
		assert.Nil(t, data)
		assert.ErrorContains(t, err, "context deadline exceeded")
		assert.Len(t, httpClient.Requests, 1) // no internal retry

		_, ok := client.getPageSize(url)
		assert.False(t, ok) // this function never touches the tracked page size
	})
}

// TestReduceAndRetry_GivesUp covers the give-up conditions reduceAndRetry checks before
// halving further: the retry budget running out, and the halved size dropping below
// minPageSize. Calling it directly (same package) avoids needing an HTTP-mocked cascade to
// exercise these, since both checks happen before any fetch is attempted.
func TestReduceAndRetry_GivesUp(t *testing.T) {
	origErr := errors.New("context deadline exceeded")

	testCases := map[string]struct {
		params  pageParams
		retries int
		origErr error
	}{
		"retries exhausted": {
			params:  pageParams{100, 0, 0, 0},
			retries: 0,
			origErr: origErr,
		},
		"halved size would drop below minPageSize": {
			params:  pageParams{2*minPageSize - 2, 0, 0, 0}, // pageSize/2 < minPageSize
			retries: maxPageSizeRetries,
			origErr: origErr,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client, _ := GetTestServiceClient()
			chRI := make(chan apiv1.ResourceInstance, 1)
			chErr := make(chan error, 1)

			client.reduceAndRetry(map[string]string{}, "/test", tc.params, chRI, chErr, tc.retries, tc.origErr)

			assert.Empty(t, chRI)
			assert.Same(t, tc.origErr, <-chErr)
		})
	}
}

// TestReduceAndRetry_SynthesizesErrorWhenGivingUpPreemptively verifies that giving up with a
// nil origErr (the shared-size preemptive-reduce path in fetchPageParams never attempts a real
// fetch) still reports a non-nil error, so the range isn't silently dropped by errors.Join.
func TestReduceAndRetry_SynthesizesErrorWhenGivingUpPreemptively(t *testing.T) {
	client, _ := GetTestServiceClient()
	chRI := make(chan apiv1.ResourceInstance, 1)
	chErr := make(chan error, 1)

	client.reduceAndRetry(map[string]string{}, "/test", pageParams{2*minPageSize - 2, 0, 0, 0}, chRI, chErr, maxPageSizeRetries, nil)

	assert.Empty(t, chRI)
	err := <-chErr
	assert.Error(t, err)
}
