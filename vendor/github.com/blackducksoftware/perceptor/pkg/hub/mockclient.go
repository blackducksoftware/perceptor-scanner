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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
)

// need: mock hub, ?mock apiserver?

// MockClient is a mock implementation of ClientInterface.
type MockClient struct {
	images  map[string]bool
	host    string
	stop    chan struct{}
	updates chan Update
}

// NewMockClient .....
func NewMockClient(host string) *MockClient {
	return &MockClient{
		images:  map[string]bool{},
		host:    host,
		stop:    make(chan struct{}),
		updates: make(chan Update),
	}
}

// func (hub *MockClient) startRandomScanFinishing() {
// 	fmt.Println("starting!")
// 	for {
// 		time.Sleep(3 * time.Second)
// 		// TODO should lock the hub
// 		length := len(hub.inProgressImages)
// 		fmt.Println("in progress -- [", strings.Join(hub.inProgressImages, ", "), "]")
// 		if length <= 0 {
// 			continue
// 		}
// 		index := rand.Intn(length)
// 		image := hub.inProgressImages[index]
// 		fmt.Println("something finished --", image)
// 		hub.inProgressImages = append(hub.inProgressImages[:index], hub.inProgressImages[index+1:]...)
// 		hub.finishedImages[image] = 1
// 	}
// }

// DeleteScan ...
func (hub *MockClient) DeleteScan(scanName string) {
	delete(hub.images, scanName)
}

// Version .....
func (hub *MockClient) Version() (string, error) {
	return "hub.hubVersion", nil
}

// SetTimeout ...
func (hub *MockClient) SetTimeout(timeout time.Duration) {
	//
}

// Model ...
func (hub *MockClient) Model() <-chan *api.ModelHub {
	ch := make(chan *api.ModelHub)
	codeLocations := map[string]*api.ModelCodeLocation{}
	for image := range hub.images {
		codeLocations[image] = &api.ModelCodeLocation{
			Href:                 "TODO",
			MappedProjectVersion: "TODO",
			UpdatedAt:            "TODO",
			URL:                  "TODO",
		}
	}
	model := &api.ModelHub{
		Errors:                    []string{},
		Status:                    "???",
		HasLoadedAllCodeLocations: true,
		CodeLocations:             codeLocations,
		CircuitBreaker:            &api.ModelCircuitBreaker{},
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		ch <- model
	}()
	return ch
}

// ResetCircuitBreaker ...
func (hub *MockClient) ResetCircuitBreaker() {
	//
}

// IsEnabled ...
func (hub *MockClient) IsEnabled() <-chan bool {
	return make(<-chan bool)
}

// Host ...
func (hub *MockClient) Host() string {
	return hub.host
}

// CodeLocationsCount ...
func (hub *MockClient) CodeLocationsCount() <-chan int {
	ch := make(chan int)
	go func() {
		time.Sleep(30 * time.Millisecond)
		ch <- len(hub.images)
	}()
	return ch
}

// StartScanClient ...
func (hub *MockClient) StartScanClient(scanName string) {
	hub.images[scanName] = false
}

// FinishScanClient ...
func (hub *MockClient) FinishScanClient(scanName string) {
	//
}

// InProgressScans ...
func (hub *MockClient) InProgressScans() <-chan []string {
	ch := make(chan []string)
	inProgress := []string{}
	for sha, isDone := range hub.images {
		if !isDone {
			inProgress = append(inProgress, sha)
		}
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		ch <- inProgress
	}()
	return ch
}

// ScanResults ...
func (hub *MockClient) ScanResults() <-chan map[string]*ScanResults {
	ch := make(chan map[string]*ScanResults)
	go func() {
		time.Sleep(30 * time.Millisecond)
		ch <- map[string]*ScanResults{}
	}()
	return ch
}

// Stop ...
func (hub *MockClient) Stop() {
	close(hub.stop)
}

// StopCh ...
func (hub *MockClient) StopCh() <-chan struct{} {
	return hub.stop
}

// Updates ...
func (hub *MockClient) Updates() <-chan Update {
	return hub.updates
}

// HasFetchedCodeLocations ...
func (hub *MockClient) HasFetchedCodeLocations() <-chan bool {
	ch := make(chan bool)
	go func() {
		time.Sleep(10 * time.Millisecond)
		ch <- true
	}()
	return ch
}

// CodeLocations ...
func (hub *MockClient) CodeLocations() <-chan map[string]bool {
	ch := make(chan map[string]bool)
	cls := map[string]bool{}
	for sha := range hub.images {
		cls[sha] = true
	}
	go func() {
		ch <- cls
	}()
	return ch
}
