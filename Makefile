.PHONY: build test clean dist

build:
	go build -o docshub ./cmd/docshub

test:
	go test ./... -v

clean:
	rm -f docshub dist/*

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION)

dist: clean
	@mkdir -p dist
	@echo "Building linux/amd64..."    && CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-linux-amd64     ./cmd/docshub
	@echo "Building linux/arm64..."    && CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-linux-arm64     ./cmd/docshub
	@echo "Building darwin/amd64..."   && CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-darwin-amd64    ./cmd/docshub
	@echo "Building darwin/arm64..."   && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-darwin-arm64    ./cmd/docshub
	@echo "Building windows/amd64..."  && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-windows-amd64.exe ./cmd/docshub
	@echo "Building windows/arm64..."  && CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/docshub-$(VERSION)-windows-arm64.exe ./cmd/docshub
	@cd dist && shasum -a 256 * > checksums-sha256.txt
	@echo "Done. Artifacts in dist/"
