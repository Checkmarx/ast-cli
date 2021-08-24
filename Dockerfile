FROM golang:1.17.0

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
