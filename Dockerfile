FROM checkmarx/bash:5.3-r12-02a1aad732e7ab@sha256:02a1aad732e7ab0659b212d83c2a0bb548d9d8bdec23336f6c0b44f8f3435cb8
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
