/*
Copyright (c) 2015, UPMC Enterprises
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.4/tools/clientcmd"
)

/* Set up a global string for our secret */
var jwtSigningKey = os.Getenv("AUTH0_CLIENT_SECRET")

var (
	argListenPort        = flag.Int("listen-port", 9080, "port to have API listen")
	argDockerRegistry    = flag.String("docker-registry", "", "docker registry to use")
	argKubecfgFile       = flag.String("kubecfg-file", "", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	argKubeMasterURL     = flag.String("kube-master-url", "", "URL to reach kubernetes master. Env variables in this flag will be expanded.")
	argTemplateNamespace = flag.String("template-namespace", "template", "Namespace to 'clone from when creating new deployments'")
	argSubDomain         = flag.String("subdomain", "k8s.local.com", "Subdomain used to configure external routing to branch (e.g. namespace.ci.k8s.local)")
	client               *kubernetes.Clientset
	defaultReplicaCount  *int32
)

const (
	appVersion = "0.0.3"
)

// Default (GET "/")
func indexRoute(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s", "welcome to Emmie!")
}

// Version (GET "/version")
var versionRoute = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%q", appVersion)
})

// Deploy (POST "/deploy/namespace/branchName")
func deployRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	imageNamespace := vars["namespace"]

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)
	log.Println("[Emmie] is deploying branch:", branchName)

	// create namespace
	err := createNamespace(branchName)

	if err != nil {
		// TODO: Don't use error for logic
		// Existing namespace, do an update
		log.Println("Existing namespace found: ", branchName, " deleting pods.")

		deletePodsByNamespace(branchName)
	} else {
		log.Println("Namespace created, deploying new app...")

		// copy controllers / services based on label query
		rcs, _ := listReplicationControllersByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(rcs.Items), " template replication controllers to copy.")

		deployments, _ := listDeploymentsByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(deployments.Items), " template deployments to copy.")

		svcs, _ := listServicesByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(svcs.Items), " template services to copy.")

		secrets, _ := listSecretsByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(secrets.Items), " template secrets to copy.")

		configmaps, _ := listConfigMapsByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(configmaps.Items), " template configmaps to copy.")

		ingresses, _ := listIngresssByNamespace(*argTemplateNamespace)
		log.Println("Found ", len(ingresses.Items), " template ingresses to copy.")

		// create configmaps
		for _, configmap := range configmaps.Items {

			requestConfigMap := &v1.ConfigMap{
				ObjectMeta: v1.ObjectMeta{
					Name:      configmap.Name,
					Namespace: branchName,
				},
				Data: configmap.Data,
			}

			createConfigMap(branchName, requestConfigMap)
		}

		// create secrets
		for _, secret := range secrets.Items {

			// skip service accounts
			if secret.Type != "kubernetes.io/service-account-token" {

				requestSecret := &v1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secret.Name,
						Namespace: branchName,
					},
					Type: secret.Type,
					Data: secret.Data,
				}

				createSecret(branchName, requestSecret)
			}
		}

		// create services
		for _, svc := range svcs.Items {

			requestService := &v1.Service{
				ObjectMeta: v1.ObjectMeta{
					Name:      svc.ObjectMeta.Name,
					Namespace: branchName,
				},
			}

			ports := []v1.ServicePort{}
			for _, port := range svc.Spec.Ports {
				newPort := v1.ServicePort{
					Name:       port.Name,
					Protocol:   port.Protocol,
					Port:       port.Port,
					TargetPort: port.TargetPort,
				}

				ports = append(ports, newPort)
			}

			requestService.Spec.Ports = ports
			requestService.Spec.Selector = svc.Spec.Selector
			requestService.Spec.Type = svc.Spec.Type
			requestService.Labels = svc.Labels

			createService(branchName, requestService)
		}

		// now that we have all replicationControllers, update them to have new image name
		for _, rc := range rcs.Items {

			containerNameToUpdate := ""

			// Looks for annotations to know which container to replace
			for key, value := range rc.Annotations {
				if key == "emmie-update" {
					containerNameToUpdate = value
				}
			}

			// Find the container which matches the annotation
			for i, container := range rc.Spec.Template.Spec.Containers {

				imageName := ""

				if containerNameToUpdate == rc.ObjectMeta.Name {
					imageName = fmt.Sprintf("%s%s/%s:%s", *argDockerRegistry, imageNamespace, rc.ObjectMeta.Name, branchName)
				} else {
					//default to current image tag if no annotations found
					imageName = container.Image
				}

				rc.Spec.Template.Spec.Containers[i].Image = imageName

				// Set the image pull policy to "Always"
				rc.Spec.Template.Spec.Containers[i].ImagePullPolicy = "Always"
			}

			requestController := &v1.ReplicationController{
				ObjectMeta: v1.ObjectMeta{
					Name:      rc.ObjectMeta.Name,
					Namespace: branchName,
				},
			}

			requestController.Spec = rc.Spec
			requestController.Spec.Replicas = defaultReplicaCount

			// create new replication controller
			createReplicationController(branchName, requestController)
		}

		// now that we have all deployments, update them to have new image name
		for _, dply := range deployments.Items {

			containerNameToUpdate := ""

			// Looks for annotations to know which container to replace
			for key, value := range dply.Annotations {
				if key == "emmie-update" {
					containerNameToUpdate = value
				}
			}

			// Find the container which matches the annotation
			for i, container := range dply.Spec.Template.Spec.Containers {

				imageName := ""

				if containerNameToUpdate == container.Name {
					imageName = fmt.Sprintf("%s%s/%s:%s", *argDockerRegistry, imageNamespace, container.Name, branchName)
				} else {
					//default to current image tag if no annotations found
					imageName = container.Image
				}

				dply.Spec.Template.Spec.Containers[i].Image = imageName

				// Set the image pull policy to "Always"
				dply.Spec.Template.Spec.Containers[i].ImagePullPolicy = "Always"
			}

			deployment := &v1beta1.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      dply.ObjectMeta.Name,
					Namespace: branchName,
				},
			}

			deployment.Spec = dply.Spec
			deployment.Spec.Replicas = defaultReplicaCount

			// create new replication controller
			createDeployment(branchName, deployment)
		}

		// create ingress
		for _, ingress := range ingresses.Items {

			rules := ingress.Spec.Rules
			rules[0].Host = fmt.Sprintf("%s.%s", branchName, *argSubDomain)

			requestIngress := &v1beta1.Ingress{
				ObjectMeta: v1.ObjectMeta{
					Name:      ingress.Name,
					Namespace: branchName,
				},
				Spec: v1beta1.IngressSpec{
					Rules: rules,
				},
			}

			createIngress(branchName, requestIngress)
		}
	}
	log.Println("[Emmie] is finished deploying branch!")
}

