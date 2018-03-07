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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/blackducksoftware/perceptor-scanner/pkg/scanner"
	"github.com/blackducksoftware/perceptor/pkg/api"
	"github.com/blackducksoftware/perceptor/pkg/core"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("started")

	// config, err := GetConfig()
	// if err != nil {
	// 	log.Errorf("Failed to load configuration: %s", err.Error())
	// 	panic(err)
	// }
	pifTester := NewPifTester()
	http.ListenAndServe(":3005", nil)
	log.Info("Http server started! -- %+v", pifTester)
}

// // ScannerConfig contains all configuration for Perceptor
// type ScannerConfig struct {
// 	HubHost         string
// 	HubUser         string
// 	HubUserPassword string
// }
//
// // GetScannerConfig returns a configuration object to configure Perceptor
// func GetScannerConfig() (*ScannerConfig, error) {
// 	var cfg *ScannerConfig
//
// 	viper.SetConfigName("perceptor_scanner_conf")
// 	viper.AddConfigPath("/etc/perceptor_scanner")
//
// 	err := viper.ReadInConfig()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read config file: %v", err)
// 	}
//
// 	err = viper.Unmarshal(&cfg)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
// 	}
// 	return cfg, nil
// }

type PifTester struct {
	ImageMap            map[core.Image]bool
	ImageErrors         map[core.Image][]string
	ImageQueue          []core.Image
	imagePuller         *scanner.ImageFacadePuller
	getNextImageChannel chan func(*core.Image)
}

func NewPifTester() *PifTester {
	responder := core.NewHTTPResponder()
	pif := &PifTester{
		ImageMap:            map[core.Image]bool{},
		ImageErrors:         map[core.Image][]string{},
		ImageQueue:          []core.Image{},
		imagePuller:         scanner.NewImageFacadePuller("http://perceptor-imagefacade"),
		getNextImageChannel: make(chan func(*core.Image)),
	}

	go func() {
		for {
			select {
			case image := <-responder.AddImageChannel:
				pif.addImage(image)
			case pod := <-responder.AddPodChannel:
				pif.addPod(pod)
			case allPods := <-responder.AllPodsChannel:
				for _, pod := range allPods {
					pif.addPod(pod)
				}
			case allImages := <-responder.AllImagesChannel:
				for _, image := range allImages {
					pif.addImage(image)
				}
			case continuation := <-responder.GetModelChannel:
				jsonBytes, err := json.Marshal(pif)
				if err != nil {
					panic(err)
				}
				go continuation(string(jsonBytes))

			case continuation := <-pif.getNextImageChannel:
				image := pif.getNextImage()
				go continuation(image)

			// we don't care about these:
			case _ = <-responder.UpdatePodChannel:
				break
			case _ = <-responder.DeletePodChannel:
				break
			case _ = <-responder.PostNextImageChannel:
				break
			case _ = <-responder.PostFinishScanJobChannel:
				break
			case _ = <-responder.SetConcurrentScanLimitChannel:
				break
			case _ = <-responder.GetScanResultsChannel:
				break
			}
		}
	}()

	go pif.startPullingImages()

	api.SetupHTTPServer(responder)
	return pif
}

func (pif *PifTester) addPod(pod core.Pod) {
	for _, cont := range pod.Containers {
		pif.addImage(cont.Image)
	}
}

func (pif *PifTester) addImage(image core.Image) {
	_, ok := pif.ImageMap[image]
	if ok {
		return
	}
	log.Infof("got new image: %s", image.PullSpec())
	pif.ImageMap[image] = false
	pif.ImageQueue = append(pif.ImageQueue, image)
}

func (pif *PifTester) getNextImage() *core.Image {
	if len(pif.ImageQueue) == 0 {
		log.Infof("no next image")
		return nil
	}
	first := pif.ImageQueue[0]
	log.Infof("next image: %s", first.PullSpec())
	pif.ImageQueue = pif.ImageQueue[1:]
	return &first
}

func (pif *PifTester) finishImage(image core.Image, err error) {
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}
	log.Infof("finish image %s with error %s", image.PullSpec(), errorString)
	if err == nil {
		pif.ImageMap[image] = true
	} else {
		// dunno -- record and try again?
		errors, ok := pif.ImageErrors[image]
		if !ok {
			panic(fmt.Errorf("unable to find image %s in ImageErrors", image.PullSpec()))
		}
		errors = append(errors, err.Error())
		pif.ImageErrors[image] = errors
		pif.ImageQueue = append(pif.ImageQueue, image)
	}
}

func (pif *PifTester) startPullingImages() {
	for {
		var wg sync.WaitGroup
		wg.Add(1)
		pif.getNextImageChannel <- func(image *core.Image) {
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
    "ImageMap": %s
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
	q, err := json.Marshal(queue)
	if err != nil {
		panic(err)
	}
	i, err := json.Marshal(iMap)
	if err != nil {
		panic(err)
	}
	jsonString := fmt.Sprintf(str, string(q), string(i))
	return []byte(jsonString), nil
}

//
// func (pif *PifTester) MarshalText() (text []byte, err error) {
// 	return []byte(s.String()), nil
// }
