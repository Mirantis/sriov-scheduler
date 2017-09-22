test:
	go test ./pkg/...

deps:
	glide install --strip-vendor

build: vendor/
	CGO_ENABLED=0 go build -o discovery ./cmd/discovery/
