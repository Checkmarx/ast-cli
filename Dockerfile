FROM checkmarx/bash:5.3-r5-98621acba7807a@sha256:98621acba7807a4e128f3e00aba3987e4f659ff352191f79cdbaa7f8a32cfb58
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
