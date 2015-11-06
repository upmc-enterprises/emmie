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
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

func getReplicationControllerRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rcName := vars["rcName"]
	namespace := vars["namespace"]

	rc, err := getReplicationController(rcName, namespace)

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

func getReplicationControllersRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value := vars["value"]
	namespace := vars["namespace"]

	rc, err := listReplicationControllers(namespace, key, value)

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

func listReplicationControllersByNamespace(namespace string) (*api.ReplicationControllerList, error) {
	list, err := client.ReplicationControllers(namespace).List(labels.Everything(), fields.Everything())

	if err != nil {
		glog.Error("[listReplicationControllersByNamespace] Error listing replicationControllers", err)
		return nil, err
	}
	return list, nil
}

func listReplicationControllers(namespace, labelKey, labelValue string) (*api.ReplicationControllerList, error) {
	selector := labels.Set{labelKey: labelValue}.AsSelector()
	list, err := client.ReplicationControllers(namespace).List(selector, fields.Everything())

	if err != nil {
		glog.Error("[listReplicationControllers] Error listing replicationControllers", err)
		return nil, err
	}
	return list, nil
}

func getReplicationController(replicationControllerName, namespace string) (*api.ReplicationController, error) {
	rc, err := client.ReplicationControllers(namespace).Get(replicationControllerName)

	if err != nil {
		glog.Error("[getReplicationController] Error getting replicationController", err)
		return nil, err
	}
	return rc, nil
}

func createReplicationController(namespace string, rc *api.ReplicationController) error {
	_, err := client.ReplicationControllers(namespace).Create(rc)

	if err != nil {
		glog.Error("[createReplicationController] Error creating replicationController:", err)
	}
	return err
}

func deleteReplicationController(namespace, name string) error {
	err := client.ReplicationControllers(namespace).Delete(name)

	if err != nil {
		glog.Error("[deleteReplicationController] Error deleting replicationController:", err)
	}
	return err
}
