FROM golang:1.17.5

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
