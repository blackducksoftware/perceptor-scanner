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

	"github.com/fatih/structs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	HubHost                 string
	HubUser                 string
	HubUserPasswordEnvVar   string
	HubPort                 int
	HubClientTimeoutSeconds int

	LogLevel string
	Port     int

	ImageFacadeHost string
	ImageFacadePort int

	PerceptorHost string
	PerceptorPort int
}

// The following does a couple of things:
// 	* Provide a map of sensible defaults for the struct
//	* Provide a map of values that can be loaded into viper so all structure
//		config keys are loaded. This allows viper to override these defaults
//		from either the configMap or environment variables
//
// If the struct above is modified, this method MUST also be modified
// so viper is aware of the new structure
func (config *Config) GetDefaults() map[string]interface{} {
	defaults := Config{
		HubHost:                 "",
		HubUser:                 "",
		HubUserPasswordEnvVar:   "",
		HubPort:                 443,
		HubClientTimeoutSeconds: 5,

		LogLevel: "warn",
		Port:     0,

		ImageFacadeHost: "localhost",
		ImageFacadePort: 0,

		PerceptorHost: "",
		PerceptorPort: 0,
	}

	return structs.Map(defaults)
}

func (config *Config) GetLogLevel() (log.Level, error) {
	return log.ParseLevel(config.LogLevel)
}

func GetConfig(configPath string, ignoreConfigMap bool) (*Config, error) {
	config := new(Config)

	// Set the defaults
	for key, value := range config.GetDefaults() {
		viper.SetDefault(key, value)
	}

	// Look for environment variables (prefixed with "PIS_")
	viper.SetEnvPrefix("pis") // uppercased automatically
	viper.AutomaticEnv()

	viper.SetConfigFile(configPath)

	// If IgnoreConfigMap set then all config items should be set via environment or defaults
	if cm.IgnoreConfigMap == false {
		err := viper.ReadInConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Ports must be reachable
	if config.Port == 0 || config.PerceptorPort == 0 || config.ImageFacadePort == 0 {
		err = fmt.Errorf("Need non zero numbers for Port (got %d), PerceptorPort (got %d), HubPort (got %d), and ImageFacadePort (got %d)",
			config.Port,
			config.PerceptorPort,
			config.HubPort,
			config.ImageFacadePort)
	}
	return config, err
}
