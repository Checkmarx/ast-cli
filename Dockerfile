FROM golang:1.16.7

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
