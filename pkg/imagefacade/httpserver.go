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

package imagefacade

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	common "github.com/blackducksoftware/perceptor-scanner/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type HTTPServer struct {
	pullImage chan *pullImage
	getImage  chan *getImage
}

func newHTTPServer() *HTTPServer {
	server := &HTTPServer{
		pullImage: make(chan *pullImage),
		getImage:  make(chan *getImage)}
	server.setup()
	return server
}

func (h *HTTPServer) PullImageChannel() <-chan *pullImage {
	return h.pullImage
}

func (h *HTTPServer) GetImageChannel() <-chan *getImage {
	return h.getImage
}

func (h *HTTPServer) setup() {
	http.HandleFunc("/pullimage", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for pullimage: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var image *common.Image
			err = json.Unmarshal(body, &image)
			if err != nil {
				log.Infof("unable to ummarshal JSON for pullimage: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var response string
			var success bool
			var wg sync.WaitGroup
			wg.Add(1)
			continuation := func(err error) {
				success = err == nil
				response = fmt.Sprintf(`{"PullSpec":"%s","Success":"%t"}`, image.PullSpec, success)
				wg.Done()
			}
			h.pullImage <- &pullImage{image, continuation}
			wg.Done()
			if success {
				fmt.Fprint(w, response)
			} else {
				http.Error(w, response, 503)
			}
		default:
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for getimage: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var image *common.Image
			err = json.Unmarshal(body, &image)
			if err != nil {
				log.Infof("unable to ummarshal JSON for getimage: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var response string
			var wg sync.WaitGroup
			wg.Add(1)
			continuation := func(isDone bool) {
				response = fmt.Sprintf(`{"PullSpec":"%s","Done":"%t"}`, image.PullSpec, isDone)
				wg.Done()
			}
			h.getImage <- &getImage{image: image, continuation: continuation}
			wg.Wait()
			fmt.Fprintf(w, response)
		default:
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		// TODO
		// statsBytes, err := json.Marshal(results)
		// if err != nil {
		// 	http.Error(w, err.Error(), 400)
		// } else {
		// 	fmt.Fprint(w, string(statsBytes))
		// }
	})
	http.Handle("/metrics", prometheus.Handler())
}
