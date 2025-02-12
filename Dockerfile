FROM checkmarx/bash:5.2.37-r2-cbecd9aeaadc77@sha256:cbecd9aeaadc775906af3b4b0b03e05d5a4e68cb300d7db4579d88129b2eb028
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE


