FROM alpine:3.18.0

RUN apk add --no-cache bash
RUN adduser --system --disabled-password cxuser
USER root

# Install dependencies
#RUN apk update && \
#    apk add --no-cache curl \
#
#RUN curl -fsSL https://get.docker.com -o get-docker.sh
#RUN sh get-docker.sh

RUN apk add --update docker openrc
RUN rc-update add docker boot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]
