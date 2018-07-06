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
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackducksoftware/perceptor-scanner/pkg/docker"

	log "github.com/sirupsen/logrus"
)

type image struct {
	name string
	path string
}

func (i *image) DockerPullSpec() string {
	return i.name
}

func (i *image) DockerTarFilePath() string {
	return i.path
}

func main() {
	ip := docker.NewImagePuller([]docker.RegistryAuth{})
	// 1. export image
	path := os.Args[1]
	if len(os.Args) >= 3 {
		name := os.Args[2]
		image := &image{name: name, path: path}
		err := ip.SaveImageToTar(image)
		if err != nil {
			panic(err)
		}
	}
	// 2. extract tar
	processFile(path, "extract-")
	// 3. run sha over everything
	hasher := sha256.New()
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		panic(err)
	}
	shaBytes := hasher.Sum(nil)
	sha := hex.EncodeToString(shaBytes)
	os.Stdout.WriteString(sha + "\n")
	fmt.Println(sha)
}

func processFile(source string, name string) {
	f, err := os.Open(source)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	tarReader := tar.NewReader(f)

	for i := 0; ; i++ {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}

		log.Infof("got: %+v", header)
		if strings.Contains(header.Name, ".json") {
			file, err := os.Create(fmt.Sprintf("%sfile%d.json", name, i))
			if err != nil {
				panic(err)
			}
			defer file.Close()
			if _, err := io.Copy(file, tarReader); err != nil {
				panic(err)
			}
		} else if strings.Contains(header.Name, "layer.tar") {
			dir := "out/" + header.Name[:len(header.Name)-9]
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				panic(err)
			}
			// make a copy of the file
			out, err := os.Create("out/" + header.Name)
			if err != nil {
				panic(err)
			}
			defer out.Close()
			if _, err := io.Copy(out, tarReader); err != nil {
				panic(err)
			}
			// calculate a sha
			in, err := os.Open("out/" + header.Name)
			if err != nil {
				panic(err)
			}
			defer in.Close()
			hasher := sha256.New()
			// hasher := sha512.New512_224() // TODO which algorithm?
			if _, err := io.Copy(hasher, in); err != nil {
				panic(err)
			}
			shaBytes := hasher.Sum(nil)
			sha := hex.EncodeToString(shaBytes)
			fmt.Printf("sha: %s\n", sha)
			shaFile, err := os.Create(dir + "sha.txt")
			if err != nil {
				panic(err)
			}
			defer shaFile.Close()
			if _, err := io.Copy(shaFile, bytes.NewBufferString(sha)); err != nil {
				panic(err)
			}
		}
		//      fmt.Println()
	}
}
