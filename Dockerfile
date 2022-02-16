FROM golang:1.17.7

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
