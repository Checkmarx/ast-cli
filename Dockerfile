FROM checkmarx/bash:5.2.37-r2-737d21762d3388
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
