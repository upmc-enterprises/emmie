[![Build Status](https://travis-ci.org/upmc-enterprises/emmie.svg?branch=addCI)](https://travis-ci.org/upmc-enterprises/emmie)

## "Emmie" is a deployment tool to facilitate deployments of applications to a Kubernetes cluster
Emmie assists with creating unique and isolated environments for each feature being worked on within an application. This allows for easier testing since the entire application is stood up and doesn't rely on brittle mocks or fake components.

## Requirements
* Working Kubernetes cluster
* Build docker images and tag with branchName (e.g. `stevesloka/web:US1234_AddLogging`)
 * Note: All branchNames need to be consistent across repos
* Deploy all replication controllers and services to a template namespace. This can be a new namespace or configured to be "develop" branch

### Application Arguments:
* listen-port: Port Emmie will listen on to take requests (NOTE: Only listens on HTTPS)
* docker-registry: Set to url of private docker registry
* template-namespace: Namespace to 'clone from when creating new deployments'
* path-to-tokens: Full path including file name to tokens file for authorization, setting to empty string will disable.

## Building

* Install Go
  * Set your $GOPATH to something you like
* Checkout or move this source directory to $GOPATH/src
* Run `cd $GOPATH/src/emmie && make emmie` to get dependencies and build project
* To build the docker immage, run `make`
* To push your image to docker hub, set $PREFIX to your account name, then `make push`
  * To push to Amazon ECR, create a repo and follow the instructions from the platform

## Get Started
1. Create auth tokens file (how?)
* Generate certs (or copy certs from your existing kubectl installation?)
* Deploy Emmie to your cluster ([See samples](k8s))
* Deploy Router to your cluster (`kubectl create -f k8s/router.yml`)
* Create template namespace (how?)
* Deploy template services / replication controllers
  * _NOTE: Make sure your templates have correct Annotations on them for what to deploy (See [Annotations](#annotations) section below)._
* Configure CI to build feature branch images
* Configure CI to curl Emmie and deploy new namespace
* Bask in the glory of how your team will produce better well tested code

## How it works

Emmie integrates into the k8s api via the supported go client. Setup your CI server to build all Docker images and tag with branch name. Then send POST request and Emmie will look at all the services and replication controllers in the configured template namespace, and deploy to a new namespace. You can repeat this as many times as your cluster has resources.

### AWS ECR Integration
By default Emmie will tag all the images in the k8s cluster with the branch name requested, however, this means that a tag for the branchname MUST exist in the Docker registry. This can be cubersome since not all images will need to be build (e.g. default to develop) in addition, there is a large overhead of time involved. 

Providing Emmie with the account credentials for the ECR repository you want to access, can now check if an image exists first, and if so, will use that tag, otherwise will default to your template image tag. 

#### Routes
* POST /deploy/{namespace}/{branchName} : Deploy a new branch
* DELETE /deploy/{branchName} : Delete an environment
* PUT /deploy/{branchName} : Update an existing environment
* GET /deploy : Get list of current deployments

_NOTE: Include a token query string to end of all requests for simple auth._

### Annotations

Since a pod can be made up of many containers, an annotation is required to tell Emmie how to update that pod when deploying new code. In the example below, Emmie looks for the key `emmie-update` and uses the value to find the container name to replace. In this case the container named `web` in the pod will be updated with the feature branch deploy.

```
annotations:
      emmie-update: web
```

## Generate Self-Signed cert
`openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes`

### Generate cert for Traefik

```
cd k8s/router
openssl req -newkey rsa:2048 -nodes -keyout tls.key -x509 -days 365 -out tls.crt
```

## Demo
Watch a demo here:<br>
[![Emmie Demo](http://img.youtube.com/vi/w7rshjxsojA/0.jpg)](http://www.youtube.com/watch?v=w7rshjxsojA)<br>
[http://www.youtube.com/watch?v=w7rshjxsojA](http://www.youtube.com/watch?v=w7rshjxsojA)

## Gotchas
* Do not have network mounts or host mounts, for now this proves difficult to manage via the template system
* If multiple containers are used in a single pod, make sure annotate

## About
Built by UPMC Enterprises in Pittsburgh, PA.
http://enterprises.upmc.com/
