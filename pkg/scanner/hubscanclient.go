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
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	hubScheme = "https"
)

// HubScanClient implements ScanClientInterface using
// the Black Duck hub and scan client programs.
type HubScanClient struct {
	host           string
	username       string
	password       string
	port           int
	scanClientInfo *ScanClientInfo
}

// NewHubScanClient requires hub login credentials
func NewHubScanClient(host string, username string, password string, port int, scanClientInfo *ScanClientInfo) (*HubScanClient, error) {
	hsc := HubScanClient{
		host:           host,
		username:       username,
		password:       password,
		port:           port,
		scanClientInfo: scanClientInfo}
	return &hsc, nil
}

func (hsc *HubScanClient) Scan(path string, projectName string, versionName string, scanName string) error {
	startTotal := time.Now()

	scanCliImplJarPath := hsc.scanClientInfo.scanCliImplJarPath()
	scanCliJarPath := hsc.scanClientInfo.scanCliJarPath()
	scanCliJavaPath := hsc.scanClientInfo.scanCliJavaPath() + "java"
	cmd := exec.Command(scanCliJavaPath,
		"-Xms512m",
		"-Xmx4096m",
		"-Dblackduck.scan.cli.benice=true",
		"-Dblackduck.scan.skipUpdate=true",
		"-Done-jar.silent=true",
		"-Done-jar.jar.path="+scanCliImplJarPath,
		"-jar", scanCliJarPath,
		"--host", hsc.host,
		"--port", fmt.Sprintf("%d", hsc.port),
		"--scheme", hubScheme,
		"--project", projectName,
		"--release", versionName,
		"--username", hsc.username,
		"--name", scanName,
		"--insecure",
		"-v",
		path)
	cmd.Env = append(cmd.Env, fmt.Sprintf("BD_HUB_PASSWORD=%s", hsc.password))

	log.Infof("running command %+v for path %s\n", cmd, path)
	startScanClient := time.Now()
	stdoutStderr, err := cmd.CombinedOutput()

	recordScanClientDuration(time.Now().Sub(startScanClient), err == nil)
	recordTotalScannerDuration(time.Now().Sub(startTotal), err == nil)

	if err != nil {
		recordScannerError("scan client failed")
		log.Errorf("java scanner failed for path %s with error %s and output:\n%s\n", path, err.Error(), string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed java scanner for path %s", path)
	log.Debugf("output from path %s: %s", path, stdoutStderr)
	return err
}
