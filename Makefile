.DEFAULT_GOAL := vet

# version embedded in the Windows exe, override with: make winres VERSION=2.3.60
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo 0.0.0)

.PHONY:fmt vet build lint winres
fmt:
	go fmt ./...
vet: fmt
	go vet ./...
build: vet winres
	go build -o bin/cx.exe ./cmd
lint: fmt
	golangci-lint run -c .golangci.yml
# generate the (gitignored) Windows version resources used when building for Windows
winres:
	go tool go-winres make --in cmd/winres/winres.json --out cmd/rsrc --arch amd64,386,arm64 --product-version $(VERSION) --file-version $(VERSION)
