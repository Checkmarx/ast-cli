FROM cgr.dev/chainguard/bash:37585d36bbc654f9dab18771bc332691cb73710a288c817d3c0126396744fb4f

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
