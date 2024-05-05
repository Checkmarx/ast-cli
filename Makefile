.DEFAULT_GOAL := vet

.PHONY:fmt vet build lint
fmt:
	go fmt ./...
vet: fmt
	go vet ./...
build: vet
	go build -o bin/cx.exe ./cmd
lint: fmt
	golangci-lint run -c .golangci.yml