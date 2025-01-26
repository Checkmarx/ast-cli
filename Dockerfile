FROM checkmarx/bash:5.2.37-r2-ef73fbf0f86d3b@sha256:ef73fbf0f86d3b0f1b9d0af383939a482f9ec0b0227fc5a330c70753f2e1da75
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
