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
	"io/ioutil"
	"net/http"
	"strings"

	pdocker "github.com/blackducksoftware/perceptor-scanner/pkg/docker"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	setupHTTPServer()
}

func setupHTTPServer() {
	config, err := GetConfig()
	if err != nil {
		log.Errorf("unable to read config: %s", err.Error())
		panic(err)
	}
	log.Infof("got config: %+v", config)
	imagePuller := pdocker.NewImagePuller(config.DockerUser, config.DockerPassword)
	results := []pdocker.ImagePullStats{}
	http.HandleFunc("/pull", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var image *Image
			err = json.Unmarshal(body, &image)
			if err != nil {
				log.Infof("unable to ummarshal JSON for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			go func() {
				results = append(results, imagePuller.PullImage(image))
			}()
		}
	})
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		statsBytes, err := json.Marshal(results)
		if err != nil {
			http.Error(w, err.Error(), 400)
		} else {
			fmt.Fprint(w, string(statsBytes))
		}
	})
	http.HandleFunc("/prettystats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "start pretty stats:\n")
		for _, result := range results {
			fmt.Fprint(w, "stats: ")
			if result.CreateDuration != nil {
				fmt.Fprintf(w, "create duration, seconds: %d", int(result.CreateDuration.Seconds()))
			}
			if result.TarFileSizeMBs != nil {
				fmt.Fprintf(w, "  file size: %d", result.TarFileSizeMBs)
			}
			if result.Err != nil {
				fmt.Fprintf(w, "  error: %+v", result.Err)
			}
			if result.SaveDuration != nil {
				fmt.Fprintf(w, "save duration, seconds: %d", int(result.SaveDuration.Seconds()))
			}
			fmt.Fprint(w, "\n")
		}
		fmt.Fprint(w, "end pretty stats")
	})

	log.Info("Serving")
	http.ListenAndServe(":3004", nil)
}

type Image struct {
	PullSpec string
}

func (image *Image) DockerPullSpec() string {
	return image.PullSpec
}

func (image *Image) DockerTarFilePath() string {
	return strings.Replace(image.PullSpec, "/", "_", -1)
}

type Config struct {
	DockerUser     string
	DockerPassword string
}

func GetConfig() (*Config, error) {
	var config *Config

	viper.SetConfigName("imagefacadewebserver_conf")
	viper.AddConfigPath("/etc/imagefacadewebserver")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return config, nil
}
