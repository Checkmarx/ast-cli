FROM checkmarx/bash-fips:5.2.32-r0@sha256:afc70868d063b0330fc7c52bcb7c874db2e466611745b362b79b4fec3478fa4e

USER root
COPY cx /app/bin/cx

RUN chown 65532:65532 /app

USER 65532

ENTRYPOINT ["/app/bin/cx"]
