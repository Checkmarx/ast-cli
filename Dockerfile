FROM checkmarx.jfrog.io/ast-docker/chainguard/bash:latest

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
