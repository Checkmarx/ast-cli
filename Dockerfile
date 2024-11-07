FROM cgr.dev/chainguard/bash@sha256:e1d16dec8d976859080d984167109b3557c2b6494f10be08147806b78bdef691
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
