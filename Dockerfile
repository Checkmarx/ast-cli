FROM cgr.dev/chainguard/bash@sha256:2faccc3e8ab049d82dec0e4d2dd8b45718c71ce640608584d95a39092b5006b5

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
