/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package hub

import (
	"fmt"
	"time"

	"github.com/blackducksoftware/hub-client-go/hubapi"
	"github.com/blackducksoftware/hub-client-go/hubclient"
	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	maxHubExponentialBackoffDuration = 1 * time.Hour
	// hubDeleteTimeout                 = 1 * time.Hour
)

// Client .....
type Client struct {
	// TODO add a second hub client -- so that there's one for rare, slow requests (all projects,
	//   all code locations) and one for frequent, quick requests
	client         RawClientInterface
	circuitBreaker *CircuitBreaker
	// basic hub info
	username string
	password string
	host     string
	port     int
	status   ClientStatus
	// data
	codeLocations   map[string]hubapi.CodeLocation
	errors          []error
	inProgressScans map[string]bool
	// TODO critical vulnerabilities
	// timers
	loginTimer                   *util.Timer
	fetchAllCodeLocationsTimer   *util.Timer
	checkScansForCompletionTimer *util.Timer
	// channels
	stop                    chan struct{}
	resetCircuitBreakerCh   chan struct{}
	getModel                chan chan *api.ModelHub
	getCodeLocationsCh      chan chan map[string]hubapi.CodeLocation
	deleteScanCh            chan string
	didDeleteScanCh         chan *Result
	didLoginCh              chan error
	startScanClientCh       chan string
	finishScanClientCh      chan string
	scanDidFinishCh         chan *ScanDidFinish
	getCodeLocationsCountCh chan chan int
	getInProgressScansCh    chan chan []string
	didFetchCodeLocationsCh chan *Result
}

// NewClient returns a new Client.  It will not be logged in.
func NewClient(username string, password string, host string, port int, hubClientTimeout time.Duration, fetchAllProjectsPause time.Duration) *Client {
	hub := &Client{
		circuitBreaker: NewCircuitBreaker(maxHubExponentialBackoffDuration),
		username:       username,
		password:       password,
		host:           host,
		port:           port,
		status:         ClientStatusDown,
		//
		codeLocations:   map[string]hubapi.CodeLocation{},
		errors:          []error{},
		inProgressScans: map[string]bool{},
		//
		stop: make(chan struct{}),
		resetCircuitBreakerCh:   make(chan struct{}),
		getModel:                make(chan chan *api.ModelHub),
		getCodeLocationsCh:      make(chan chan map[string]hubapi.CodeLocation),
		deleteScanCh:            make(chan string),
		didDeleteScanCh:         make(chan *Result),
		didLoginCh:              make(chan error),
		startScanClientCh:       make(chan string),
		finishScanClientCh:      make(chan string),
		scanDidFinishCh:         make(chan *ScanDidFinish),
		getCodeLocationsCountCh: make(chan chan int),
		getInProgressScansCh:    make(chan chan []string),
		didFetchCodeLocationsCh: make(chan *Result)}
	// initialize hub client
	baseURL := fmt.Sprintf("https://%s:%d", host, port)
	client, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings, hubClientTimeout)
	if err != nil {
		hub.status = ClientStatusError
	}
	hub.client = client
	// action processing
	go func() {
		for {
			select {
			case <-hub.stop:
				return
			case <-hub.resetCircuitBreakerCh:
				hub.circuitBreaker.Reset()
			case ch := <-hub.getModel:
				ch <- hub.apiModel()
			case ch := <-hub.getCodeLocationsCh:
				ch <- hub.codeLocations
			case scanName := <-hub.deleteScanCh:
				hub.recordError(hub.deleteScanAndProjectVersion(scanName))
			case result := <-hub.didDeleteScanCh:
				hub.recordError(result.Err)
				if result.Err == nil {
					scanName := result.Value.(string)
					delete(hub.codeLocations, scanName)
				}
			case result := <-hub.didFetchCodeLocationsCh:
				hub.recordError(result.Err)
				if result.Err == nil {
					hub.codeLocations = result.Value.(map[string]hubapi.CodeLocation)
				}
			case scanName := <-hub.startScanClientCh:
				// anything to do?
				log.Debugf("nothing to do: received scan client start for scan %s", scanName)
			case scanName := <-hub.finishScanClientCh:
				hub.inProgressScans[scanName] = true
			case s := <-hub.scanDidFinishCh:
				delete(hub.inProgressScans, s.ScanName)
				hub.scanDidFinishCh <- s
			case get := <-hub.getCodeLocationsCountCh:
				get <- len(hub.codeLocations)
			case get := <-hub.getInProgressScansCh:
				scans := []string{}
				for scanName := range hub.inProgressScans {
					scans = append(scans, scanName)
				}
				get <- scans
			case err := <-hub.didLoginCh:
				hub.recordError(err)
				if err != nil && hub.status == ClientStatusUp {
					hub.status = ClientStatusDown
					hub.recordError(hub.checkScansForCompletionTimer.Pause())
					hub.recordError(hub.fetchAllCodeLocationsTimer.Pause())
				} else if err == nil && hub.status == ClientStatusDown {
					hub.status = ClientStatusUp
					hub.recordError(hub.checkScansForCompletionTimer.Resume(true))
					hub.recordError(hub.fetchAllCodeLocationsTimer.Resume(true))
				}
			}
		}
	}()
	hub.checkScansForCompletionTimer = hub.startCheckScansForCompletionTimer()
	hub.fetchAllCodeLocationsTimer = hub.startFetchAllCodeLocationsTimer(fetchAllProjectsPause)
	hub.loginTimer = hub.startLoginTimer()
	return hub
}

