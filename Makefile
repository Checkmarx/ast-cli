.DEFAULT_GOAL := vet

.PHONY:fmt vet build lint winres
fmt:
	go fmt ./...
vet: fmt
	go vet ./...
build: vet
	go build -o bin/cx.exe ./cmd
lint: fmt
	golangci-lint run -c .golangci.yml
# regenerate the committed Windows version resources (only needed when cmd/winres/winres.json changes)
winres:
	go run github.com/tc-hib/go-winres@v0.3.3 make --in cmd/winres/winres.json --out cmd/rsrc --arch amd64,386,arm64
