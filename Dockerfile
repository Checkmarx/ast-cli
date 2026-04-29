FROM checkmarx/bash:5.3-r12-0e56cb6e000601@sha256:0e56cb6e000601d35ed11ddcc973ca268c431a176be53cdc31bc85f3208dc44a
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
