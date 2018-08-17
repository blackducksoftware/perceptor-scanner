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

package core

import (
	"fmt"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/hub"
	log "github.com/sirupsen/logrus"
)

// HubManagerInterface ...
type HubManagerInterface interface {
	SetHubs(hubURLs []string)
	HubClients() map[string]hub.ClientInterface
}

// HubManager ...
type HubManager struct {
	username    string
	password    string
	port        int
	httpTimeout time.Duration
	//
	stop <-chan struct{}
	//
	hubs map[string]hub.ClientInterface
}

// NewHubManager ...
func NewHubManager(username string, password string, port int, httpTimeout time.Duration, stop <-chan struct{}) *HubManager {
	return &HubManager{username: username, password: password, port: port, httpTimeout: httpTimeout, stop: stop, hubs: map[string]hub.ClientInterface{}}
}

// SetHubs ...
func (hm *HubManager) SetHubs(hubURLs []string) {
	newHubURLs := map[string]bool{}
	for _, hubURL := range hubURLs {
		newHubURLs[hubURL] = true
	}
	hubsToCreate := map[string]bool{}
	for hubURL := range newHubURLs {
		if _, ok := hm.hubs[hubURL]; !ok {
			hubsToCreate[hubURL] = true
		}
	}
	// 1. create new hubs
	// TODO handle retries and failures intelligently
	go func() {
		for hubURL := range hubsToCreate {
			err := hm.create(hubURL)
			if err != nil {
				log.Errorf("unable to create Hub client for %s: %s", hubURL, err.Error())
			}
		}
	}()
	// 2. delete removed hubs
	for hubURL, hub := range hm.hubs {
		if _, ok := newHubURLs[hubURL]; !ok {
			hub.Stop()
			delete(hm.hubs, hubURL)
		}
	}
}

// Create ...
func (hm *HubManager) create(hubURL string) error {
	if _, ok := hm.hubs[hubURL]; ok {
		return fmt.Errorf("cannot create hub %s: already exists", hubURL)
	}
	hubClient := hub.NewClient(hm.username, hm.password, hubURL, hm.port, hm.httpTimeout, 999999*time.Hour)
	hm.hubs[hubURL] = hubClient
	return nil
}

// HubClients ...
func (hm *HubManager) HubClients() map[string]hub.ClientInterface {
	return hm.hubs
}

// MockHubCreater ...
type MockHubCreater struct{}

// SetHubs ...
func (mhc *MockHubCreater) SetHubs(hubURLs []string) {
	// TODO
}

// HubClients ...
func (mhc *MockHubCreater) HubClients() map[string]hub.ClientInterface {
	return nil
}