// Stop ...
func (hub *Client) Stop() {
	close(hub.stop)
}

// Host ...
func (hub *Client) Host() string {
	return hub.host
}

// ResetCircuitBreaker ...
func (hub *Client) ResetCircuitBreaker() {
	hub.resetCircuitBreakerCh <- struct{}{}
}

// Model ...
func (hub *Client) Model() *api.ModelHub {
	ch := make(chan *api.ModelHub)
	hub.getModel <- ch
	return <-ch
}

// Private methods

func (hub *Client) recordError(err error) {
	if err != nil {
		hub.errors = append(hub.errors, err)
	}
	if len(hub.errors) > 1000 {
		hub.errors = hub.errors[500:]
	}
}

// login ignores the circuit breaker, just in case the circuit breaker
// is closed because the calls were failing due to being unauthenticated.
// Or maybe TODO we need to distinguish between different types of
// request failure (network vs. 400 vs. 500 etc.)
// TODO could reset circuit breaker on success
func (hub *Client) login() error {
	start := time.Now()
	err := hub.client.Login(hub.username, hub.password)
	recordHubResponse("login", err == nil)
	recordHubResponseTime("login", time.Now().Sub(start))
	return err
}

func (hub *Client) apiModel() *api.ModelHub {
	errors := make([]string, len(hub.errors))
	for ix, err := range hub.errors {
		errors[ix] = err.Error()
	}
	codeLocations := map[string]*api.ModelCodeLocation{}
	for name, cl := range hub.codeLocations {
		codeLocations[name] = &api.ModelCodeLocation{
			Href:                 cl.Meta.Href,
			URL:                  cl.URL,
			MappedProjectVersion: cl.MappedProjectVersion,
			UpdatedAt:            cl.UpdatedAt,
		}
	}
	return &api.ModelHub{
		Errors:         errors,
		Status:         hub.status.String(),
		IsLoggedIn:     false, // TODO
		CodeLocations:  codeLocations,
		CircuitBreaker: hub.circuitBreaker.Model(),
	}
}

// Regular jobs

func (hub *Client) startLoginTimer() *util.Timer {
	pause := 30 * time.Second // Minute
	name := fmt.Sprintf("login-%s", hub.host)
	return util.NewRunningTimer(name, pause, hub.stop, true, func() {
		log.Debugf("starting to login to hub")
		err := hub.login()
		select {
		case hub.didLoginCh <- err:
		case <-hub.stop:
		}
	})
}

func (hub *Client) startFetchAllCodeLocationsTimer(pause time.Duration) *util.Timer {
	name := fmt.Sprintf("fetchCodeLocations-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to fetch all code locations")
		result := hub.fetchAllCodeLocations()
		select {
		case hub.didFetchCodeLocationsCh <- result:
		case <-hub.stop: // TODO should cancel when this happens
		}
	})
}

