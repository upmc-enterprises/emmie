FROM scratch
MAINTAINER Steve Sloka <steve@stevesloka.com>
ADD certs/ certs/
ADD emmie emmie
ENTRYPOINT ["/emmie"]
