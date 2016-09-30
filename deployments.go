/*
Copyright (c) 2016, UPMC Enterprises
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
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.4/pkg/fields"
	"k8s.io/client-go/1.4/pkg/labels"
)

func getDeploymentRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentName := vars["deploymentName"]
	namespace := vars["namespace"]

	rc, err := getDeployment(deploymentName, namespace)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(rc); err != nil {
			panic(err)
		}
	}
}

func getDeploymentsRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value := vars["value"]
	namespace := vars["namespace"]

	rc, err := listDeployments(namespace, key, value)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if len(rc.Items) > 0 {
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(rc); err != nil {
				panic(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func listDeploymentsByNamespace(namespace string) (*v1beta1.DeploymentList, error) {
	list, err := client.Deployments(namespace).List(api.ListOptions{})

	if err != nil {
		log.Println("[listDeploymentsByNamespace] Error listing Deployments", err)
		return nil, err
	}
	return list, nil
}

func listDeployments(namespace, labelKey, labelValue string) (*v1beta1.DeploymentList, error) {
	selector := labels.Set{labelKey: labelValue}.AsSelector()
	listOptions := api.ListOptions{FieldSelector: fields.Everything(), LabelSelector: selector}
	list, err := client.Deployments(namespace).List(listOptions)

	if err != nil {
		log.Println("[listDeployments] Error listing Deployments", err)
		return nil, err
	}
	return list, nil
}

func getDeployment(DeploymentName, namespace string) (*v1beta1.Deployment, error) {
	rc, err := client.Deployments(namespace).Get(DeploymentName)

	if err != nil {
		log.Println("[getDeployment] Error getting Deployment", err)
		return nil, err
	}
	return rc, nil
}

func createDeployment(namespace string, rc *v1beta1.Deployment) error {
	_, err := client.Deployments(namespace).Create(rc)

	if err != nil {
		log.Println("[createDeployment] Error creating Deployment:", err)
	}
	return err
}

func deleteDeployment(namespace, name string) error {
	// TODO: Use nil?
	err := client.Deployments(namespace).Delete(name, nil)

	if err != nil {
		log.Println("[deleteDeployment] Error deleting Deployment:", err)
	}
	return err
}
