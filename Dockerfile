FROM checkmarx/bash:5.3-r14-fad83378373438@sha256:fad833783734389f27b9d8ab3b5e9dda6eb44fdf1892145c70058cb774da817d
USER nonroot

COPY cx /app/bin/cx 
 
ENTRYPOINT ["/app/bin/cx"] 

HEALTHCHECK NONE
