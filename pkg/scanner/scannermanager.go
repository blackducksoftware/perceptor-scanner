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

package scanner

import (
	"fmt"
	"os"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

const (
	requestScanJobPause = 20 * time.Second
	imageFacadeBaseURL  = "http://localhost"
)

type ScannerManager struct {
	scanner         *Scanner
	perceptorClient *PerceptorClient
}

func NewScannerManager(config *Config) (*ScannerManager, error) {
	log.Infof("instantiating ScannerManager with config %+v", config)

	hubPassword, ok := os.LookupEnv(config.HubUserPasswordEnvVar)
	if !ok {
		return nil, fmt.Errorf("unable to get Hub password: environment variable %s not set", config.HubUserPasswordEnvVar)
	}

	cliRootPath := "/tmp/scanner"
	scanClientInfo, err := DownloadScanClient(
		OSTypeLinux,
		cliRootPath,
		config.HubHost,
		config.HubUser,
		hubPassword,
		config.HubPort,
		time.Duration(config.HubClientTimeoutSeconds)*time.Second)
	if err != nil {
		log.Errorf("unable to download scan client: %s", err.Error())
		return nil, err
	}

	log.Infof("instantiating scanner with hub %s, user %s", config.HubHost, config.HubUser)

	imagePuller := NewImageFacadePuller(imageFacadeBaseURL, config.ImageFacadePort)
	scanClient, err := NewHubScanClient(
		config.HubHost,
		config.HubUser,
		hubPassword,
		config.HubPort,
		scanClientInfo)
	if err != nil {
		log.Errorf("unable to instantiate hub scan client: %s", err.Error())
		return nil, err
	}

	scanner := NewScanner(imagePuller, scanClient, config.ImageDirectory)

	pc := NewPerceptorClient(config.PerceptorHost, config.PerceptorPort)
	scannerManager := ScannerManager{
		scanner:         scanner,
		perceptorClient: pc}

	go func() {
		for {
			select {
			case action := <-scanner.shouldScanLayer:
				response, err := pc.GetShouldScanLayer(action.request)
				if err != nil {
					action.err <- err
				} else {
					action.done <- response.ShouldScan
				}
			case imageLayers := <-scanner.imageLayers:
				imageLayers.done <- pc.PostImageLayers(imageLayers.layers)
			}
		}
	}()

	return &scannerManager, nil
}

// StartRequestingScanJobs will start asking for work
func (sm *ScannerManager) StartRequestingScanJobs() {
	log.Infof("starting to request scan jobs")
	go func() {
		for {
			time.Sleep(requestScanJobPause)
			sm.requestAndRunScanJob()
		}
	}()
}

func (sm *ScannerManager) requestAndRunScanJob() {
	log.Debug("requesting scan job")
	nextImage, err := sm.perceptorClient.GetNextImage()
	if err != nil {
		log.Errorf("unable to request scan job: %s", err.Error())
		return
	}
	if nextImage == nil {
		log.Debug("requested scan job, got nil")
		return
	}

	log.Infof("processing scan job %+v", nextImage)

	err = sm.scanner.ScanLayersInDockerSaveTarFile(nextImage.ImageSpec)
	errorString := ""
	if err != nil {
		log.Errorf("scan error: %s", err.Error())
		errorString = err.Error()
	}

	finishedJob := api.FinishedScanClientJob{Err: errorString, ImageSpec: *nextImage.ImageSpec}
	log.Infof("about to finish job, going to send over %+v", finishedJob)
	sm.perceptorClient.PostFinishedScan(&finishedJob)
	if err != nil {
		log.Errorf("unable to finish scan job: %s", err.Error())
	}
}
