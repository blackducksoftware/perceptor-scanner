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

package model

import (
	"fmt"
	"reflect"

	ds "github.com/blackducksoftware/perceptor/pkg/datastructures"
	log "github.com/sirupsen/logrus"
)

// Model is the root of the core model
type Model struct {
	// Pods is a map of qualified name ("<namespace>/<name>") to pod
	Pods                 map[string]Pod
	Images               map[DockerImageSha]*ImageInfo
	Layers               map[string]*LayerInfo
	ImageScanQueue       *ds.PriorityQueue
	ImagePriority        map[DockerImageSha]int
	LayerHubCheckQueue   []string
	LayerRefreshQueue    []string
	LayerRefreshQueueSet map[string]bool
	HubVersion           string
	Config               *Config
	Timings              *Timings
	IsHubEnabled         bool
}

// NewModel .....
func NewModel(hubVersion string, config *Config, timings *Timings) *Model {
	return &Model{
		Pods:                 make(map[string]Pod),
		Images:               make(map[DockerImageSha]*ImageInfo),
		Layers:               make(map[string]*LayerInfo),
		ImageScanQueue:       ds.NewPriorityQueue(),
		ImagePriority:        map[DockerImageSha]int{},
		LayerHubCheckQueue:   []string{},
		LayerRefreshQueue:    []string{},
		LayerRefreshQueueSet: make(map[string]bool),
		HubVersion:           hubVersion,
		Config:               config,
		Timings:              timings,
		IsHubEnabled:         true,
	}
}

// DeletePod removes the record of a pod, but does not affect images.
func (model *Model) DeletePod(podName string) {
	delete(model.Pods, podName)
}

// AddPod adds a pod and all the images in a pod to the model.
// If the pod is already present in the model, it will be removed
// and a new one created in its place.
// The key is the combination of the pod's namespace and name.
// It extracts the containers and images from the pod,
// adding them into the cache.
func (model *Model) AddPod(newPod Pod) {
	log.Debugf("about to add pod: UID %s, qualified name %s", newPod.UID, newPod.QualifiedName())
	if len(newPod.Containers) == 0 {
		recordEvent("adding pod with 0 containers")
		log.Warnf("adding pod %s with 0 containers: %+v", newPod.QualifiedName(), newPod)
	}
	for _, newCont := range newPod.Containers {
		model.AddImage(newCont.Image, 1)
	}
	log.Debugf("done adding containers+images from pod %s -- %s", newPod.UID, newPod.QualifiedName())
	model.Pods[newPod.QualifiedName()] = newPod
}

// AddImage adds an image to the model, adding it to the queue for hub checking.
func (model *Model) AddImage(image Image, priority int) {
	added := model.createImage(image)
	if added {
		model.ImagePriority[image.Sha] = priority
		model.addImageToScanQueue(image.Sha)
	}
}

// layer state transitions

func (model *Model) setLayerScanStatus(sha string, newScanStatus ScanStatus) error {
	layerInfo, ok := model.Layers[sha]
	if !ok {
		return fmt.Errorf("can not set scan status for sha %s, sha not found", sha)
	}

	isLegal := IsLegalTransition(layerInfo.ScanStatus, newScanStatus)
	recordStateTransition(layerInfo.ScanStatus, newScanStatus, isLegal)
	if !isLegal {
		return fmt.Errorf("illegal layer state transition from %s to %s for sha %s", layerInfo.ScanStatus, newScanStatus, sha)
	}

	layerInfo.setScanStatus(newScanStatus)

	return nil
}

// createImage adds the image to the model, but not to the scan queue
func (model *Model) createImage(image Image) bool {
	_, hasImage := model.Images[image.Sha]
	if !hasImage {
		newInfo := NewImageInfo(image.Sha, image.Name)
		model.Images[image.Sha] = newInfo
		log.Debugf("added image %s to model", image.HumanReadableName())
	} else {
		log.Debugf("not adding image %s to model, already have in cache", image.HumanReadableName())
	}
	return !hasImage
}

