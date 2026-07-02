FROM checkmarx/bash:5.3-r12-fd4144660b936c@sha256:fd4144660b936cfa93aaf980ff81eaa13aff00cb420e4b115f39fc251bfd86e1
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
