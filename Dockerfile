FROM cgr.dev/chainguard/bash:latest

USER root

# Update the package list and upgrade curl to the fixed version
RUN apk update && \
    apk add curl=8.9.1-r0 && \
    apk clean && \
    rm -rf /var/cache/apk/*

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