// Be sure that `sha` is in `model.Images` before calling this method
func (model *Model) unsafeGet(sha DockerImageSha) *ImageInfo {
	results, ok := model.Images[sha]
	if !ok {
		message := fmt.Sprintf("expected to already have image %s, but did not", string(sha))
		log.Error(message)
		panic(message)
	}
	return results
}

func (model *Model) addLayer(layer string, imageSha DockerImageSha) error {
	_, ok := model.Layers[layer]
	if ok {
		log.Debugf("skipping addition of layer %s, already present", layer)
		return nil
	}
	model.Layers[layer] = NewLayerInfo(imageSha)
	return model.addLayerToHubCheckQueue(layer)
}

// Adding and removing from scan queues.  These are "unsafe" calls and should
// only be called by methods that have already checked all the error conditions
// (things are in the right state, things that are expected to be present are
// actually present, etc.)

func (model *Model) addLayerToHubCheckQueue(sha string) error {
	model.LayerHubCheckQueue = append(model.LayerHubCheckQueue, sha)
	return nil
}

func (model *Model) RemoveLayerFromHubCheckQueue(sha string) error {
	index := -1
	for i := 0; i < len(model.LayerHubCheckQueue); i++ {
		if model.LayerHubCheckQueue[i] == sha {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("unable to remove sha %s from hub check queue, not found", string(sha))
	}

	model.LayerHubCheckQueue = append(model.LayerHubCheckQueue[:index], model.LayerHubCheckQueue[index+1:]...)
	return nil
}

func (model *Model) addImageToScanQueue(sha DockerImageSha) error {
	priority := model.ImagePriority[sha]
	return model.ImageScanQueue.Add(string(sha), priority, sha)
}

func (model *Model) removeImageFromScanQueue(sha DockerImageSha) error {
	_, err := model.ImageScanQueue.Remove(string(sha))
	return err
}

// "Public" methods

// SetLayersForImage ...
func (model *Model) SetLayersForImage(imageSha DockerImageSha, layers []string) error {
	// TODO if image not present, could add ...
	imageInfo, ok := model.Images[imageSha]
	if !ok {
		return fmt.Errorf("cannot add layers, image %s not found", imageSha)
	}
	// add layers to image
	err := imageInfo.SetLayers(layers)
	if err != nil {
		return err
	}
	// add layers to global scope
	for _, layer := range layers {
		err = model.addLayer(layer, imageSha)
		if err != nil {
			return err
		}
	}
	// done
	return nil
}

// SetLayerScanStatus .....
func (model *Model) SetLayerScanStatus(sha string, newScanStatus ScanStatus) error {
	err := model.setLayerScanStatus(sha, newScanStatus)
	if err != nil {
		layerInfo, ok := model.Layers[sha]
		statusString := "sha not found"
		if ok {
			statusString = layerInfo.ScanStatus.String()
		}
		log.Errorf("unable to transition layer state for sha %s from <%s> to %s", sha, statusString, newScanStatus)
	}
	return err
}

// GetNextLayerFromHubCheckQueue .....
func (model *Model) GetNextLayerFromHubCheckQueue() *string {
	if len(model.LayerHubCheckQueue) == 0 {
		log.Debug("hub check queue empty")
		return nil
	}

	first := model.LayerHubCheckQueue[0]
	return &first
}

// ShouldScanLayer .....
func (model *Model) ShouldScanLayer(layer string) (ShouldScanLayer, error) {
	layerInfo, ok := model.Layers[layer]
	if !ok {
		return ShouldScanLayerNo, fmt.Errorf("layer %s not found", layer)
	}

	if !model.IsHubEnabled {
		log.Debugf("Hub not enabled, can't start a new scan")
		return ShouldScanLayerWait, nil
	}

	if model.InProgressScanCount() >= model.Config.ConcurrentScanLimit {
		log.Debugf("max concurrent scan count reached, can't start a new scan -- %v", model.InProgressScans())
		return ShouldScanLayerWait, nil
	}

	switch layerInfo.ScanStatus {
	case ScanStatusUnknown:
		return ShouldScanLayerWait, nil
	case ScanStatusNotScanned:
		return ShouldScanLayerYes, nil
	default:
		return ShouldScanLayerNo, nil
	}
}

// GetNextImageFromScanQueue .....
func (model *Model) GetNextImageFromScanQueue() *Image {
	if !model.IsHubEnabled {
		log.Debugf("Hub not enabled, can't start a new scan")
		return nil
	}

	if model.InProgressScanCount() >= model.Config.ConcurrentScanLimit {
		log.Debugf("max concurrent scan count reached, can't start a new scan -- %v", model.InProgressScans())
		return nil
	}

	if model.ImageScanQueue.IsEmpty() {
		log.Debug("scan queue empty, can't start a new scan")
		return nil
	}

	first, err := model.ImageScanQueue.Pop()
	if err != nil {
		log.Errorf("unable to get next image from scan queue: %s", err.Error())
		return nil
	}

	switch sha := first.(type) {
	case DockerImageSha:
		image := model.unsafeGet(sha).Image()
		return &image
	default:
		log.Errorf("expected type DockerImageSha from priority queue, got %s", reflect.TypeOf(first))
		return nil
	}
}

// AddLayerToRefreshQueue .....
func (model *Model) AddLayerToRefreshQueue(sha string) error {
	layerInfo, ok := model.Layers[sha]
	if !ok {
		return fmt.Errorf("expected to already have layer %s, but did not", sha)
	}

	if layerInfo.ScanStatus != ScanStatusComplete {
		return fmt.Errorf("unable to refresh layer %s, scan status is %s", sha, layerInfo.ScanStatus.String())
	}

	// if it's already in the refresh queue, don't add it again
	_, ok = model.LayerRefreshQueueSet[sha]
	if ok {
		return fmt.Errorf("unable to add layer %s to refresh queue, already in queue", sha)
	}

	model.LayerRefreshQueue = append(model.LayerRefreshQueue, sha)
	model.LayerRefreshQueueSet[sha] = false
	return nil
}

// GetNextLayerFromRefreshQueue .....
func (model *Model) GetNextLayerFromRefreshQueue() *string {
	if len(model.LayerRefreshQueue) == 0 {
		log.Debug("refresh queue empty")
		return nil
	}

	first := model.LayerRefreshQueue[0]
	return &first
}

// RemoveLayerFromRefreshQueue .....
func (model *Model) RemoveLayerFromRefreshQueue(sha string) error {
	index := -1
	for i := 0; i < len(model.LayerRefreshQueue); i++ {
		if model.LayerRefreshQueue[i] == sha {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("unable to remove sha %s from refresh queue, not found", string(sha))
	}

	model.LayerRefreshQueue = append(model.LayerRefreshQueue[:index], model.LayerRefreshQueue[index+1:]...)
	delete(model.LayerRefreshQueueSet, sha)
	return nil
}

// FinishRunningScanClient .....
func (model *Model) FinishRunningScanClient(sha string, scanClientError error) {
	_, ok := model.Layers[sha]

	// TODO if we don't have this sha already ... bail out?
	if !ok {
		log.Errorf("finish running scan client -- expected to already have layer %s, but did not", sha)
		return
	}

	scanStatus := ScanStatusRunningHubScan
	if scanClientError != nil {
		scanStatus = ScanStatusNotScanned
		log.Errorf("error running scan client -- %s", scanClientError.Error())
	}

	model.setLayerScanStatus(sha, scanStatus)
}

// additional methods

// InProgressScans .....
func (model *Model) InProgressScans() []string {
	inProgressShas := []string{}
	for sha, results := range model.Layers {
		switch results.ScanStatus {
		case ScanStatusRunningScanClient, ScanStatusRunningHubScan:
			inProgressShas = append(inProgressShas, sha)
		default:
			break
		}
	}
	return inProgressShas
}

// InProgressScanCount .....
func (model *Model) InProgressScanCount() int {
	return len(model.InProgressScans())
}

// InProgressHubScans .....
func (model *Model) InProgressHubScans() *([]string) {
	inProgressHubScans := []string{}
	for sha, results := range model.Layers {
		switch results.ScanStatus {
		case ScanStatusRunningHubScan:
			inProgressHubScans = append(inProgressHubScans, sha)
		}
	}
	return &inProgressHubScans
}
