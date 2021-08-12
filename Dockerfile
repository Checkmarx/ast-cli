FROM golang:1.16.7-alpine3.14

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
