FROM cgr.dev/chainguard/bash@sha256:1c8b9cd1227a63a04b3e6f7807fcd7f8868fe99bc842866afef8250fbb0e003a

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
