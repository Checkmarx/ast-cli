FROM golang:1.17.1

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