func (hub *Client) startCheckScansForCompletionTimer() *util.Timer {
	pause := 1 * time.Minute
	name := fmt.Sprintf("checkScansForCompletion-%s", hub.host)
	return util.NewTimer(name, pause, hub.stop, func() {
		log.Debugf("starting to check scans for completion")
		var scanNames []string
		select {
		case scanNames = <-hub.InProgressScans():
		case <-hub.stop:
			return
		}
		for _, scanName := range scanNames {
			scanResults, err := hub.fetchScan(scanName)
			if err != nil {
				log.Errorf("unable to fetch scan %s: %s", scanName, err.Error())
				continue
			}
			switch scanResults.ScanSummaryStatus() {
			case ScanSummaryStatusInProgress:
				// nothing to do
			case ScanSummaryStatusFailure, ScanSummaryStatusSuccess:
				select {
				case hub.scanDidFinishCh <- &ScanDidFinish{ScanName: scanName, ScanResults: scanResults}:
				case <-hub.stop:
					return
				}
			}
		}
	})
}

// Hub api calls

func (hub *Client) fetchAllCodeLocations() *Result {
	codeLocationList, err := hub.listAllCodeLocations()
	if err != nil {
		return &Result{Value: nil, Err: err}
	}
	log.Debugf("fetched all code locations: found %d, expected %d", len(codeLocationList.Items), codeLocationList.TotalCount)
	cls := map[string]hubapi.CodeLocation{}
	for _, cl := range codeLocationList.Items {
		cls[cl.Name] = cl
	}
	return &Result{Value: cls, Err: nil}
}

// IsEnabled returns whether the fetcher is currently enabled
// example: the circuit breaker is disabled -> the fetcher is disabled
// func (hub *Client) IsEnabled() <-chan bool {
// 	return hub.circuitBreaker.IsEnabledChannel
// }

// Version fetches the hub version
func (hub *Client) Version() (string, error) {
	start := time.Now()
	currentVersion, err := hub.client.CurrentVersion()
	recordHubResponse("version", err == nil)
	recordHubResponseTime("version", time.Now().Sub(start))
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return "", err
	}

	log.Infof("successfully got hub version %s", currentVersion.Version)
	return currentVersion.Version, nil
}

// SetTimeout is currently not concurrent-safe, and should be made so TODO
func (hub *Client) SetTimeout(timeout time.Duration) {
	hub.client.SetTimeout(timeout)
}

// DeleteScan deletes the code location and project version (but NOT the project)
// associated with the given scan name.
func (hub *Client) DeleteScan(scanName string) {
	hub.deleteScanCh <- scanName
}

func (hub *Client) deleteScanAndProjectVersion(scanName string) error {
	cl, ok := hub.codeLocations[scanName]
	if !ok {
		return fmt.Errorf("unable to delete scan %s, not found", scanName)
	}
	clURL := cl.Meta.Href
	projectVersionURL := cl.MappedProjectVersion
	finish := func(err error) {
		select {
		case hub.didDeleteScanCh <- &Result{Value: scanName, Err: err}:
		case <-hub.stop:
		}
	}
	go func() {
		err := hub.deleteCodeLocation(clURL)
		if err != nil {
			finish(err)
			return
		}
		finish(hub.deleteProjectVersion(projectVersionURL))
	}()
	return nil
}

// StartScanClient ...
func (hub *Client) StartScanClient(scanName string) {
	hub.startScanClientCh <- scanName
}

// FinishScanClient ...
func (hub *Client) FinishScanClient(scanName string) {
	hub.finishScanClientCh <- scanName
}

// ScanDidFinish ...
func (hub *Client) ScanDidFinish() <-chan *ScanDidFinish {
	return hub.scanDidFinishCh
}

// CodeLocationsCount ...
func (hub *Client) CodeLocationsCount() <-chan int {
	ch := make(chan int)
	hub.getCodeLocationsCountCh <- ch
	return ch
}

