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
	"sync"
	"time"

	"github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/blackducksoftware/perceptor-scanner/pkg/scanner"
	"github.com/blackducksoftware/perceptor/pkg/api"
	m "github.com/blackducksoftware/perceptor/pkg/core/model"
	"github.com/prometheus/common/log"
)

type PifTester struct {
	ImageMap            map[m.Image]bool
	ImageErrors         map[m.Image][]string
	ImageQueue          []m.Image
	imagePuller         *scanner.ImageFacadePuller
	getNextImageChannel chan func(*m.Image)
}

func NewPifTester(imageFacadePort int) *PifTester {
	return nil // TODO get this working again
	// responder := core.NewHTTPResponder()
	// pif := &PifTester{
	// 	ImageMap:            map[m.Image]bool{},
	// 	ImageErrors:         map[m.Image][]string{},
	// 	ImageQueue:          []m.Image{},
	// 	imagePuller:         scanner.NewImageFacadePuller("http://perceptor-imagefacade", imageFacadePort),
	// 	getNextImageChannel: make(chan func(*m.Image)),
	// }
	//
	// go func() {
	// 	for {
	// 		select {
	// 		case image := <-responder.AddImageChannel:
	// 			pif.addImage(image.Image)
	// 		case pod := <-responder.AddPodChannel:
	// 			pif.addPod(pod.Pod)
	// 		case allPods := <-responder.AllPodsChannel:
	// 			for _, pod := range allPods.Pods {
	// 				pif.addPod(pod)
	// 			}
	// 		case allImages := <-responder.AllImagesChannel:
	// 			for _, image := range allImages.Images {
	// 				pif.addImage(image)
	// 			}
	// 		case action := <-responder.GetModelChannel:
	// 			model := pif.APIModel()
	// 			action.Done <- &model
	//
	// 		case continuation := <-pif.getNextImageChannel:
	// 			image := pif.getNextImage()
	// 			go continuation(image)
	//
	// 		// we don't care about these:
	// 		case _ = <-responder.UpdatePodChannel:
	// 			break
	// 		case _ = <-responder.DeletePodChannel:
	// 			break
	// 		case _ = <-responder.PostNextImageChannel:
	// 			break
	// 		case _ = <-responder.PostFinishScanJobChannel:
	// 			break
	// 		case _ = <-responder.PostConfigChannel:
	// 			break
	// 		case _ = <-responder.GetScanResultsChannel:
	// 			break
	// 		case _ = <-responder.ResetCircuitBreakerChannel:
	// 			break
	// 		}
	// 	}
	// }()
	//
	// go pif.startPullingImages()
	//
	// api.SetupHTTPServer(responder)
	// return pif
}

func (pif *PifTester) addPod(pod m.Pod) {
	for _, cont := range pod.Containers {
		pif.addImage(cont.Image)
	}
}

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
		var wg sync.WaitGroup
		wg.Add(1)
		pif.getNextImageChannel <- func(image *m.Image) {
			if image != nil {
				err := pif.imagePuller.PullImage(&common.Image{PullSpec: image.PullSpec()})
				pif.finishImage(*image, err)
			}
			wg.Done()
		}
		wg.Wait()
		time.Sleep(5 * time.Second)
	}
}

func (pif *PifTester) MarshalJSON() ([]byte, error) {
	str := `
  {
		"ImageQueue": %s,
		"ImageMap": %s,
		"ImageErrors": %s
  }
  `
	queue := []string{}
	for _, item := range pif.ImageQueue {
		queue = append(queue, item.PullSpec())
	}
	iMap := map[string]bool{}
	for key, val := range pif.ImageMap {
		iMap[key.PullSpec()] = val
	}
	eMap := map[string][]string{}
	for key, val := range pif.ImageErrors {
		eMap[key.PullSpec()] = val
	}
	q, err := json.Marshal(queue)
	if err != nil {
		panic(err)
	}
	i, err := json.Marshal(iMap)
	if err != nil {
		panic(err)
	}
	e, err := json.Marshal(eMap)
	if err != nil {
		panic(err)
	}
	jsonString := fmt.Sprintf(str, string(q), string(i), string(e))
	return []byte(jsonString), nil
}

//
// func (pif *PifTester) MarshalText() (text []byte, err error) {
// 	return []byte(s.String()), nil
// }
