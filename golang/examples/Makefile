# Makefile — короткие имена для частых команд. Отступы — табами.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all fmt lint test build docker vuln

all: fmt lint test build

fmt:
	gofmt -l -w .
	go fix ./...

lint:
	go vet ./...
	golangci-lint run ./...

test:
	go test -race -count=1 -shuffle=on ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/todo ./httpapi/cmd/todo

docker:
	docker build -f httpapi/Dockerfile --build-arg VERSION=$(VERSION) -t todo:$(VERSION) .

vuln:
	go tool govulncheck ./...    # после: go get -tool golang.org/x/vuln/cmd/govulncheck
