FROM cgr.dev/chainguard/bash@sha256:4eb8145143515a9be1d04b90e911431b3e48b74ddf62948c516d300c453c845f
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
