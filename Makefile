BINARY   := ssx
BUILD_DIR := bin
GO        := go
MODULE    := github.com/highfredo/ssx

.PHONY: build run clean tidy vet fmt help

## build: compile the binary to bin/ssx
build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/$(BINARY)

## run: build then run directly
run: build
	./$(BUILD_DIR)/$(BINARY)

## tidy: update go.mod / go.sum
tidy:
	$(GO) mod tidy

## vet: run go vet across all packages
vet:
	$(GO) vet ./...

## fmt: format all Go source files
fmt:
	$(GO) fmt ./...

## clean: remove compiled artifacts
clean:
	rm -rf $(BUILD_DIR)

## snapshot: build a local snapshot with goreleaser (no git tag required)
snapshot:
	goreleaser release --snapshot --clean

## release-dry: dry-run a release with goreleaser
release-dry:
	goreleaser release --skip=publish --clean

## release: publish a release (requires GITHUB_TOKEN and a git tag)
release:
	goreleaser release --clean

## help: list available targets
help:
	@grep -E '^##' Makefile | sed 's/## //' | column -t -s ':'
