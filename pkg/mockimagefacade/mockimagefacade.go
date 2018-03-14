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
	"io"
	"os"

	common "github.com/blackducksoftware/perceptor-scanner/pkg/common"
	imagefacade "github.com/blackducksoftware/perceptor-scanner/pkg/imagefacade"
	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/core"
	log "github.com/sirupsen/logrus"
)

type MockImagefacade struct {
	server *imagefacade.HTTPServer
}

func NewMockImagefacade() *MockImagefacade {
	server := imagefacade.NewHTTPServer()

	// ImageFacade REST API
	go func() {
		for {
			select {
			case pullImage := <-server.PullImageChannel():
				log.Infof("received pullImage: %+v", pullImage.Image)
				pullImage.Continuation(nil)
			case getImage := <-server.GetImageChannel():
				log.Infof("received getImage: %+v", getImage.Image)
				sourcePath := "/tmp/alpine.tar"
				err := copyFile(sourcePath, getImage.Image.DockerTarFilePath())
				status := common.ImageStatusDone
				if err != nil {
					log.Errorf("unable to copy file from %s to %s: %s", sourcePath, getImage.Image.DockerTarFilePath(), err.Error())
					status = common.ImageStatusError
				}
				getImage.Continuation(status)
			}
		}
	}()

	// Perceptor REST API
	responder := core.NewHTTPResponder()

	go func() {
		for {
			select {
			case continuation := <-responder.PostNextImageChannel:
				go continuation(core.NewImage("host/user/project", "somesha"))
			case scanJob := <-responder.PostFinishScanJobChannel:
				log.Infof("finished scan job: %+v", scanJob)
			case continuation := <-responder.GetModelChannel:
				go continuation(string("{\"status\":\"TODO\"}"))

			// we don't care about these:
			case _ = <-responder.AddImageChannel:
				break
			case _ = <-responder.AddPodChannel:
				break
			case _ = <-responder.AllPodsChannel:
				break
			case _ = <-responder.AllImagesChannel:
				break
			case _ = <-responder.UpdatePodChannel:
				break
			case _ = <-responder.DeletePodChannel:
				break
			case _ = <-responder.SetConcurrentScanLimitChannel:
				break
			case _ = <-responder.GetScanResultsChannel:
				break
			}
		}
	}()

	api.SetupHTTPServer(responder)

	return &MockImagefacade{server: server}
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
