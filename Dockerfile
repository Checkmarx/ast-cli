FROM checkmarx/bash:5.3-r14-ca00804d797fe8@sha256:ca00804d797fe8c0846a2f1ec4fc159bd08fdf66a6c8621031c954d1bee61193
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