// Put (PUT "/deploy")
func updateRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	log.Println(w, "[Emmie] is updating branch:", branchName)

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)

	deletePodsByNamespace(branchName)

	log.Println("Finished updating branch!")
}

// Delete (DELETE "/deploy")
func deleteRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	log.Println("[Emmie] is deleting branch:", branchName)

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)

	// get controllers / services / secrets in namespace
	rcs, _ := listReplicationControllersByNamespace(*argTemplateNamespace)
	for _, rc := range rcs.Items {
		deleteReplicationController(branchName, rc.ObjectMeta.Name)
		log.Println("Deleted replicationController:", rc.ObjectMeta.Name)
	}

	deployments, _ := listDeploymentsByNamespace(*argTemplateNamespace)
	for _, dply := range deployments.Items {
		deleteDeployment(branchName, dply.ObjectMeta.Name)
		log.Println("Deleted deployment:", dply.ObjectMeta.Name)
	}

	svcs, _ := listServicesByNamespace(*argTemplateNamespace)
	for _, svc := range svcs.Items {
		deleteService(branchName, svc.ObjectMeta.Name)
		log.Println("Deleted service:", svc.ObjectMeta.Name)
	}

	secrets, _ := listSecretsByNamespace(*argTemplateNamespace)
	for _, secret := range secrets.Items {
		deleteSecret(branchName, secret.ObjectMeta.Name)
		log.Println("Deleted secret:", secret.ObjectMeta.Name)
	}

	configmaps, _ := listConfigMapsByNamespace(*argTemplateNamespace)
	for _, configmap := range configmaps.Items {
		deleteSecret(branchName, configmap.ObjectMeta.Name)
		log.Println("Deleted configmap:", configmap.ObjectMeta.Name)
	}

	ingresses, _ := listIngresssByNamespace(*argTemplateNamespace)
	for _, ingress := range ingresses.Items {
		deleteIngress(branchName, ingress.ObjectMeta.Name)
		log.Println("Deleted ingress:", ingress.ObjectMeta.Name)
	}

	deleteNamespace(branchName)
	log.Println("[Emmie] is done deleting branch.")
}

// Get (GET "/deploy")
func getRoute(w http.ResponseWriter, r *http.Request) {
	ns, err := listNamespaces("deployedBy", "emmie")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ns)
}

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		signingKey, err := base64.URLEncoding.DecodeString(jwtSigningKey)
		if err != nil {
			return nil, err
		}
		return signingKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

func main() {
	flag.Parse()
	log.Println("[Emmie] is up and running!", time.Now())

	// Sanitize docker registry
	if *argDockerRegistry != "" {
		*argDockerRegistry = fmt.Sprintf("%s/", *argDockerRegistry)
	}

	// Configure router
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", indexRoute)
	router.HandleFunc("/deploy/{namespace}/{branchName}", deployRoute).Methods("POST")
	router.HandleFunc("/deploy/{branchName}", deleteRoute).Methods("DELETE")
	router.HandleFunc("/deploy/{branchName}", updateRoute).Methods("PUT")
	router.HandleFunc("/deploy", getRoute).Methods("GET")

	// Services
	// router.HandleFunc("/services/{namespace}/{serviceName}", getServiceRoute).Methods("GET")
	// router.HandleFunc("/services/{namespace}/{key}/{value}", getServicesRoute).Methods("GET")

	// ReplicationControllers
	// router.HandleFunc("/replicationControllers/{namespace}/{rcName}", getReplicationControllerRoute).Methods("GET")
	// router.HandleFunc("/replicationControllers/{namespace}/{key}/{value}", getReplicationControllersRoute).Methods("GET")

	// Deployments
	// router.HandleFunc("/deployments/{namespace}/{deploymentName}", getDeploymentRoute).Methods("GET")
	// router.HandleFunc("/deployments/{namespace}/{key}/{value}", getDeploymentsRoute).Methods("GET")

	// Version
	router.Handle("/version", jwtMiddleware.Handler(versionRoute)).Methods("GET")

	// Create k8s client
	//config, err := rest.InClusterConfig()
	config, err := clientcmd.BuildConfigFromFlags("", *argKubecfgFile)
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	client = clientset

	// Start server
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", *argListenPort), "certs/cert.pem", "certs/key.pem", router))
	//log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *argListenPort), router))
}
