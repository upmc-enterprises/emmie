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

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

// createNamespace creates a new namespace
func createNamespace(name string) error {
	// mark the namespace as being deployed by emmie
	m := make(map[string]string)
	m["deployedBy"] = "emmie"

	ns := &api.Namespace{
		ObjectMeta: api.ObjectMeta{Name: name, Labels: m},
	}

	_, err := client.Namespaces().Create(ns)

	if err != nil {
		log.Println("[createNamespace] Error creating namespace", err)
		return err
	}

	return err
}

// listNamespaces by label
func listNamespaces(labelKey, labelValue string) (*api.NamespaceList, error) {
	selector := labels.Set{labelKey: labelValue}.AsSelector()
	listOptions := api.ListOptions{FieldSelector: fields.Everything(), LabelSelector: selector}
	list, err := client.Namespaces().List(listOptions)

	if err != nil {
		log.Println("[listServices] Error listing namespaces", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		log.Println("[listServices] No namespaces could be found: labelKey: ", labelKey, " labelValue: ", labelValue)
	}

	return list, nil
}

// deleteNamespace delete a namespace
func deleteNamespace(name string) {
	err := client.Namespaces().Delete(name)

	if err != nil {
		log.Println("[deleteNamespace] Error deleting namespace", err)
		return
	}

	log.Println("Deleted namespace:", name)
}

func getDeploymentsRoute(w http.ResponseWriter, r *http.Request) {
	nss, err := listNamespaces("deployedBy", "emmie")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(nss); err != nil {
			panic(err)
		}
	}
}
