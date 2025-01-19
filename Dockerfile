FROM checkmarx/bash:5.2.37-r2-737d21762d3388@sha256:737d21762d3388d14fd47973cd2e6c1505fd1942b5347c07c7dd8de3305cbbee
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
