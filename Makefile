.DEFAULT_GOAL := vet

.PHONY:fmt vet build lint winres
fmt:
	go fmt ./...
vet: fmt
	go vet ./...
winres:
	go run github.com/tc-hib/go-winres@v0.3.3 make --in cmd/winres/winres.json --out cmd/rsrc --arch amd64 --file-version 0.0.0.0 --product-version 0.0.0.0

build: vet winres
	go build -o bin/cx.exe ./cmd
lint: fmt
	golangci-lint run -c .golangci.yml