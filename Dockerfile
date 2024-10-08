FROM cgr.dev/chainguard/bash@sha256:f8e48690d991e6814c81f063833176439e8f0d4bc1c5f0a47f94858dea3e4f44
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
