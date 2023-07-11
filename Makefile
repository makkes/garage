GORELEASER_DEBUG ?= false
GORELEASER_PARALLELISM ?= $(shell nproc --ignore=1)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

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


.PHONY: build-snapshot
build-snapshot:
	goreleaser --debug=$(GORELEASER_DEBUG) \
		build \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--single-target \
		--skip-post-hooks

.PHONY: release-snapshot
release-snapshot:
	goreleaser --debug=$(GORELEASER_DEBUG) \
		release \
		--snapshot \
		--clean \
		--parallelism=$(GORELEASER_PARALLELISM) \
		--skip-publish
