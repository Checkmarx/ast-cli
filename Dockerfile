FROM golang:1.16.6

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
