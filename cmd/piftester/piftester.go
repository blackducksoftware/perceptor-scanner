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

	piftester "github.com/blackducksoftware/perceptor-scanner/pkg/piftester"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("starting piftester")
	configPath := os.Args[1]
	log.Infof("Config path: %s", configPath)

	config, err := piftester.GetConfig(configPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}

	level, err := config.GetLogLevel()
	if err != nil {
		log.Errorf(err.Error())
		panic(err)
	}
	log.SetLevel(level)

	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	http.Handle("/metrics", prometheus.Handler())

	pifTester := piftester.NewPifTester(config.ImageFacadePort)
	addr := fmt.Sprintf(":%d", config.Port)
	http.ListenAndServe(addr, nil)
	log.Infof("Http server started! -- %+v", pifTester)
}
