FROM cgr.dev/chainguard/bash@sha256:27dc752a2ebacd10571c4045d3e1732f4bbb764446373ac85626602b69132776

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
