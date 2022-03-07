FROM golang:1.17.8

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
