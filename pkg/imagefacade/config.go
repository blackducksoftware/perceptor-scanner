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
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	// These are openshift specific, and allow pulling from the openshift docker registry
	DockerUser               string
	DockerPassword           string
	InternalDockerRegistries []string

	CreateImagesOnly bool
	Port             int
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