// InProgressScans ...
func (hub *Client) InProgressScans() <-chan []string {
	ch := make(chan []string)
	hub.getInProgressScansCh <- ch
	return ch
}

// FetchScan finds ScanResults by starting from a code location,
// and following links from there.
// It returns:
//  - nil, if there's no code location with a matching name
//  - nil, if there's 0 scan summaries for the code location
//  - an error, if there were any HTTP problems or link problems
//  - an ScanResults, but possibly with garbage data, in all other cases
// Weird cases to watch out for:
//  - multiple code locations with a matching name
//  - multiple scan summaries for a code location
//  - zero scan summaries for a code location
func (hub *Client) fetchScan(scanNameSearchString string) (*ScanResults, error) {
	codeLocationList, err := hub.listCodeLocations(scanNameSearchString)

	if err != nil {
		log.Errorf("error fetching code location list: %v", err)
		return nil, err
	}
	codeLocations := codeLocationList.Items
	switch len(codeLocations) {
	case 0:
		recordHubData("codeLocations", true)
		return nil, nil
	case 1:
		recordHubData("codeLocations", true) // good to go
	default:
		recordHubData("codeLocations", false)
		log.Warnf("expected 1 code location matching name search string %s, found %d", scanNameSearchString, len(codeLocations))
	}

	codeLocation := codeLocations[0]
	return hub.fetchScanResultsUsingCodeLocation(codeLocation, scanNameSearchString)
}

func (hub *Client) fetchScanResultsUsingCodeLocation(codeLocation hubapi.CodeLocation, scanNameSearchString string) (*ScanResults, error) {
	versionLink, err := codeLocation.GetProjectVersionLink()
	if err != nil {
		log.Errorf("unable to get project version link: %s", err.Error())
		return nil, err
	}

	version, err := hub.getProjectVersion(*versionLink)
	if err != nil {
		log.Errorf("unable to fetch project version: %s", err.Error())
		return nil, err
	}

	riskProfileLink, err := version.GetProjectVersionRiskProfileLink()
	if err != nil {
		log.Errorf("error getting risk profile link: %v", err)
		return nil, err
	}

	riskProfile, err := hub.getProjectVersionRiskProfile(*riskProfileLink)
	if err != nil {
		log.Errorf("error fetching project version risk profile: %v", err)
		return nil, err
	}

	policyStatusLink, err := version.GetProjectVersionPolicyStatusLink()
	if err != nil {
		log.Errorf("error getting policy status link: %v", err)
		return nil, err
	}
	policyStatus, err := hub.getProjectVersionPolicyStatus(*policyStatusLink)
	if err != nil {
		log.Errorf("error fetching project version policy status: %v", err)
		return nil, err
	}

	componentsLink, err := version.GetComponentsLink()
	if err != nil {
		log.Errorf("error getting components link: %v", err)
		return nil, err
	}

	scanSummariesLink, err := codeLocation.GetScanSummariesLink()
	if err != nil {
		log.Errorf("error getting scan summaries link: %v", err)
		return nil, err
	}
	scanSummariesList, err := hub.listScanSummaries(*scanSummariesLink)
	if err != nil {
		log.Errorf("error fetching scan summaries: %v", err)
		return nil, err
	}

	switch len(scanSummariesList.Items) {
	case 0:
		recordHubData("scan summaries", true)
		return nil, nil
	case 1:
		recordHubData("scan summaries", true) // good to go, continue
	default:
		recordHubData("scan summaries", false)
		log.Warnf("expected to find one scan summary for code location %s, found %d", scanNameSearchString, len(scanSummariesList.Items))
	}

	mappedRiskProfile, err := newRiskProfile(riskProfile.BomLastUpdatedAt, riskProfile.Categories)
	if err != nil {
		return nil, err
	}

	mappedPolicyStatus, err := newPolicyStatus(policyStatus.OverallStatus, policyStatus.UpdatedAt, policyStatus.ComponentVersionStatusCounts)
	if err != nil {
		return nil, err
	}

	scanSummaries := make([]ScanSummary, len(scanSummariesList.Items))
	for i, scanSummary := range scanSummariesList.Items {
		scanSummaries[i] = *NewScanSummaryFromHub(scanSummary)
	}

	scan := ScanResults{
		RiskProfile:           *mappedRiskProfile,
		PolicyStatus:          *mappedPolicyStatus,
		ComponentsHref:        componentsLink.Href,
		ScanSummaries:         scanSummaries,
		CodeLocationCreatedAt: codeLocation.CreatedAt,
		CodeLocationName:      codeLocation.Name,
		CodeLocationType:      codeLocation.Type,
		CodeLocationURL:       codeLocation.URL,
		CodeLocationUpdatedAt: codeLocation.UpdatedAt,
	}

	return &scan, nil
}

