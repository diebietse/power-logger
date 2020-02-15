.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: test
test:
	go test -v -race -mod=vendor ./...

.PHONY: build-arm
build-arm:
	CGO=0 GOOS=linux GOARCH=arm GOARM=5 go build -mod=vendor -o power-logger-arm ./cmd/power-logger

.PHONY: build-x64
build-x64:
	CGO=0 GOOS=linux GOARCH=amd64 GOARM=5 go build -mod=vendor -o power-logger-x64 ./cmd/power-logger

.PHONY: build-all
build-all: build-x64 build-arm
