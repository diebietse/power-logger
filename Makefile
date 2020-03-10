.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: test
test:
	go test -v -race -mod=vendor -cover ./...

.PHONY: build-arm
build-arm:
	CGO=0 GOOS=linux GOARCH=arm GOARM=5 go build -mod=vendor -o bin/power-logger-arm ./cmd/power-logger

.PHONY: build-x64
build-x64:
	CGO=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -o bin/power-logger-x64 ./cmd/power-logger

.PHONY: gofmt
gofmt:
		gofmt -l -s -w ./cmd ./logger

.PHONY: lint
lint:
	docker run --rm -it \
		-w /src -v $(shell pwd):/src \
		golangci/golangci-lint:v1.23 golangci-lint run \
		-v -c .golangci.yml

.PHONY: build-all
build-all: build-x64 build-arm
