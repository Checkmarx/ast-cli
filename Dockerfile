FROM checkmarx/bash:5.3-r12-aff502cf53c8ca@sha256:aff502cf53c8cacba69bdd43fcad8544324af790240c9025ef081617e836f068
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
