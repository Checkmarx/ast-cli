FROM cgr.dev/chainguard/bash:9ad7b9ca9a929ebc7e5c8340367797fb3b228341177447e37208df77e56a1838

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
