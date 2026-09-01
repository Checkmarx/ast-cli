FROM checkmarx/bash:5.3-r12-aff502cf53c8ca@sha256:72d2a5e2ffb428bab97e6bb154f135f0f9638436605178f06806d367fb8d8601
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
