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
	"fmt"
	"net/http"
	"os"

	"github.com/blackducksoftware/perceptor-scanner/pkg/imagefacade"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	log.Info("started")

	config, err := GetConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}

	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	imageFacade := imagefacade.NewImageFacade(config.DockerUser, config.DockerPassword)

	log.Infof("successfully instantiated imagefacade -- %+v", imageFacade)

	port := "3004"
	addr := fmt.Sprintf(":%s", port) // api.PerceptorImagefacadePort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

type Config struct {
	DockerUser     string // DockerUser and DockerPassword are openshift specific -- to allow pulling from the openshift docker registry
	DockerPassword string
}

// GetConfig returns a configuration object to configure Perceptor
func GetConfig() (*Config, error) {
	var cfg *Config

	viper.SetConfigName("perceptor_imagefacade_conf")
	viper.AddConfigPath("/etc/perceptor_imagefacade")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return cfg, nil
}
