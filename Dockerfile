FROM checkmarx/bash:5.3-r14-e52a196c3a914c@sha256:e52a196c3a914c6496c72465ad4a8f7efd1ff8ad988b328cbb92018652c23c7d
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
