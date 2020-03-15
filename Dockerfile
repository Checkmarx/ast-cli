FROM golang:1.13.7-alpine3.11 as build-env

WORKDIR /app

#Copy go mod and sum files
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# COPY the source code as the last step
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/ast-cli cmd/main.go cmd/config.go

#runtime image
FROM scratch

COPY --from=build-env /app/bin/ast-cli /app/bin/ast-cli

ENTRYPOINT ["/app/bin/ast-cli"]