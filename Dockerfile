FROM checkmarx/bash-fips:5.2.32-r0@sha256:e5f5d689936ae073dd3f1b8d5f32700bd739ca3c0a73b8507c561e2b8d3d4e27
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
