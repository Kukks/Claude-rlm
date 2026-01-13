.PHONY: all build test clean install

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(COMMIT)

all: test build

build:
	@echo "Building RLM $(VERSION)..."
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm ./cmd/rlm

build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm-linux-amd64 ./cmd/rlm
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm-linux-arm64 ./cmd/rlm
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm-darwin-amd64 ./cmd/rlm
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm-darwin-arm64 ./cmd/rlm
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rlm-windows-amd64.exe ./cmd/rlm
	@echo "All builds complete!"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	@echo "Running integration tests..."
	@echo "Starting Qdrant..."
	@docker run -d --name rlm-qdrant-test -p 6334:6334 qdrant/qdrant:latest || true
	@sleep 3
	go test -v -tags=integration ./...
	@docker stop rlm-qdrant-test || true
	@docker rm rlm-qdrant-test || true

lint:
	@echo "Running linter..."
	golangci-lint run || go vet ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/ dist/ coverage.out
	go clean

install:
	@echo "Installing..."
	go install -ldflags="$(LDFLAGS)" ./cmd/rlm

run:
	@echo "Running RLM..."
	go run -ldflags="$(LDFLAGS)" ./cmd/rlm

help:
	@echo "Available targets:"
	@echo "  make build          - Build binary"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make test           - Run tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install binary"
	@echo "  make run            - Run application"
