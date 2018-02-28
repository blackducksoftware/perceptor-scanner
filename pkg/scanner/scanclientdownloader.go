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
	"os/exec"

	"github.com/blackducksoftware/hub-client-go/hubclient"
	log "github.com/sirupsen/logrus"
)

const (
	scanClientRootPath = "/tmp/scanner"
)

var scanClientTarGzPath = fmt.Sprintf("%s/scanclient.tar.gz", scanClientRootPath)

func downloadScanClient(hubHost string, hubUser string, hubPassword string) (*scanClientInfo, error) {
	// 1. instantiate hub client
	hubClient, err := hubclient.NewWithSession(baseURL, hubclient.HubClientDebugTimings)
	if err != nil {
		log.Errorf("unable to instantiate hub client: %s", err.Error())
		return nil, err
	}

	// 1a. log in to hub client
	err = hubClient.Login(hubUser, hubPassword)
	if err != nil {
		log.Errorf("unable to log in to hub: %s", err.Error())
		return nil, err
	}

	// 2. get hub version
	currentVersion, err := hubClient.CurrentVersion()
	if err != nil {
		log.Errorf("unable to get hub version: %s", err.Error())
		return nil, err
	}

	// 3. pull down scan client as .tar.gz
	err = hubClient.DownloadScanClientLinux(scanClientTarGzPath)
	if err != nil {
		log.Errorf("unable to download scan client: %s", err.Error())
		return nil, err
	}

	// 4. untar, unzip scan client using os call to tar
	cmd := exec.Command("tar", "-xvpf", scanClientTarGzPath, "-C", scanClientRootPath)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("unable to untar/unzip scan client: %s", stdoutStderr)
		return nil, err
	}

	// 5. we're done
	return &scanClientInfo{hubVersion: currentVersion.Version, scanClientRootPath: scanClientRootPath}, nil
}
