FROM checkmarx/bash:5.3-r12-f48dd8a45af577@sha256:f48dd8a45af5771e98cb5d56d204ada0e0dc045093ca3272b4c3dbe3f85e6e4f
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
