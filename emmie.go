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
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kclientcmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

var (
	argListenPort        = flag.Int("listen-port", 9080, "port to have API listen")
	argDockerRegistry    = flag.String("docker-registry", "", "docker registry to use")
	argKubecfgFile       = flag.String("kubecfg-file", "", "Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens")
	argKubeMasterURL     = flag.String("kube-master-url", "", "URL to reach kubernetes master. Env variables in this flag will be expanded.")
	argTemplateNamespace = flag.String("template-namespace", "template", "Namespace to 'clone from when creating new deployments'")
	argPathToTokens      = flag.String("path-to-tokens", "tokens.txt", "Full path including file name to tokens file for authorization, setting to empty string will disable.")
	client               *kclient.Client
)

const (
	appVersion = "0.0.1"
)

func expandKubeMasterURL() (string, error) {
	parsedURL, err := url.Parse(os.ExpandEnv(*argKubeMasterURL))
	if err != nil {
		return "", fmt.Errorf("failed to parse --kube_master_url %s - %v", *argKubeMasterURL, err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.Host == ":" {
		return "", fmt.Errorf("invalid --kube_master_url specified %s", *argKubeMasterURL)
	}
	return parsedURL.String(), nil
}

func newKubeClient() (*kclient.Client, error) {
	var (
		config    *kclient.Config
		err       error
		masterURL string
	)

	if *argKubeMasterURL != "" {
		masterURL, err = expandKubeMasterURL()

		if err != nil {
			return nil, err
		}
	}

	if masterURL != "" && *argKubecfgFile == "" {
		config = &kclient.Config{
			Host:    masterURL,
			Version: "v1",
		}
	} else {
		overrides := &kclientcmd.ConfigOverrides{}
		overrides.ClusterInfo.Server = masterURL
		rules := &kclientcmd.ClientConfigLoadingRules{ExplicitPath: *argKubecfgFile}
		if config, err = kclientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig(); err != nil {
			return nil, err
		}
	}

	glog.Infof("Using %s for kubernetes master", config.Host)
	glog.Infof("Using kubernetes API %s", config.Version)
	return kclient.New(config)
}

// Default (GET "/")
func indexRoute(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s", "welcome to Emmie!")
}

// Version (GET "/version")
func versionRoute(w http.ResponseWriter, r *http.Request) {
	if !tokenIsValid(r.FormValue("token")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, "%q", appVersion)
}

// Deploy (POST "/deploy/namespace/branchName")
func deployRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	imageNamespace := vars["namespace"]

	if !tokenIsValid(r.FormValue("token")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)
	glog.Info("[Emmie] is deploying branch:", branchName)

	// create namespace
	err := createNamespace(branchName)

	if err != nil {
		// TODO: Don't use error for logic
		// Existing namespace, do an update
		glog.Info("Existing namespace found: ", branchName, " deleting pods.")

		deletePodsByNamespace(branchName)
	} else {
		glog.Info("Namespace created, deploying new app...")

		// copy controllers / services based on label query
		rcs, _ := listReplicationControllersByNamespace(*argTemplateNamespace)
		glog.Info("Found ", len(rcs.Items), " template replication controllers to copy.")

		svcs, _ := listServicesByNamespace(*argTemplateNamespace)
		glog.Info("Found ", len(svcs.Items), " template services to copy.")

		// create services
		for _, svc := range svcs.Items {

			requestService := &api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:      svc.ObjectMeta.Name,
					Namespace: branchName,
				},
			}

			ports := []api.ServicePort{}
			for _, port := range svc.Spec.Ports {
				newPort := api.ServicePort{
					Protocol:   port.Protocol,
					Port:       port.Port,
					TargetPort: port.TargetPort,
				}

				ports = append(ports, newPort)
			}

			requestService.Spec.Ports = ports
			requestService.Spec.Selector = svc.Spec.Selector
			requestService.Spec.Type = svc.Spec.Type

			createService(branchName, requestService)
		}

		// now that we have all replicationControllers, update them to have new image name
		for _, rc := range rcs.Items {
			// TODO: Need to specify which containers should get new images (Defaulting to the first container)
			rc.Spec.Template.Spec.Containers[0].Image =
				fmt.Sprintf("%s%s/%s:%s", *argDockerRegistry, imageNamespace, rc.ObjectMeta.Labels["name"], branchName)

				// Set the image pull policy to "Always"
			rc.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Always"

			requestController := &api.ReplicationController{
				ObjectMeta: api.ObjectMeta{
					Name:      rc.ObjectMeta.Name,
					Namespace: branchName,
				},
			}

			requestController.Spec = rc.Spec
			requestController.Spec.Replicas = 1

			// create new replication controller
			createReplicationController(branchName, requestController)
		}
	}
	glog.Info("[Emmie] is finished deploying branch!")
}

// Put (PUT "/deploy")
func updateRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	glog.Info(w, "[Emmie] is updating branch:", branchName)

	if !tokenIsValid(r.FormValue("token")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)

	deletePodsByNamespace(branchName)

	glog.Info("Finished updating branch!")
}

// Delete (DELETE "/deploy")
func deleteRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	branchName := vars["branchName"]
	glog.Info("[Emmie] is deleting branch:", branchName)

	if !tokenIsValid(r.FormValue("token")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// sanitize BranchName
	branchName = strings.Replace(branchName, "_", "-", -1)

	// get controllers / services in namespace
	rcs, _ := listReplicationControllersByNamespace(*argTemplateNamespace)

	for _, rc := range rcs.Items {
		deleteReplicationController(branchName, rc.ObjectMeta.Name)
		glog.Info("Deleted replicationController:", rc.ObjectMeta.Name)
	}

	svcs, _ := listServicesByNamespace(*argTemplateNamespace)
	for _, svc := range svcs.Items {
		deleteService(branchName, svc.ObjectMeta.Name)
		glog.Info("Deleted service:", svc.ObjectMeta.Name)
	}

	deleteNamespace(branchName)
	glog.Info("[Emmie] is done deleting branch.")
}

func tokenIsValid(token string) bool {
	// If no path is passed, then auth is disabled
	if *argPathToTokens == "" {
		return true
	}

	file, err := os.Open(*argPathToTokens)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if token == scanner.Text() {
			fmt.Println("Token IS valid!")
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Token is NOT valid! =(")
	return false
}

func main() {
	flag.Parse()
	glog.Info("[Emmie] is up and running!", time.Now())

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
	router.HandleFunc("/deploy", getDeploymentsRoute).Methods("GET")

	// Services
	// router.HandleFunc("/services/{namespace}/{serviceName}", getServiceRoute).Methods("GET")
	// router.HandleFunc("/services/{namespace}/{key}/{value}", getServicesRoute).Methods("GET")

	// ReplicationControllers
	// router.HandleFunc("/replicationControllers/{namespace}/{rcName}", getReplicationControllerRoute).Methods("GET")
	// router.HandleFunc("/replicationControllers/{namespace}/{key}/{value}", getReplicationControllersRoute).Methods("GET")

	// Version
	router.HandleFunc("/version", versionRoute)

	// Create k8s client
	kubeClient, err := newKubeClient()
	if err != nil {
		glog.Fatalf("Failed to create a kubernetes client: %v", err)
	}
	client = kubeClient

	// Start server
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", *argListenPort), "certs/cert.pem", "certs/key.pem", router))
}
