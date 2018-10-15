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

package core

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// HubConfig handles Hub-specific configuration
type HubConfig struct {
	User                      string
	PasswordEnvVar            string
	ClientTimeoutMilliseconds int
	Port                      int
	ConcurrentScanLimit       int
	TotalScanLimit            int
}

// ClientTimeout converts the milliseconds to a duration
func (config *HubConfig) ClientTimeout() time.Duration {
	return time.Duration(config.ClientTimeoutMilliseconds) * time.Millisecond
}

// Timings ...
type Timings struct {
	CheckForStalledScansPauseHours  int
	StalledScanClientTimeoutHours   int
	ModelMetricsPauseSeconds        int
	UnknownImagePauseMilliseconds   int
	PruneOrphanedImagesPauseMinutes int
}

// CheckForStalledScansPause ...
func (t *Timings) CheckForStalledScansPause() time.Duration {
	return time.Duration(t.CheckForStalledScansPauseHours) * time.Hour
}

// StalledScanClientTimeout ...
func (t *Timings) StalledScanClientTimeout() time.Duration {
	return time.Duration(t.StalledScanClientTimeoutHours) * time.Hour
}

// ModelMetricsPause ...
func (t *Timings) ModelMetricsPause() time.Duration {
	return time.Duration(t.ModelMetricsPauseSeconds) * time.Second
}

// UnknownImagePause ...
func (t *Timings) UnknownImagePause() time.Duration {
	return time.Duration(t.UnknownImagePauseMilliseconds) * time.Millisecond
}

// PruneOrphanedImagesPause ...
func (t *Timings) PruneOrphanedImagesPause() time.Duration {
	return time.Duration(t.PruneOrphanedImagesPauseMinutes) * time.Minute
}

// Config contains all configuration for Perceptor
type Config struct {
	Hub         *HubConfig
	Timings     *Timings
	UseMockMode bool
	Port        int
	LogLevel    string
}

// GetLogLevel .....
func (config *Config) GetLogLevel() (log.Level, error) {
	return log.ParseLevel(config.LogLevel)
}
