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

package piftester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blackducksoftware/perceptor/pkg/core"

	"github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/blackducksoftware/perceptor-scanner/pkg/scanner"
	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	log "github.com/sirupsen/logrus"
)

type action struct {
	name  string
	apply func() error
}

// PifTester helps with testing the image facade.
type PifTester struct {
	ImageMap          map[m.Image]bool
	ImageErrors       map[m.Image][]string
	ImageQueue        []m.Image
	imageFacadeClient scanner.ImageFacadeClientInterface
	actions           chan *action
	stop              <-chan struct{}
}

// NewPifTester ...
func NewPifTester(imageFacadeHost string, imageFacadePort int, stop <-chan struct{}) *PifTester {
	pif := &PifTester{
		ImageMap:          map[m.Image]bool{},
		ImageErrors:       map[m.Image][]string{},
		ImageQueue:        []m.Image{},
		imageFacadeClient: scanner.NewImageFacadeClient(imageFacadeHost, imageFacadePort),
		actions:           make(chan *action),
		stop:              stop,
	}

	go func() {
		for {
			select {
			case <-stop:
				return
			case a := <-pif.actions:
				log.Debugf("about to start processing action %s", a.name)
				err := a.apply()
				if err != nil {
					log.Errorf("unable to process %s: %s", a.name, err.Error())
				} else {
					log.Debugf("successfully processed %s", a.name)
				}
			}
		}
	}()

	go pif.startPullingImages()

	api.SetupHTTPServer(pif)
	return pif
}

func (pif *PifTester) addPod(pod m.Pod) {
	for _, cont := range pod.Containers {
		pif.addImage(cont.Image)
	}
}

// APIModel ...
func (pif *PifTester) APIModel() api.Model {
	_, err := json.MarshalIndent(pif, "", "  ")
	if err != nil {
		panic(err)
	}
	// TODO make this work?
	return api.Model{}
}

func (pif *PifTester) addImage(image m.Image) {
	_, ok := pif.ImageMap[image]
	if ok {
		return
	}
	log.Infof("got new image: %s", image.PullSpec())
	pif.ImageMap[image] = false
	pif.ImageQueue = append(pif.ImageQueue, image)
}

func (pif *PifTester) getNextImage() *m.Image {
	if len(pif.ImageQueue) == 0 {
		log.Infof("no next image")
		return nil
	}
	first := pif.ImageQueue[0]
	log.Infof("next image: %s", first.PullSpec())
	pif.ImageQueue = pif.ImageQueue[1:]
	return &first
}

func (pif *PifTester) finishImage(image m.Image, err error) {
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}
	log.Infof("finish image %s with error %s", image.PullSpec(), errorString)
	recordImagePullResult(err == nil)
	if err == nil {
		pif.ImageMap[image] = true
	} else {
		errors, ok := pif.ImageErrors[image]
		// no reason it should ever not already be in the map
		if !ok {
			panic(fmt.Errorf("unable to find image %s in ImageErrors", image.PullSpec()))
		}
		// dunno -- record and try again?
		errors = append(errors, err.Error())
		pif.ImageErrors[image] = errors
		pif.ImageQueue = append(pif.ImageQueue, image)
	}
}

func (pif *PifTester) startPullingImages() {
	for {
		image := pif.getNextImage()
		if image != nil {
			err := pif.imageFacadeClient.PullImage(&common.Image{PullSpec: image.PullSpec()})
			pif.finishImage(*image, err)
		}
		select {
		case <-pif.stop:
			return
		case <-time.After(5 * time.Second):
			// continue
		}
	}
}

// api.Responder implementation

// GetModel ...
func (pif *PifTester) GetModel() api.Model {
	ch := make(chan api.Model)
	pif.actions <- &action{"getModel", func() error {
		model := pif.APIModel()
		ch <- model
		return nil
	}}
	return <-ch
}

// perceiver

// AddPod ...
func (pif *PifTester) AddPod(pod api.Pod) error {
	pif.actions <- &action{"addPod", func() error {
		corePod, err := core.APIPodToCorePod(pod)
		if err != nil {
			return err
		}
		pif.addPod(*corePod)
		return nil
	}}
	return nil
}

// UpdatePod ...
func (pif *PifTester) UpdatePod(pod api.Pod) error {
	pif.actions <- &action{"updatePod", func() error {
		corePod, err := core.APIPodToCorePod(pod)
		if err != nil {
			return err
		}
		pif.addPod(*corePod)
		return nil
	}}
	return nil
}

// DeletePod ...
func (pif *PifTester) DeletePod(qualifiedName string) {}

// GetScanResults ...
func (pif *PifTester) GetScanResults() api.ScanResults { return api.ScanResults{} }

// AddImage ...
func (pif *PifTester) AddImage(image api.Image) error {
	pif.actions <- &action{"addImage", func() error {
		coreImage, err := core.APIImageToCoreImage(image)
		if err != nil {
			return err
		}
		pif.addImage(*coreImage)
		return nil
	}}
	return nil
}

// UpdateAllPods ...
func (pif *PifTester) UpdateAllPods(allPods api.AllPods) error {
	pif.actions <- &action{"allPods", func() error {
		for _, pod := range allPods.Pods {
			corePod, err := core.APIPodToCorePod(pod)
			if err != nil {
				return err
			}
			pif.addPod(*corePod)
		}
		return nil
	}}
	return nil
}

// UpdateAllImages ...
func (pif *PifTester) UpdateAllImages(allImages api.AllImages) error {
	pif.actions <- &action{"allImages", func() error {
		for _, image := range allImages.Images {
			coreImage, err := core.APIImageToCoreImage(image)
			if err != nil {
				return err
			}
			pif.addImage(*coreImage)
		}
		return nil
	}}
	return nil
}

// scanner

// GetNextImage ...
func (pif *PifTester) GetNextImage() api.NextImage {
	ch := make(chan *api.NextImage)
	pif.actions <- &action{"getNextImage", func() error {
		image := pif.getNextImage()
		ch <- api.NewNextImage(&api.ImageSpec{
			HubProjectName:        image.HubProjectName(),
			HubProjectVersionName: image.HubProjectVersionName(),
			HubScanName:           image.HubScanName(),
			HubURL:                "????? let's hope this isn't required",
			Priority:              image.Priority,
			Repository:            image.Repository,
			Sha:                   string(image.Sha),
			Tag:                   image.Tag})
		return nil
	}}
	return *(<-ch)
}

// PostFinishScan ...
func (pif *PifTester) PostFinishScan(job api.FinishedScanClientJob) error { return nil }

// internal use

// PostCommand ...
func (pif *PifTester) PostCommand(commands *api.PostCommand) {}

// errors

// NotFound ...
func (pif *PifTester) NotFound(w http.ResponseWriter, r *http.Request) {
	log.Errorf("not found: %s", r.URL.String())
}

// Error ...
func (pif *PifTester) Error(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	log.Errorf("%s from %s", err.Error(), r.URL.String())
}
