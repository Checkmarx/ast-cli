FROM cgr.dev/chainguard/bash@sha256:1abc09ac352efdc60d855bd159b9b66df6596a174400752ae3c537b5350779a9
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
