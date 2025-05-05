FROM checkmarx/bash:5.2.37-r30-0714eec7a3fa2e@sha256:0714eec7a3fa2eadb3a6bdf2049bc158cc0311182a2475e8a467dbb2834df23f
USER nonroot

COPY cx /app/bin/cx


ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
