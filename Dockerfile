FROM checkmarx/bash:5.3-r13-f99a9136837d1b@sha256:63716ceab566d449b6c1de5032f0ca1e6da5ffa8627b419941f07b35451e3626
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
