.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: build-arm
build-arm:
	CGO=0 GOOS=linux GOARCH=arm GOARM=5 go build -mod=vendor ./cmd/power-logger
