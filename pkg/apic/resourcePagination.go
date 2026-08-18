package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
)

const (
	minPageSize = 5
)

// GetAPIServiceRevisions - management.APIServiceRevision
func (c *ServiceClient) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*management.APIServiceRevision, error) {
	resources, err := c.GetAPIV1ResourceInstances(queryParams, URL)
	if err != nil {
		return nil, err
	}
	apiServiceInstances, err := management.APIServiceRevisionFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	filteredAPIRevisions := make([]*management.APIServiceRevision, 0)

	// create array and filter by stage name. Check the stage name as this does not apply for v7
	if stage != "" {
		for _, apiServer := range apiServiceInstances {
			if strings.Contains(strings.ToLower(apiServer.Name), strings.ToLower(fmt.Sprintf("%s.", stage))) {
				filteredAPIRevisions = append(filteredAPIRevisions, apiServer)
			}
		}
	} else {
		filteredAPIRevisions = apiServiceInstances
	}

	return filteredAPIRevisions, nil
}

// GetAPIServiceInstances - get management.APIServiceInstance
func (c *ServiceClient) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*management.APIServiceInstance, error) {
	resources, err := c.GetAPIV1ResourceInstances(queryParams, URL)
	if err != nil {
		return nil, err
	}
	apiServiceIntances, err := management.APIServiceInstanceFromInstanceArray(resources)
	if err != nil {
		return nil, err
	}

	return apiServiceIntances, nil
}

// GetAPIV1ResourceInstances - return apiv1 Resource instance with the default page size.
// It first tries a single fetch at the known page size (no HEAD/count call). Only if that
// first page comes back full (suggesting there's more) or times out does it fall back to
// a HEAD count call plus concurrent, worker-pool-based fetching of the rest.
func (c *ServiceClient) GetAPIV1ResourceInstances(queryParams map[string]string, url string) ([]*apiv1.ResourceInstance, error) {
	startingSize := c.cfg.GetPageSize()
	if size, ok := c.getPageSize(url); ok {
		startingSize = size
	}

	// pageParams.page is 0-indexed throughout this package; GetAPIV1ResourceInstancesWithPageAndSize
	// converts to the 1-indexed page the API expects.
	first, err := c.getAPIV1ResourceInstancesWithPageSize(queryParams, url, pageParams{startingSize, 0, 0, 0})
	switch {
	case err == nil && len(first) < startingSize:
		// fewer than a full page came back - there's nothing left to fetch
		return first, nil
	case err == nil:
		// first page was full - there's likely more; find out how much and fetch the rest concurrently
		return c.getRemainingResourceInstancesConcurrently(queryParams, url, startingSize, 1, first)
	case strings.Contains(err.Error(), "context deadline exceeded"):
		// the first page itself timed out at startingSize - record that immediately so every
		// worker in the pool skips straight to the halved size via fetchPageParams's
		// shared-size check, instead of each independently re-discovering that startingSize is
		// too large through its own wasted timeout
		c.setPageSizeIfSmaller(url, startingSize/2)
		return c.getRemainingResourceInstancesConcurrently(queryParams, url, startingSize, 0, nil)
	default:
		return nil, err
	}
}

// getRemainingResourceInstancesConcurrently fetches pages [startPage, numPages) with a
// worker pool, using the total count from a HEAD request to know how many pages remain,
// and appends them to the already-fetched first (which may be nil).
func (c *ServiceClient) getRemainingResourceInstancesConcurrently(queryParams map[string]string, url string, pageSize, startPage int, first []*apiv1.ResourceInstance) ([]*apiv1.ResourceInstance, error) {
	count, err := c.GetAPIV1ResourceCount(url)
	if err != nil {
		return nil, err
	}

	numPages := (count + pageSize - 1) / pageSize
	if numPages <= startPage {
		return first, nil
	}

	rest, err := c.fetchPagesConcurrently(queryParams, url, pageSize, startPage, numPages)
	if err != nil {
		return nil, err
	}
	return append(first, rest...), nil
}

// fetchPagesConcurrently fetches pages [startPage, numPages) of pageSize using a pool of
// c.numberOfWorkers workers, each retrying its assigned page through reducePageParams on
// a context deadline rather than dropping it.
func (c *ServiceClient) fetchPagesConcurrently(queryParams map[string]string, url string, pageSize, startPage, numPages int) ([]*apiv1.ResourceInstance, error) {
	chans := make([]chan clientParams, c.numberOfWorkers)
	riChan := make(chan apiv1.ResourceInstance)
	errChan := make(chan error)

	var wg sync.WaitGroup
	for i := range chans {
		chans[i] = make(chan clientParams)
		wg.Add(1)
		go func(ch chan clientParams) {
			defer wg.Done()
			c.worker(ch, riChan, errChan)
		}(chans[i])
	}
	go func() {
		wg.Wait()
		close(riChan)
		close(errChan)
	}()

	go func() {
		for page := startPage; page < numPages; page++ {
			chans[page%c.numberOfWorkers] <- clientParams{
				queryParams: queryParams,
				url:         url,
				pageParams:  pageParams{pageSize, page, 0, 0},
			}
		}
		for _, ch := range chans {
			close(ch)
		}
	}()

	resourceInstances := make([]*apiv1.ResourceInstance, 0, (numPages-startPage)*pageSize)
	var errs []error
	for riChan != nil || errChan != nil {
		select {
		case ri, ok := <-riChan:
			if !ok {
				riChan = nil
				continue
			}
			resourceInstances = append(resourceInstances, &ri)
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			errs = append(errs, err)
		}
	}

	return resourceInstances, errors.Join(errs...)
}

