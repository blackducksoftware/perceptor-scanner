COMPONENTS = mockimagefacade perceptor-imagefacade perceptor-scanner piftester

ifndef REGISTRY
REGISTRY=gcr.io/gke-verification
endif

ifdef IMAGE_PREFIX
PREFIX="$(IMAGE_PREFIX)-"
endif

ifneq (, $(findstring gcr.io,$(REGISTRY)))
PREFIX_CMD="gcloud"
DOCKER_OPTS="--"
endif

CURRENT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
OUTDIR=_output

.PHONY: test ${OUTDIR} ${COMPONENTS}

all: compile

compile: ${OUTDIR} ${COMPONENTS}

${COMPONENTS}:
	docker run -t -i --rm -v ${CURRENT_DIR}:/go/src/github.com/blackducksoftware/perceptor-scanner/ -w /go/src/github.com/blackducksoftware/perceptor-scanner/cmd/$@ -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.11 go build -o $@
	cp cmd/$@/$@ $(OUTDIR)

container: $(COMPONENTS)
	$(foreach p,${COMPONENTS},cd ${CURRENT_DIR}/cmd/$p; docker build -t $(REGISTRY)/$(PREFIX)${p} .;)

push: container
	$(foreach p,${COMPONENTS},$(PREFIX_CMD) docker $(DOCKER_OPTS) push $(REGISTRY)/$(PREFIX)${p}:latest;)

test:
	docker run -t -i --rm -v ${CURRENT_DIR}:/go/src/github.com/blackducksoftware/perceptor-scanner/ -w /go/src/github.com/blackducksoftware/perceptor-scanner -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.11 go test ./pkg/...

clean:
	rm -rf ${OUTDIR}
	$(foreach p,${COMPONENTS},rm -f cmd/$p/$p;)

${OUTDIR}:
	mkdir -p ${OUTDIR}

lint:
	./hack/verify-gofmt.sh
	./hack/verify-golint.sh
	./hack/verify-govet.sh
