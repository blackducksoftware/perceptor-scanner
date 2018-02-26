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

import "github.com/blackducksoftware/perceptor-scanner/pkg/common"

type Action interface {
	apply(model *Model)
}

type pullImage struct {
	image        *common.Image
	continuation func(err error)
}

func (p *pullImage) apply(model *Model) {
	err := model.pullImage(p.image)
	go p.continuation(err)
}

type getImage struct {
	image        *common.Image
	continuation func(isDone bool)
}

func newGetImage(image *common.Image, continuation func(isDone bool)) *getImage {
	return &getImage{image: image, continuation: continuation}
}

func (g *getImage) apply(model *Model) {
	isDone := model.isImageDone(g.image)
	go g.continuation(isDone)
}

type finishedImagePull struct {
	image *common.Image
	err   error
}

func (f *finishedImagePull) apply(model *Model) {
	model.finishImagePull(f.image, f.err)
}
