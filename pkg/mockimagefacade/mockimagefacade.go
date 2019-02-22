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

package mockimagefacade

import (
	"fmt"
	"io"
	"net/url"
	"os"

	common "github.com/blackducksoftware/perceptor-scanner/pkg/common"
	imagefacade "github.com/blackducksoftware/perceptor-scanner/pkg/imagefacade"
	log "github.com/sirupsen/logrus"
)

// MockImagefacade ...
type MockImagefacade struct{}

// NewMockImagefacade ...
func NewMockImagefacade() *MockImagefacade {
	mif := &MockImagefacade{}
	imagefacade.SetupHTTPServer(mif)
	return mif
}

func copyFile(source string, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// PullImage ...
func (mif *MockImagefacade) PullImage(image *common.Image) error {
	log.Infof("received pullImage: %+v", image)
	return nil
}

// GetImage ...
func (mif *MockImagefacade) GetImage(image *common.Image) common.ImageState {
	log.Infof("received getImage: %+v", image)
	sourcePath := "/tmp/alpine.tar"
	err := copyFile(sourcePath, image.DockerTarFilePath())
	if err != nil {
		log.Errorf("unable to copy file from %s to %s: %s", sourcePath, image.DockerTarFilePath(), err.Error())
		return *common.NewImageState(common.ImageStatusError, nil)
	}
	downloadURL, _ := url.Parse(fmt.Sprintf("file://%s", image.DockerPullSpec()))
	return *common.NewImageState(common.ImageStatusDone, downloadURL)
}

// GetModel ...
func (mif *MockImagefacade) GetModel() map[string]interface{} {
	return map[string]interface{}{"todo": "unimplemented"}
}
