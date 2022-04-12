FROM golang:1.17.8

RUN useradd -r -m cxuser
USER cxuser
COPY /bin/cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
