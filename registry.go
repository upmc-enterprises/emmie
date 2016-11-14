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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func imageTagExists(repositoryName, tag, awsRegion, awsRegistryID string) (bool, error) {
	// Default to only look for tagged images
	awsTagStatus := "TAGGED"

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return false, err
	}

	svc := ecr.New(sess, &aws.Config{Region: aws.String(awsRegion)})

	// Get images
	params := &ecr.ListImagesInput{
		RepositoryName: aws.String(repositoryName), // Required
		Filter: &ecr.ListImagesFilter{
			TagStatus: aws.String(awsTagStatus),
		},
		RegistryId: aws.String(awsRegistryID),
	}

	resp, err := svc.ListImages(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return false, err
	}

	for _, image := range resp.ImageIds {
		if image.ImageTag != nil && *image.ImageTag == tag {
			return true, nil
		}
	}

	return false, nil
}
