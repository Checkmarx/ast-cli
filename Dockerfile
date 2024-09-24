FROM checkmarx/bash-fips:golden-amd64@sha256:7e22d30bfdcec9c4acfc5616aadcc166e085633c0ab9b73266878acc610f22e6

USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
