# Makefile for the Docker image stevesloka/emmie
# MAINTAINER: Steve Sloka <steve@stevesloka.com>
# If you update this image please bump the tag value before pushing.

.PHONY: all emmie container push clean test

TAG = 0.0.2
PREFIX = stevesloka

all: container

emmie: emmie.go pods.go replicationControllers.go services.go namespaces.go secrets.go configmaps.go deployments.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo --ldflags '-w' ./emmie.go ./pods.go ./replicationControllers.go ./services.go ./namespaces.go ./secrets.go ./configmaps.go ./deployments.go

container: emmie
	docker build -t $(PREFIX)/emmie:$(TAG) .

push:
	docker push $(PREFIX)/emmie:$(TAG)

clean:
	rm -f emmie

test: clean
	godep go test -v --vmodule=*=4
