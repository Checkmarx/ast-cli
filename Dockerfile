FROM golang:1.15.10-alpine3.13 as build-env
ARG GIT_TOKEN

ENV GOPRIVATE=github.com/checkmarxDev/*,github.com/CheckmarxDev/*

# Copy the source from the current directory to the Working Directory inside the container
WORKDIR /app

RUN apk add --no-cache git \
  && git config \
  --global \
  url."https://api:${GIT_TOKEN}@github.com".insteadOf \
  "https://github.com"

#Copy go mod and sum files
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# COPY the source code as the last step
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/cx cmd/main.go

#runtime image
FROM golang:1.15.10-alpine3.13

COPY --from=build-env /app/bin/cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
