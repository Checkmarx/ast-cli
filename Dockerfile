FROM checkmarx.jfrog.io/ast-docker/library/alpine:3.13.1

RUN apk add bash
RUN adduser --system --disabled-password cxuser
USER cxuser

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
