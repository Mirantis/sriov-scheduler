IMAGE_REPO ?= yashulyak/sriov-scheduler-extender
IMAGE_BRANCH ?= latest
HOME ?= /home

deps vendor/:
	go get github.com/Masterminds/glide
	glide install --strip-vendor

test: vendor/
	go test ./pkg/...

build: vendor/ discovery extender

discovery: vendor/
	CGO_ENABLED=0 go build -o discovery ./cmd/discovery/

extender: vendor/
	CGO_ENABLED=0 go build -o extender ./cmd/extender/

docker: build
	docker build -t $(IMAGE_REPO):$(IMAGE_BRANCH) .

e2e e2e.test:
	go test -c -o e2e.test ./tests/

import:
	IMAGE_REPO=$(IMAGE_REPO) IMAGE_BRANCH=$(IMAGE_BRANCH) ./utils/import.sh

run-e2e: e2e.test
	./e2e.test -deployments=./tools/ -kubeconfig=$(HOME)/.kube/config 

clean: 
	-rm discovery
	-rm extender
	-rm e2e.test

clean-k8s:
	-./utils/clean.sh

ci: clean-k8s clean docker import run-e2e

docker-push:
	docker push $(IMAGE_REPO):$(IMAGE_BRANCH)

quick-release: docker docker-push

release: ci docker-push
