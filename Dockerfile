FROM checkmarx.jfrog.io/docker/echo/bash:5.3.20@sha256:24d85520a82edf5d09ab3277123690ddd7e84220e51510529af1e6691880ac49
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
