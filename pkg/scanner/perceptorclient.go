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
	"time"

	"github.com/blackducksoftware/perceptor/pkg/api"
	resty "github.com/go-resty/resty"
	log "github.com/sirupsen/logrus"
)

const (
	nextImagePath       = "nextimage"
	imageLayersPath     = "imagelayers"
	shouldScanLayerPath = "shouldscanlayer"
	finishedScanPath    = "finishedscan"
)

type PerceptorClient struct {
	Resty *resty.Client
	Host  string
	Port  int
}

func NewPerceptorClient(host string, port int) *PerceptorClient {
	restyClient := resty.New()
	restyClient.SetRetryCount(3)
	restyClient.SetRetryWaitTime(500 * time.Millisecond)
	restyClient.SetTimeout(time.Duration(5 * time.Second))
	return &PerceptorClient{
		Resty: restyClient,
		Host:  host,
		Port:  port,
	}
}

func (pc *PerceptorClient) GetNextImage() (*api.NextImage, error) {
	url := fmt.Sprintf("http://%s:%d/%s", pc.Host, pc.Port, nextImagePath)
	nextImage := api.NextImage{}
	log.Debugf("about to issue post request to url %s", url)
	resp, err := pc.Resty.R().
		SetHeader("Content-Type", "application/json").
		SetResult(&nextImage).
		Post(url)
	log.Debugf("received resp %+v and error %+v from url %s", resp, err, url)
	recordHTTPStats(nextImagePath, resp.StatusCode())
	if err != nil {
		recordScannerError("unable to get next image")
		log.Errorf("unable to get next image: %s", err.Error())
		return nil, err
	} else if (resp.StatusCode() < 200) || (resp.StatusCode() >= 300) {
		recordScannerError("unable to get next image -- bad status code")
		err := fmt.Errorf("unable to get next image; body %s and status code %d", string(resp.Body()), resp.StatusCode())
		log.Error(err.Error())
		return nil, err
	}
	return &nextImage, nil
}

func (pc *PerceptorClient) PostImageLayers(imageLayers *api.ImageLayers) error {
	url := fmt.Sprintf("http://%s:%d/%s", pc.Host, pc.Port, imageLayersPath)
	log.Debugf("about to issue post request %+v to url %s", imageLayers, url)
	resp, err := pc.Resty.R().
		SetBody(imageLayers).
		Post(url)
	log.Debugf("received resp %+v, status code %d, error %+v from url %s", resp, resp.StatusCode(), err, url)
	recordHTTPStats(imageLayersPath, resp.StatusCode())
	if (resp.StatusCode() < 200) || (resp.StatusCode() >= 300) {
		recordScannerError("unable to post image layers -- bad status code")
		err := fmt.Errorf("unable to post image layers; body %s and status code %d", string(resp.Body()), resp.StatusCode())
		log.Error(err.Error())
		return err
	}
	return err
}

func (pc *PerceptorClient) GetShouldScanLayer(request *api.LayerScanRequest) (*api.LayerScanResponse, error) {
	url := fmt.Sprintf("http://%s:%d/%s", pc.Host, pc.Port, shouldScanLayerPath)
	response := api.LayerScanResponse{}
	log.Debugf("about to issue post request %+v to url %s", request, url)
	resp, err := pc.Resty.R().SetBody(request).SetResult(&response).Post(url)
	log.Debugf("received resp %+v, status code %d, error %+v from url %s", resp, resp.StatusCode(), err, url)
	recordHTTPStats(shouldScanLayerPath, resp.StatusCode())
	if err != nil {
		return nil, err
	} else if (resp.StatusCode() < 200) || (resp.StatusCode() >= 300) {
		recordScannerError("unable to post should scan layer -- bad status code")
		err := fmt.Errorf("unable to post should scan layer; body %s and status code %d", string(resp.Body()), resp.StatusCode())
		log.Error(err.Error())
		return nil, err
	}
	return &response, nil
}

func (pc *PerceptorClient) PostFinishedScan(scan *api.FinishedScanClientJob) error {
	url := fmt.Sprintf("http://%s:%d/%s", pc.Host, pc.Port, finishedScanPath)
	log.Debugf("about to issue post request %+v to url %s", scan, url)
	resp, err := pc.Resty.R().SetBody(scan).Post(url)
	log.Debugf("received resp %+v, status code %d, error %+v from url %s", resp, resp.StatusCode(), err, url)
	recordHTTPStats(finishedScanPath, resp.StatusCode())
	if (resp.StatusCode() < 200) || (resp.StatusCode() >= 300) {
		recordScannerError("unable to post finished scan -- bad status code")
		err := fmt.Errorf("unable to post finished scan; body %s and status code %d", string(resp.Body()), resp.StatusCode())
		log.Error(err.Error())
		return err
	}
	return err
}
