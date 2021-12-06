FROM golang:1.17.4

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
