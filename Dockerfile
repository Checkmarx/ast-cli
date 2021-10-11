FROM golang:1.17.2

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
