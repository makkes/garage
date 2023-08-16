GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7
BUILD_ARGS ?=
IMG ?= ghcr.io/makkes/garage
TAG ?= latest

.PHONY: test
test:
	go test -v -race -coverprofile cover.out ./...

RUN_OPTS ?=
.PHONY: run
run:
	go run -race ./cmd/garage/main.go $(RUN_OPTS)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: docker-build
docker-build:
	docker buildx build \
		--platform=$(BUILD_PLATFORMS) \
		-t $(IMG):$(TAG) \
		$(BUILD_ARGS) .
