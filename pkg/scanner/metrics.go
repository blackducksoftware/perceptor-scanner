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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var httpResults *prometheus.CounterVec
var durationsHistogram *prometheus.HistogramVec
var errorsCounter *prometheus.CounterVec

// helpers

func recordError(errorStage string, errorName string) {
	errorsCounter.With(prometheus.Labels{"stage": errorStage, "errorName": errorName}).Inc()
}

func recordDuration(operation string, duration time.Duration) {
	durationsHistogram.With(prometheus.Labels{"operation": operation}).Observe(duration.Seconds())
}

// recorders

func recordHttpStats(path string, statusCode int) {
	httpResults.With(prometheus.Labels{"path": path, "code": fmt.Sprintf("%d", statusCode)}).Inc()
}

func recordScanClientDuration(duration time.Duration, isSuccess bool) {
	operation := "scan client success"
	if !isSuccess {
		operation = "scan client error"
	}
	recordDuration(operation, duration)
}

func recordTotalScannerDuration(duration time.Duration, isSuccess bool) {
	operation := "scanner total success"
	if !isSuccess {
		operation = "scanner total error"
	}
	recordDuration(operation, duration)
}

func recordScannerError(errorName string) {
	recordError("scan client", errorName)
}

func recordCleanUpTarFile(isSuccess bool) {
	if !isSuccess {
		recordScannerError("clean up tar file")
	}
	// TODO should we have a metric for success?
}

// init

func init() {
	httpResults = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "scanner",
		Name:      "http_response_status_codes",
		Help:      "status codes for responses from HTTP requests issued by scanner",
	},
		[]string{"path", "code"})

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
		Help:      "error codes from image scanning",
	}, []string{"stage", "errorName"})

	prometheus.MustRegister(errorsCounter)
	prometheus.MustRegister(durationsHistogram)
	prometheus.MustRegister(httpResults)
}