// "Raw" API calls

// ListAllProjects pulls in all projects in a single API call.
func (hub *Client) listAllProjects() (*hubapi.ProjectList, error) {
	var list *hubapi.ProjectList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("allProjects", func() error {
		limit := 2000000
		list, fetchError = hub.client.ListProjects(&hubapi.GetListOptions{Limit: &limit})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListAllCodeLocations pulls in all code locations in a single API call.
func (hub *Client) listAllCodeLocations() (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("allCodeLocations", func() error {
		limit := 2000000
		list, fetchError = hub.client.ListAllCodeLocations(&hubapi.GetListOptions{Limit: &limit})
		if fetchError != nil {
			log.Errorf("fetch error: %s", fetchError.Error())
		}
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// ListCodeLocations ...
func (hub *Client) listCodeLocations(codeLocationName string) (*hubapi.CodeLocationList, error) {
	var list *hubapi.CodeLocationList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("codeLocations", func() error {
		queryString := fmt.Sprintf("name:%s", codeLocationName)
		list, fetchError = hub.client.ListAllCodeLocations(&hubapi.GetListOptions{Q: &queryString})
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return list, fetchError
}

// GetProjectVersion ...
func (hub *Client) getProjectVersion(link hubapi.ResourceLink) (*hubapi.ProjectVersion, error) {
	var pv *hubapi.ProjectVersion
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersion", func() error {
		pv, fetchError = hub.client.GetProjectVersion(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return pv, fetchError
}

// GetProject ...
func (hub *Client) getProject(link hubapi.ResourceLink) (*hubapi.Project, error) {
	var val *hubapi.Project
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("project", func() error {
		val, fetchError = hub.client.GetProject(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionRiskProfile ...
func (hub *Client) getProjectVersionRiskProfile(link hubapi.ResourceLink) (*hubapi.ProjectVersionRiskProfile, error) {
	var val *hubapi.ProjectVersionRiskProfile
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersionRiskProfile", func() error {
		val, fetchError = hub.client.GetProjectVersionRiskProfile(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// GetProjectVersionPolicyStatus ...
func (hub *Client) getProjectVersionPolicyStatus(link hubapi.ResourceLink) (*hubapi.ProjectVersionPolicyStatus, error) {
	var val *hubapi.ProjectVersionPolicyStatus
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("projectVersionPolicyStatus", func() error {
		val, fetchError = hub.client.GetProjectVersionPolicyStatus(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// ListScanSummaries ...
func (hub *Client) listScanSummaries(link hubapi.ResourceLink) (*hubapi.ScanSummaryList, error) {
	var val *hubapi.ScanSummaryList
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("scanSummaries", func() error {
		val, fetchError = hub.client.ListScanSummaries(link)
		return fetchError
	})
	if err != nil {
		return nil, err
	}
	return val, fetchError
}

// DeleteProjectVersion ...
func (hub *Client) deleteProjectVersion(projectVersionHRef string) error {
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("deleteVersion", func() error {
		fetchError = hub.client.DeleteProjectVersion(projectVersionHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}

// DeleteCodeLocation ...
func (hub *Client) deleteCodeLocation(codeLocationHRef string) error {
	var fetchError error
	err := hub.circuitBreaker.IssueRequest("deleteCodeLocation", func() error {
		fetchError = hub.client.DeleteCodeLocation(codeLocationHRef)
		return fetchError
	})
	if err != nil {
		return err
	}
	return fetchError
}
