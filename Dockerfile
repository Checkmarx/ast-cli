FROM golang:alpine

RUN adduser -S -D cxuser
USER cxuser && chown -R cxuser /app
COPY /bin/cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
