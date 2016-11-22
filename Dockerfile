FROM alpine
MAINTAINER Steve Sloka <steve@stevesloka.com>

RUN apk add --update ca-certificates && rm -rf /var/cache/apk/*

ADD certs/ certs/
ADD emmie emmie
ENTRYPOINT ["/emmie"]
