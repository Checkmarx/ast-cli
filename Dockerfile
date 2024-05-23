FROM cgr.dev/chainguard/bash:latest

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
