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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsCounter *prometheus.CounterVec
var actionsCounter *prometheus.CounterVec
var reducerActivityCounter *prometheus.CounterVec

func recordHttpRequest(path string) {
	httpRequestsCounter.With(prometheus.Labels{"path": path}).Inc()
}

func recordActionType(action string) {
	actionsCounter.With(prometheus.Labels{"action": action}).Inc()
}

func recordReducerActivity(isActive bool, duration time.Duration) {
	state := "idle"
	if isActive {
		state = "active"
	}
	reducerActivityCounter.With(prometheus.Labels{"state": state}).Add(duration.Seconds())
}

func init() {
	httpRequestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "imagefacade",
		Name:      "http_requests_received",
		Help:      "HTTP requests received by imagefacade",
	},
		[]string{"path"})

	actionsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "imagefacade",
		Name:      "actions",
		Help:      "actions processed by imagefacade and applied to the model",
	},
		[]string{"action"})

	reducerActivityCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "imagefacade",
		Name:      "reducer_activity",
		Help:      "activity of the reducer -- how much time it's been idle and active, in seconds",
	}, []string{"state"})

	prometheus.MustRegister(httpRequestsCounter)
}
