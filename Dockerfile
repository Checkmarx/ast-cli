FROM alpine:3.17.3

RUN apk add --no-cache libstdc++ glib krb5 pcre bash 
RUN adduser --system --disabled-password cxuser
USER cxuser

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
