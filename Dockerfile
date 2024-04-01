FROM checkmarx.jfrog.io/docker/chainguard/go:1.22.1-r1--ed1afb92146ee7

RUN adduser --system --disabled-password cxuser
USER cxuser

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
