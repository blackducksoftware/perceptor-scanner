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
	"strings"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// HubConfig ...
type HubConfig struct {
	User                 string
	PasswordEnvVar       string
	Port                 int
	ClientTimeoutSeconds int
}

// ImageFacadeConfig ...
type ImageFacadeConfig struct {
	Host string
	Port int
}

// GetHost ...
func (ifc *ImageFacadeConfig) GetHost() string {
	if ifc.Host == "" {
		return "localhost"
	}
	return ifc.Host
}

// PerceptorConfig ...
type PerceptorConfig struct {
	Host string
	Port int
}

// Config ...
type Config struct {
	Hub         *HubConfig
	ImageFacade *ImageFacadeConfig
	Perceptor   *PerceptorConfig

	ImageDirectory string

	LogLevel string
	Port     int
}

// GetLogLevel ...
func (config *Config) GetLogLevel() (log.Level, error) {
	return log.ParseLevel(config.LogLevel)
}

// GetConfig ...
func GetConfig(configPath string) (*Config, error) {
	var config *Config

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetEnvPrefix("PCP")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		viper.BindEnv("ImageFacade_Host")
		viper.BindEnv("ImageFacade_Port")

		viper.BindEnv("Perceptor_Host")
		viper.BindEnv("Perceptor_Port")

		viper.BindEnv("Hub_User")
		viper.BindEnv("Hub_Port")
		viper.BindEnv("Hub_PasswordEnvVar")
		viper.BindEnv("Hub_ClientTimeoutSeconds")

		viper.BindEnv("LogLevel")
		viper.BindEnv("Port")

		viper.AutomaticEnv()
	}

	viper.SetConfigFile(configPath)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, errors.Annotatef(err, "failed to read config file")
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to unmarshal config")
	}

	return config, nil
}
