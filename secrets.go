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

	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/fields"
	"k8s.io/client-go/1.4/pkg/labels"
)

func getSecretRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	secretName := vars["secretName"]
	namespace := vars["namespace"]

	secret, err := getSecret(secretName, namespace)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(secret); err != nil {
			panic(err)
		}
	}
}

func getSecretsRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value := vars["value"]
	namespace := vars["namespace"]

	secret, err := listSecrets(namespace, key, value)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if len(secret.Items) > 0 {
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(secret); err != nil {
				panic(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func listSecretsByNamespace(namespace string) (*v1.SecretList, error) {
	list, err := client.Core().Secrets(namespace).List(api.ListOptions{})

	if err != nil {
		log.Println("[listSecretsByNamespace] error listing secrets", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		log.Println("[listSecretsByNamespace] No secrets could be found for namespace!", namespace)
	}

	return list, nil
}

func listSecrets(namespace, labelKey, labelValue string) (*v1.SecretList, error) {
	selector := labels.Set{labelKey: labelValue}.AsSelector()
	listOptions := api.ListOptions{FieldSelector: fields.Everything(), LabelSelector: selector}
	list, err := client.Core().Secrets(namespace).List(listOptions)

	if err != nil {
		log.Println("[listSecrets] Error listing secrets", err)
		return nil, err
	}

	if len(list.Items) == 0 {
		log.Println("[listSecrets] No secrets could be found for namespace:", namespace, " labelKey: ", labelKey, " labelValue: ", labelValue)
	}

	return list, nil
}

func getSecret(secretName, namespace string) (*v1.Secret, error) {
	svc, err := client.Core().Secrets(namespace).Get(secretName)

	if err != nil {
		log.Println("[getSecret] Error getting secret!", err)
		return nil, err
	}

	return svc, nil
}

func createSecret(namespace string, secret *v1.Secret) error {
	_, err := client.Core().Secrets(namespace).Create(secret)

	if err != nil {
		log.Println("[createSecret] Error creating secret:", err)
	}
	return err
}

func deleteSecret(namespace, name string) error {
	// TODO: Use nil?
	err := client.Secrets(namespace).Delete(name, nil)

	if err != nil {
		log.Println("[deleteSecret] Error deleting secret:", err)
	}
	return err
}
