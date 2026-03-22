BINARY  := opsdeck
PKG     := github.com/opsdeck/opsdeck/cmd/opsdeck
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build test install lint clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) $(PKG)

test:
	go test -race -count=1 ./...

install:
	go install -ldflags "-s -w -X main.version=$(VERSION)" $(PKG)

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
