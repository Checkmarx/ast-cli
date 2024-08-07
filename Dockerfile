FROM cgr.dev/chainguard/bash@sha256:e9ef27c933aca00e5264240ffb386f99c28b4055a98ccf696d2d83858e83f2ee

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
