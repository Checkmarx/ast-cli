FROM golang:1.17.6

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
