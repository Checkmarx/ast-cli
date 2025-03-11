FROM checkmarx/bash:5.2.37-r2-c5dcfc6a2fbe1c@sha256:c5dcfc6a2fbe1c8f9d11bdf902b5485bb78b4733864a99806749d5e244a6b75e
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
