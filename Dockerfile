FROM golang:1.18.1

RUN useradd -r -m cxuser
USER cxuser
COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
