FROM checkmarx/bash:5.3-r14-365aa4c3c7b3fc@sha256:365aa4c3c7b3fcb580687c427e90ee9cda26e75cd12cae3a2b521777c0b4d3f3
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
