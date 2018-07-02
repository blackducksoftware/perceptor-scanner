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
	"testing"
)

type testImage struct {
	pullSpec     string
	registryAuth *RegistryAuth
}

func (ti *testImage) DockerPullSpec() string {
	return ti.pullSpec
}

func (ti *testImage) DockerTarFilePath() string {
	return "TODO"
}

func TestNeedsAuthHeader(t *testing.T) {
	internalDockerRegistries := []RegistryAuth{
		{Url: "abc.def:5000", User: "", Password: ""},
		{Url: "docker-registry.default.svc:5000", User: "", Password: ""},
		{Url: "172.1.1.0:abcd", User: "", Password: ""},
	}
	testCases := []*testImage{
		{
			pullSpec: "", registryAuth: nil,
		},
		{
			pullSpec: "abc.def:5000/qqq", registryAuth: &internalDockerRegistries[0],
		},
		{
			pullSpec: "docker-registry.default.svc:5000/ttt", registryAuth: &internalDockerRegistries[1],
		},
		{
			pullSpec: "172.1.1.0:abcd/abc", registryAuth: &internalDockerRegistries[2],
		},
		{
			pullSpec: "172.1.1.0:abc/abc", registryAuth: nil,
		},
	}
	for _, testCase := range testCases {
		registryAuth := needsAuthHeader(testCase, internalDockerRegistries)
		if registryAuth != testCase.registryAuth {
			t.Errorf("expected %+v for %s, got %+v", testCase.registryAuth, testCase.pullSpec, registryAuth)
		}
	}
}
