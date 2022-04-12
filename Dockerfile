FROM golang:alpine

RUN adduser -S -D cxuser
USER cxuser
COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
