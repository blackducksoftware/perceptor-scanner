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

	"github.com/blackducksoftware/perceptor-scanner/pkg/scanner"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("started")

	config, err := scanner.GetConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err.Error())
		panic(err)
	}

	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	scannerManager, err := scanner.NewScanner(config)
	if err != nil {
		log.Errorf("unable to instantiate scanner: %v", err.Error())
		panic(err)
	}
	addr := fmt.Sprintf(":%d", config.Port)
	
	log.Info("successfully instantiated scanner: ( %v ), now starting on address %v", scannerManager, addr )

	http.Handle("/metrics", prometheus.Handler())

	log.Info("Starting webserver now. Metrics available on the /metrics endpoint.")
	http.ListenAndServe(addr, nil)
}
