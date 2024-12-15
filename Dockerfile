FROM checkmarx.jfrog.io/ast-docker/chainguard/bash:5.2.37-r2@sha256:423efc2e971a0dc620ab63e87a5b2577a32083563ff7290380f62e02aa7e0d2e
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