// GetAPIV1ResourceCount issues a HEAD request and returns the total resource count
// from the X-Axway-total-count response header. Returns 0 if the header is absent.
func (c *ServiceClient) GetAPIV1ResourceCount(url string) (int, error) {
	if !strings.HasPrefix(url, c.cfg.GetAPIServerURL()) && !strings.HasPrefix(url, c.cfg.GetAPIServerVersionURL()) {
		url = c.createAPIServerURL(url)
	}

	response, err := c.executeAPI(http.MethodHead, url, nil, nil, nil)
	if err != nil {
		return 0, err
	}
	if response.Code != http.StatusOK {
		return 0, fmt.Errorf("HEAD %s returned %d", url, response.Code)
	}

	vals := response.Headers[http.CanonicalHeaderKey("X-Axway-total-count")]
	if len(vals) == 0 {
		return 0, nil
	}

	count, err := strconv.Atoi(vals[0])
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (c *ServiceClient) getPageSize(url string) (int, bool) {
	c.pageSizeMutex.Lock()
	defer c.pageSizeMutex.Unlock()
	size, ok := c.pageSizes[url]
	return size, ok
}

func (c *ServiceClient) setPageSizeIfSmaller(url string, size int) {
	c.pageSizeMutex.Lock()
	defer c.pageSizeMutex.Unlock()
	if known, ok := c.pageSizes[url]; !ok || size < known {
		c.pageSizes[url] = size
	}
}

func (c *ServiceClient) getAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, url string, pageParams pageParams) ([]*apiv1.ResourceInstance, error) {
	resourceInstances := make([]*apiv1.ResourceInstance, 0)
	if !strings.HasPrefix(url, c.cfg.GetAPIServerURL()) && !strings.HasPrefix(url, c.cfg.GetAPIServerVersionURL()) {
		url = c.createAPIServerURL(url)
	}

	log := c.logger.WithField("endpoint", url).WithField("pageSize", pageParams.pageSize).WithField("page", pageParams.page)
	log.Trace("retrieving all resources from endpoint")
	query := map[string]string{
		// pageParams.page is 0-indexed internally; the API expects a 1-indexed page
		"page":     strconv.Itoa(pageParams.page + 1),
		"pageSize": strconv.Itoa(pageParams.pageSize),
	}

	// Add query params for getting revisions for the service and use the latest one as last reference
	for key, value := range queryParams {
		query[key] = value
	}
	response, err := c.ExecuteAPI(coreapi.GET, url, query, nil)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(response, &resourceInstances); err != nil {
		log.WithError(err).Debug("error deserializing resource page response")
		return nil, err
	}

	from := min(pageParams.skipFirstN, len(resourceInstances))
	to := min(pageParams.pageSize-pageParams.skipLastN, len(resourceInstances))
	if to < from {
		return nil, nil
	}
	return resourceInstances[from:to], nil
}

type clientParams struct {
	queryParams map[string]string
	url         string
	pageParams  pageParams
}

type pageParams struct {
	pageSize   int
	page       int
	skipFirstN int
	skipLastN  int
}

const maxPageSizeRetries = 3

func (c *ServiceClient) worker(ch <-chan clientParams, chRI chan<- apiv1.ResourceInstance, chErr chan<- error) {
	for cp := range ch {
		c.fetchPageParams(cp.queryParams, cp.url, cp.pageParams, chRI, chErr, maxPageSizeRetries)
	}
}

