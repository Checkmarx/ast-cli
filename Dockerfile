FROM cgr.dev/chainguard/bash:4133d46b21c513947d6a36f21b7a361943d83f05e7c9b1928c66f43bb1d88725

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
