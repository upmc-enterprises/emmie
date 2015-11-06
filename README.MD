## "Emmie" is a deployment tool to facilitate deployments of applications to a Kubernetes cluster
Emmie assists with creating unique and isolated environments for each feature being worked on within an application. This allows for easier testing since the entire application is stood up and doesn't rely on brittle mocks or fake components.

## Requirements
* Working Kubernetes cluster
* Build docker images and tag with branchName (e.g. `stevesloka/web:US1234_AddLogging`)
 * Note: All branchNames need to be consistent across repos
* Deploy all replication controllers and services to a template namespace. This can be a new namespace or configured to be "develop" branch

### Application Arguments:
* listen-port: Port Emmie will listen on to take requests
* docker-registry: Set to url of private docker registry
* kubecfg-file: Location of kubecfg file for access to kubernetes master service
* kube-master-url:  overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens
* kube-master-url: URL to reach kubernetes master. Env variables in this flag will be expanded
* template-namespace: Namespace to 'clone from when creating new deployments'

## How it works
Emmie integrates into the k8s api via the supported go client. Setup your CI server to build all Docker images and tag with branch name. Then send POST request and Emmie will look at all the services and replication controllers in the configured template namespace, and deploy to a new namespace. You can repeat this as many times as your cluster has resources.

#### Routes
* POST /deploy/{namespace}/{branchName} : Deploy a new branch
* DELETE /deploy/{branchName} : Delete an environment
* PUT /deploy/{branchName} : Update an existing environment
* GET /deploy : Get list of current deployments

## Get Started
1. Deploy Emmie to your cluster ([See samples](k8s))
* Create template namespace
* Deploy template services / replication controllers
* Configure CI to build feature branch images
* Configure CI to curl Emmie and deploy new namespace
* Bask in the glory of how your team will produce better well tested code

<a href="http://www.youtube.com/watch?feature=player_embedded&v=w7rshjxsojA"
  target="_blank"><img src="http://img.youtube.com/vi/w7rshjxsojA/0.jpg"
alt="Emmie demo" width="240" height="180" border="10" /></a>

## Gotchas
* Do not have network mounts or host mounts, for now this proves difficult to manage via the template system
* If multiple containers are used in a single pod, make sure the image that needs updated is the first in the collection

## About
Built by UPMC Enterprises in Pittsburgh, PA.
http://enterprises.upmc.com/
