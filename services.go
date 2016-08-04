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
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

func getServiceRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["serviceName"]
	namespace := vars["namespace"]

	svc, err := getService(serviceName, namespace)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(svc); err != nil {
			panic(err)
		}
	}
}

func getServicesRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value := vars["value"]
	namespace := vars["namespace"]

	svc, err := listServices(namespace, key, value)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if len(svc.Items) > 0 {
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(svc); err != nil {
				panic(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func listServicesByNamespace(namespace string) (*api.ServiceList, error) {
	list, err := client.Services(namespace).List(api.ListOptions{})

	if err != nil {
		log.Println("[listServicesByNamespace] error listing services", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		log.Println("[listServicesByNamespace] No services could be found for namespace!", namespace)
	}

	return list, nil
}

func listServices(namespace, labelKey, labelValue string) (*api.ServiceList, error) {
	selector := labels.Set{labelKey: labelValue}.AsSelector()
	listOptions := api.ListOptions{FieldSelector: fields.Everything(), LabelSelector: selector}
	list, err := client.Services(namespace).List(listOptions)

	if err != nil {
		log.Println("[listServices] Error listing services", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		log.Println("[listServices] No services could be found for namespace:", namespace, " labelKey: ", labelKey, " labelValue: ", labelValue)
	}

	return list, nil
}

func getService(serviceName, namespace string) (*api.Service, error) {
	svc, err := client.Services(namespace).Get(serviceName)

	if err != nil {
		log.Println("[getService] Error getting service!", err)
		return nil, err
	}

	return svc, nil
}

func createService(namespace string, svc *api.Service) error {
	_, err := client.Services(namespace).Create(svc)

	if err != nil {
		log.Println("[createService] Error creating service:", err)
	}
	return err
}

func deleteService(namespace, name string) error {
	err := client.Services(namespace).Delete(name)

	if err != nil {
		log.Println("[deleteService] Error deleting service:", err)
	}
	return err
}
