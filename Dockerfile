FROM checkmarx/bash:5.2.37-r2-737d21762d3388@sha256:631b3b846c1744ff5ddc7114f8df0eaf771f0e8bea3acea05155e8e0caa8532e
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
