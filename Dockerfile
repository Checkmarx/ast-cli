FROM golang:alpine3.19 as build-env

RUN apk add --no-cache bash
RUN adduser --system --disabled-password cxuser

# Copy the source from the current directory to the Working Directory inside the container
WORKDIR /app
# Copy the Go module files
COPY go.mod go.sum ./
# Download the Go module dependencies
RUN go mod download
# Copy the application source code
COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o bin/cx cmd/main.go

FROM alpine:3.19.0
RUN apk add --update docker openrc
RUN rc-update add docker boot
COPY --from=build-env /app/bin/cx /app/bin/cx
#ENTRYPOINT ["tail", "-f", "/dev/null"]
ENTRYPOINT ["/app/bin/cx"]