// fetchPageParams fetches the range described by params, halving and recursing through
// reducePageParams on a context deadline until either it succeeds, retries run out, or
// pageSize would drop below minPageSize - at which point the range is given up on and the
// error reported.
//
// Before attempting the fetch, it checks whether another worker has already discovered a
// smaller working page size for this url; if so, it skips the doomed attempt at this
// already-known-too-large size and reduces immediately. Without this, every worker that
// times out around the same time (likely, since they're hitting the same overloaded
// endpoint) would independently rediscover the same halving sequence from scratch instead
// of converging on whatever size the fastest one already found.
func (c *ServiceClient) fetchPageParams(queryParams map[string]string, url string, params pageParams, chRI chan<- apiv1.ResourceInstance, chErr chan<- error, retries int) {
	log := c.logger.WithField("endpoint", url).WithField("pageSize", params.pageSize).WithField("page", params.page).
		WithField("skipFirstN", params.skipFirstN).WithField("skipLastN", params.skipLastN)

	if known, ok := c.getPageSize(url); ok && known < params.pageSize {
		log.Trace("a smaller page size was already discovered for this url, skipping straight to reducing")
		c.reduceAndRetry(queryParams, url, params, chRI, chErr, retries, nil)
		return
	}

	ris, err := c.getAPIV1ResourceInstancesWithPageSize(queryParams, url, params)
	if err == nil {
		for _, ri := range ris {
			chRI <- *ri
		}
		log.Trace("finished sending batch")
		return
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		log.WithError(err).Debug("error while retrieving resources")
		chErr <- err
		return
	}

	c.reduceAndRetry(queryParams, url, params, chRI, chErr, retries, err)
}

// reduceAndRetry halves params via reducePageParams and retries each resulting sub-range,
// as long as retries remain and the halved size wouldn't drop below minPageSize. origErr is
// reported on chErr when giving up; it may be nil when reducing preemptively (see
// fetchPageParams).
func (c *ServiceClient) reduceAndRetry(queryParams map[string]string, url string, params pageParams, chRI chan<- apiv1.ResourceInstance, chErr chan<- error, retries int, origErr error) {
	log := c.logger.WithField("endpoint", url).WithField("pageSize", params.pageSize).WithField("page", params.page)

	// a nil origErr means this range was reduced preemptively (fetchPageParams's shared-size
	// check) without ever attempting a real fetch; giving up still needs a non-nil error so
	// it isn't silently swallowed by errors.Join
	giveUp := func(reason string) {
		if origErr == nil {
			origErr = fmt.Errorf("giving up on page %d at size %d: %s", params.page, params.pageSize, reason)
		}
		log.WithError(origErr).Debug(reason)
		chErr <- origErr
	}

	if retries <= 0 {
		giveUp("retries exhausted")
		return
	}
	if params.pageSize/2 < minPageSize {
		giveUp("page size already at minimum")
		return
	}

	log.Trace("halving page size and retrying assigned range")
	c.setPageSizeIfSmaller(url, params.pageSize/2)
	for _, reduced := range reducePageParams(params) {
		c.fetchPageParams(queryParams, url, reduced, chRI, chErr, retries-1)
	}
}

// reducePageParams re-expresses the absolute item range described by p - using half
// the page size, so a worker that hit a context deadline at p.pageSize can retry the exact
// same range at a smaller size instead of dropping it or refetching everything from scratch.
//
// The halved grid doesn't necessarily align with the original range's boundaries, so the
// range can span up to 3 pages of the new size: a partial first page, an optional full page,
// and a partial last page. Each candidate page's skipLastN is derived by matching its end to
// the original range's end (not by naively halving p.skipLastN); that's what keeps this
// correct even when p.pageSize is odd, where pageSize/2 alone wouldn't tile evenly.
func reducePageParams(p pageParams) []pageParams {
	params := []pageParams{}

	// param1.page/skipFirstN place the range's absolute start onto the new (half-size) grid
	// via plain division/remainder - exact regardless of whether p.pageSize is even or odd.
	param1 := pageParams{}
	param1.pageSize = p.pageSize / 2
	param1.page = (p.pageSize*p.page + p.skipFirstN) / (p.pageSize / 2)
	param1.skipFirstN = (p.pageSize*p.page + p.skipFirstN) % (p.pageSize / 2)
	param1.skipLastN = param1.pageSize*(param1.page+1) + p.skipLastN - p.pageSize*(p.page+1)
	params = append(params, param1)
	// skipLastN >= 0 means this page's end already reaches (>0) or exactly matches (==0) the
	// original range's end, so it's the last page needed and nothing more has to be added.
	// A negative value is just an internal "hasn't reached the end yet" signal, not a literal
	// skip count - it gets clamped to 0 below once we know there's a next page after all.
	if param1.skipLastN >= 0 {
		return params
	}

	params[0].skipLastN = 0
	param2 := pageParams{}
	param2.pageSize = param1.pageSize
	param2.page = param1.page + 1
	param2.skipFirstN = 0
	param2.skipLastN = param2.pageSize*(param2.page+1) + p.skipLastN - p.pageSize*(p.page+1)
	params = append(params, param2)
	if param2.skipLastN >= 0 {
		return params
	}

	// still short of the end after 2 pages - this only happens when the original range spans
	// more than 2 half-sized pages - so add a third and final one.
	params[1].skipLastN = 0
	param3 := pageParams{}
	param3.pageSize = param1.pageSize
	param3.page = param2.page + 1
	param3.skipFirstN = 0
	param3.skipLastN = param3.pageSize*(param3.page+1) + p.skipLastN - p.pageSize*(p.page+1)
	return append(params, param3)
}
