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
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/blackducksoftware/perceptor/pkg/api"
	log "github.com/sirupsen/logrus"
)

// TODO eventually, this will need to check whether layers have been scanned
// before scanning them.  But for now, let's just scan everything.

// Scanner ......
type Scanner struct {
	imagePuller ImagePullerInterface
	scanClient  ScanClientInterface
}

// NewScanner .....
func NewScanner(imagePuller ImagePullerInterface, scanClient ScanClientInterface) *Scanner {
	return &Scanner{imagePuller: imagePuller, scanClient: scanClient}
}

// ScanFullDockerImage is the 2.0 functionality
func (scanner *Scanner) ScanFullDockerImage(apiImage *api.ImageSpec) error {
	image := &common.Image{PullSpec: apiImage.PullSpec}
	err := scanner.imagePuller.PullImage(image)
	if err != nil {
		return err
	}
	defer cleanUpFile(image.DockerTarFilePath())
	return scanner.scanClient.Scan(image.DockerTarFilePath(), apiImage.HubProjectName, apiImage.HubProjectVersionName, apiImage.HubScanName)
}

// ScanLayersInDockerSaveTarFile avoids repeatedly scanning the same layers
func (scanner *Scanner) ScanLayersInDockerSaveTarFile(apiImage *api.ImageSpec) error {
	image := &common.Image{PullSpec: apiImage.PullSpec}
	// 1. pull image
	log.Debugf("about to pull %s to %s", image.DockerPullSpec(), image.DockerTarFilePath())
	err := scanner.imagePuller.PullImage(image)
	if err != nil {
		return err
	}
	log.Debugf("successfully pulled %s to %s", image.DockerPullSpec(), image.DockerTarFilePath())
	defer cleanUpFile(image.DockerTarFilePath())
	// 2. extract full image
	extractedDir := "/var/images/extracted/" + strings.Replace(image.PullSpec, "/", "_", -1)
	log.Debugf("about to extract %s to %s", image.DockerTarFilePath(), extractedDir)
	err = extractTarFile(image.DockerTarFilePath(), extractedDir)
	if err != nil {
		return err
	}
	defer cleanUpDir(extractedDir)
	log.Debugf("successfully extracted %s to %s", image.DockerTarFilePath(), extractedDir)
	// 3. read manifest.json to find the layers and calculate the hashes
	log.Debugf("about to build layer hashes from %s", extractedDir)
	shaToFilename, err := buildLayerHashes(extractedDir)
	if err != nil {
		return err
	}
	log.Debugf("successfully built layer hashes from %s", extractedDir)
	// 4. TODO check whether the layers need to be scanned

	// 5. scan the layers
	// TODO how should error handling work?
	// retry?  abort everything?  partial success?
	errors := []error{}
	for sha, filename := range shaToFilename {
		log.Debugf("about to scan %s", filename)
		err = scanner.ScanFile(filename, image.PullSpec, image.PullSpec, sha)
		if err != nil {
			errors = append(errors, err)
			log.Errorf("unable to scan file: %s", err.Error())
		}
	}

	// TODO this is kind of a hack, fix it
	if len(errors) > 0 {
		return fmt.Errorf("scan errors: %+v", errors)
	}

	// success!
	return nil
}

func (scanner *Scanner) ScanFile(path string, hubProjectName string, hubVersionName string, hubScanName string) error {
	return scanner.scanClient.Scan(path, hubProjectName, hubVersionName, hubScanName)
}

func buildLayerHashes(extractedDockerTarFileDir string) (map[string]string, error) {
	// 1. read manifest.json
	bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/manifest.json", extractedDockerTarFileDir))
	if err != nil {
		return nil, err
	}
	var images []manifestImage
	err = json.Unmarshal(bytes, &images)
	if err != nil {
		return nil, err
	}
	log.Debugf("parsed json: %+v", images)
	// 2. verify that there's only 1 image
	if len(images) != 1 {
		return nil, fmt.Errorf("expected 1 image, found %d", len(images))
	}
	// 3. go through json[0].Layers and calculate shas from files
	shaToFilename := map[string]string{} // map of sha to filename
	for _, layerId := range images[0].Layers {
		layerFileName := extractedDockerTarFileDir + "/" + layerId
		layerFile, err := os.Open(layerFileName)
		if err != nil {
			return nil, err
		}
		// defer layerFile.Close()
		hasher := sha256.New()
		// hasher := sha512.New512_224() // TODO which algorithm?
		if _, err := io.Copy(hasher, layerFile); err != nil {
			return nil, err
		}
		layerFile.Close()
		shaBytes := hasher.Sum(nil)
		sha := hex.EncodeToString(shaBytes)
		log.Infof("sha for %s: %s\n", layerFileName, sha)
		shaToFilename[sha] = layerFileName
	}
	return shaToFilename, nil
}

func extractTarFile(source string, dir string) error {
	log.Debugf("extract %s", source)
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()

	tarReader := tar.NewReader(f)

	for i := 0; ; i++ {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		log.Debugf("extractTarFile found a file: %+v", header)
		if header.Typeflag == tar.TypeDir {
			err := os.MkdirAll(fmt.Sprintf("%s/%s", dir, header.Name), 0755)
			if err != nil {
				return err
			}
		} else {
			file, err := os.Create(fmt.Sprintf("%s/%s", dir, header.Name))
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanUpFile(path string) {
	err := os.Remove(path)
	recordCleanUpFile(err == nil)
	if err != nil {
		log.Errorf("unable to remove file %s: %s", path, err.Error())
	} else {
		log.Infof("successfully cleaned up file %s", path)
	}
}

func cleanUpDir(path string) {
	err := os.RemoveAll(path)
	recordCleanUpFile(err == nil)
	if err != nil {
		log.Errorf("unable to remove directory %s: %s", path, err.Error())
	} else {
		log.Infof("successfully cleaned up directory %s", path)
	}
}
