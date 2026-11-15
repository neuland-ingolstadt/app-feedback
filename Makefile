MODULE := github.com/neuland-ingolstadt/app-feedback
BIN := bin/server
IMG ?= ghcr.io/neuland-ingolstadt/app-feedback
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test tidy fmt vet docker-build

all: tidy fmt vet test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o $(BIN) ./cmd/server

test:
	go test ./... -race -count=1

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$$(git rev-parse HEAD 2>/dev/null || echo unknown) -t $(IMG):$(VERSION) .
