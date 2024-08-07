FROM checkmarx.jfrog.io/cgr-public/chainguard/bash@sha256:6f0c9e28cbbe206781cb6b0ace299d1d4edbb2450bfadffb8b2e125596d0f6b0

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
