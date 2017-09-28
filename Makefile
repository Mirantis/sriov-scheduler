IMAGE_REPO ?= yashulyak/sriov-scheduler-extender
IMAGE_BRANCH ?= latest

test:
	go test ./pkg/...

deps:
	glide install --strip-vendor

build: vendor/ discovery extender

discovery: vendor/
	CGO_ENABLED=0 go build -o discovery ./cmd/discovery/

extender: vendor/
	CGO_ENABLED=0 go build -o extender ./cmd/discovery/

docker: build
	docker build -t $(IMAGE_REPO):$(IMAGE_BRANCH) .
