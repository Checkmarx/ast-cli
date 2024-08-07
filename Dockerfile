FROM cgr.dev/chainguard/bash@sha256:24162d1b30cd7a9bf1aab85544074513bc45a0b1f9abf6661d1a1337b9592c48

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
