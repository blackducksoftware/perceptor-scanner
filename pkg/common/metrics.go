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

package common

import (
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var durationsHistogram *prometheus.HistogramVec
var errorsCounter *prometheus.CounterVec

func RecordError(errorStage string, errorName string) {
	errorsCounter.With(prometheus.Labels{"stage": errorStage, "errorName": errorName}).Inc()
}

func RecordDuration(operation string, duration time.Duration) {
	durationsHistogram.With(prometheus.Labels{"operation": operation}).Observe(duration.Seconds())
}

func init() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	durationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "timings",
			Help:      "time durations of scanner operations",
			Buckets:   prometheus.ExponentialBuckets(0.25, 2, 20),
		},
		[]string{"operation"})

	errorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "scanner",
		Name:      "scannerErrors",
		Help:      "error codes from image pulling and scanning",
	}, []string{"stage", "errorName"})

	prometheus.MustRegister(errorsCounter)
	prometheus.MustRegister(durationsHistogram)
}
