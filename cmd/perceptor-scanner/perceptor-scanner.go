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
	"github.com/spf13/viper"
)

func main() {
	log.Info("started")

	config, err := GetScannerConfig()
	if err != nil {
		log.Errorf("Failed to load configuration: %s", err.Error())
		panic(err)
	}

	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	scannerManager, err := scanner.NewScanner(config.HubHost, config.HubUser, config.HubUserPassword)
	if err != nil {
		log.Errorf("unable to instantiate scanner: %s", err.Error())
		panic(err)
	}

	log.Info("successfully instantiated scanner: %s", scannerManager)

	http.Handle("/metrics", prometheus.Handler())

	addr := fmt.Sprintf(":%d", config.Port)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

// ScannerConfig contains all configuration for Perceptor
type ScannerConfig struct {
	HubHost         string
	HubUser         string
	HubUserPassword string
	Port            int
}

// GetScannerConfig returns a configuration object to configure Perceptor
func GetScannerConfig() (*ScannerConfig, error) {
	var cfg *ScannerConfig

	viper.SetConfigName("perceptor_scanner_conf")
	viper.AddConfigPath("/etc/perceptor_scanner")

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
