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
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

// HubScanClient implements ScanClientInterface using
// the Black Duck hub and scan client programs.
type HubScanClient struct {
	host        string
	username    string
	password    string
	imagePuller ImagePullerInterface
}

// NewHubScanClient requires login credentials in order to instantiate
// a HubScanClient.
func NewHubScanClient(host string, username string, password string, imagePuller ImagePullerInterface) (*HubScanClient, error) {
	hsc := HubScanClient{
		host:        host,
		username:    username,
		password:    password,
		imagePuller: imagePuller}
	return &hsc, nil
}

func mapKeys(m map[string]ScanJob) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func (hsc *HubScanClient) Scan(job ScanJob) error {
	startTotal := time.Now()
	image := job.image()
	err := hsc.imagePuller.PullImage(image)

	defer cleanUpTarFile(image.DockerTarFilePath())

	if err != nil {
		// TODO do we even need to add this metric?  since the errors should already
		//   have been reported in the docker package
		// recordScannerError("docker image pull and tar file creation", errorName)
		log.Errorf("unable to pull docker image %s: %s", job.PullSpec, err.Error())
		return err
	}

	// TODO coupla problems here:
	//   1. hardcoded path
	//   2. hardcoded version number
	scanCliImplJarPath := "./dependencies/scan.cli-4.3.0/lib/cache/scan.cli.impl-standalone.jar"
	scanCliJarPath := "./dependencies/scan.cli-4.3.0/lib/scan.cli-4.3.0-standalone.jar"
	path := image.DockerTarFilePath()
	cmd := exec.Command("java",
		"-Xms512m",
		"-Xmx4096m",
		"-Dblackduck.scan.cli.benice=true",
		"-Dblackduck.scan.skipUpdate=true",
		"-Done-jar.silent=true",
		"-Done-jar.jar.path="+scanCliImplJarPath,
		"-jar", scanCliJarPath,
		"--host", hsc.host,
		"--port", "443", // "--port", "8443", // TODO or should this be 8080 or something else? or should we just leave it off and let it default?
		"--scheme", "https", // TODO or should this be http?
		"--project", job.HubProjectName,
		"--release", job.HubProjectVersionName,
		"--username", hsc.username,
		"--name", job.HubScanName,
		"--insecure", // TODO not sure about this
		"-v",
		path)

	log.Infof("running command %+v for image %s\n", cmd, job.Sha)
	startScanClient := time.Now()
	stdoutStderr, err := cmd.CombinedOutput()

	recordScanClientDuration(time.Now().Sub(startScanClient), err == nil)
	recordTotalScannerDuration(time.Now().Sub(startTotal), err == nil)

	if err != nil {
		recordScannerError("scan client failed")
		log.Errorf("java scanner failed for image %s with error %s and output:\n%s\n", job.Sha, err.Error(), string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed java scanner for image %s: %s", job.Sha, stdoutStderr)
	return err
}

// func (hsc *HubScanClient) ScanCliSh(job ScanJob) error {
// 	pathToScanner := "./dependencies/scan.cli-4.3.0/bin/scan.cli.sh"
// 	cmd := exec.Command(pathToScanner,
// 		"--project", job.Image.HubProjectName(),
// 		"--host", hsc.host,
// 		"--port", "443",
// 		"--insecure",
// 		"--username", hsc.username,
// 		job.Image.HumanReadableName())
// 	log.Infof("running command %v for image %s\n", cmd, job.Image.HumanReadableName())
// 	stdoutStderr, err := cmd.CombinedOutput()
// 	if err != nil {
// 		message := fmt.Sprintf("failed to run scan.cli.sh: %s", err.Error())
// 		log.Error(message)
// 		log.Errorf("output from scan.cli.sh:\n%v\n", string(stdoutStderr))
// 		return err
// 	}
// 	log.Infof("successfully completed scan.cli.sh: %s", stdoutStderr)
// 	return nil
// }
//
// func (hsc *HubScanClient) ScanDockerSh(job ScanJob) error {
// 	pathToScanner := "./dependencies/scan.cli-4.3.0/bin/scan.docker.sh"
// 	cmd := exec.Command(pathToScanner,
// 		"--image", job.Image.ShaName(),
// 		"--name", job.Image.HumanReadableName(),
// 		"--release", job.Image.HubVersionName(),
// 		"--project", job.Image.HubProjectName(),
// 		"--host", hsc.host,
// 		"--username", hsc.username)
// 	log.Infof("running command %v for image %s\n", cmd, job.Image.HumanReadableName())
// 	stdoutStderr, err := cmd.CombinedOutput()
// 	if err != nil {
// 		message := fmt.Sprintf("failed to run scan.docker.sh: %s", err.Error())
// 		log.Error(message)
// 		log.Errorf("output from scan.docker.sh:\n%v\n", string(stdoutStderr))
// 		return err
// 	}
// 	log.Infof("successfully completed ./scan.docker.sh: %s", stdoutStderr)
// 	return nil
// }

func cleanUpTarFile(path string) {
	err := os.Remove(path)
	recordCleanUpTarFile(err == nil)
	if err != nil {
		log.Errorf("unable to remove file %s: %s", path, err.Error())
	} else {
		log.Infof("successfully cleaned up file %s", path)
	}
}
