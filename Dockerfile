FROM golang:1.17.3

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
