FROM checkmarx/bash:5.2.37-r32-044701a6758b91@sha256:044701a6758b91913c1e6435723becfce973ce727baf76ecb0add2340e5aeb25
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
