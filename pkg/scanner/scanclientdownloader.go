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

	"github.com/blackducksoftware/hub-client-go/hubclient"
	log "github.com/sirupsen/logrus"
)

func DownloadScanClient(cliRootPath string, hubHost string, hubUser string, hubPassword string, hubPort int, timeout time.Duration) (*scanClientInfo, error) {
	// 1. instantiate hub client
	hubBaseURL := fmt.Sprintf("https://%s:%d", hubHost, hubPort)
	hubClient, err := hubclient.NewWithSession(hubBaseURL, hubclient.HubClientDebugTimings, timeout)
	if err != nil {
		log.Errorf("unable to instantiate hub client: %s", err.Error())
		return nil, err
	}

	log.Infof("successfully instantiated hub client %s", hubBaseURL)

	// 2. log in to hub client
	err = hubClient.Login(hubUser, hubPassword)
	if err != nil {
		log.Errorf("unable to log in to hub: %s", err.Error())
		return nil, fmt.Errorf("unable to log in to hub")
	}

	log.Info("successfully logged in to hub")

	// 3. get hub version
	currentVersion, err := hubClient.CurrentVersion()
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return nil, err
	}

	log.Infof("got hub version: %s", currentVersion.Version)

	cliInfo := &scanClientInfo{hubVersion: currentVersion.Version, scanClientRootPath: cliRootPath}

	// 4. create directory
	err = os.MkdirAll(cliInfo.scanClientRootPath, 0755)
	if err != nil {
		log.Errorf("unable to make dir %s: %s", cliInfo.scanClientRootPath, err.Error())
		return nil, err
	}

	// 5. pull down scan client as .zip
	err = hubClient.DownloadScanClientLinux(cliInfo.scanCliZipPath())
	if err != nil {
		log.Errorf("unable to download scan client: %s", err.Error())
		return nil, err
	}

	log.Infof("successfully downloaded scan client to %s", cliInfo.scanCliZipPath())

	// 6. unzip scan client
	err = unzip(cliInfo.scanCliZipPath(), cliInfo.scanClientRootPath)
	if err != nil {
		log.Errorf("unable to unzip %s: %s", cliInfo.scanCliZipPath(), err.Error())
		return nil, err
	}
	log.Infof("successfully unzipped from %s to %s", cliInfo.scanCliZipPath(), cliInfo.scanClientRootPath)

	// 7. we're done
	return cliInfo, nil
}
