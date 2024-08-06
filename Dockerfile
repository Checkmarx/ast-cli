FROM cgr.dev/chainguard/bash:5.2.21

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
