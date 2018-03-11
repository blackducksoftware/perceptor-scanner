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
	"encoding/json"
	"testing"

	"github.com/blackducksoftware/perceptor/pkg/core"
	log "github.com/sirupsen/logrus"
)

func TestPifTester(t *testing.T) {
	pif := NewPifTester(3114)
	pif.addImage(core.Image{Name: "abc", Sha: "123"})
	jsonBytes, err := json.Marshal(pif)
	if err != nil {
		t.Errorf("unable to produce json: %s", err.Error())
	}
	log.Info(string(jsonBytes))
}
