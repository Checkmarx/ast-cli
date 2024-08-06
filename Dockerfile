FROM cgr.dev/chainguard/bash:5.2.32-r2

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
