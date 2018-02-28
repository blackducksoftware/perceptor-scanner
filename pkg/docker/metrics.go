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

package docker

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var tarballSize *prometheus.HistogramVec
var durationsHistogram *prometheus.HistogramVec
var errorsCounter *prometheus.CounterVec

// durations

func recordDuration(operation string, duration time.Duration) {
	durationsHistogram.With(prometheus.Labels{"operation": operation}).Observe(duration.Seconds())
}

func recordDockerCreateDuration(duration time.Duration) {
	recordDuration("docker create", duration)
}

func recordDockerGetDuration(duration time.Duration) {
	recordDuration("docker save", duration)
}

func recordDockerTotalDuration(duration time.Duration) {
	recordDuration("docker get image total", duration)
}

// tar file size and docker errors

func recordTarFileSize(fileSizeMBs int) {
	tarballSize.WithLabelValues("tarballSize").Observe(float64(fileSizeMBs))
}

func recordDockerError(errorStage string, errorName string, image Image, err error) {
	// TODO what use can be made of `image` and `err`?
	// we might want to group the errors by image sha or something
	errorsCounter.With(prometheus.Labels{"stage": errorStage, "errorName": errorName}).Inc()
}

// init

func init() {
	tarballSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "imagefacade",
			Name:      "tarballsize",
			Help:      "tarball file size in MBs",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		},
		[]string{"tarballSize"})

	durationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "imagefacade",
			Name:      "timings",
			Help:      "time durations of scanner operations",
			Buckets:   prometheus.ExponentialBuckets(0.25, 2, 20),
		},
		[]string{"operation"})

	errorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "imagefacade",
		Name:      "dockerErrors",
		Help:      "error codes from image pulling from docker",
	}, []string{"stage", "errorName"})

	prometheus.MustRegister(errorsCounter)
	prometheus.MustRegister(durationsHistogram)
	prometheus.MustRegister(tarballSize)
}
