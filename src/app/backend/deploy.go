// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	api "k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/util"
)

const (
	DescriptionAnnotationKey = "description"
	NameLabelKey             = "name"
)

// Specification for an app deployment.
type AppDeploymentSpec struct {
	// Name of the application.
	Name string `json:"name"`

	// Docker image path for the application.
	ContainerImage string `json:"containerImage"`

	// Number of replicas of the image to maintain.
	Replicas int `json:"replicas"`

	// Port mappings for the service that is created. The service is created if there is at least
	// one port mapping.
	PortMappings []PortMapping `json:"portMappings"`

	// Whether the created service is external.
	IsExternal bool `json:"isExternal"`

	// Description of the deployment.
	Description string `json:"description"`

	// Target namespace of the application.
	Namespace string `json:"namespace"`
}

// Port mapping for an application deployment.
type PortMapping struct {
	// Port that will be exposed on the service.
	Port int `json:"port"`

	// Docker image path for the application.
	TargetPort int `json:"targetPort"`

	// IP protocol for the mapping, e.g., "TCP" or "UDP".
	Protocol api.Protocol `json:"protocol"`
}

// Deploys an app based on the given configuration. The app is deployed using the given client.
// App deployment consists of a replication controller and an optional service. Both of them share
// common labels.
// TODO(bryk): Write tests for this function.
func DeployApp(spec *AppDeploymentSpec, client *client.Client) error {
	annotations := map[string]string{DescriptionAnnotationKey: spec.Description}
	labels := map[string]string{NameLabelKey: spec.Name}
	objectMeta := api.ObjectMeta{
		Annotations: annotations,
		Name:        spec.Name,
		Labels:      labels,
	}

	podTemplate := &api.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec: api.PodSpec{
			Containers: []api.Container{{
				Name:  spec.Name,
				Image: spec.ContainerImage,
			}},
		},
	}

	replicaSet := &api.ReplicationController{
		ObjectMeta: objectMeta,
		Spec: api.ReplicationControllerSpec{
			Replicas: spec.Replicas,
			Selector: labels,
			Template: podTemplate,
		},
	}

	_, err := client.ReplicationControllers(spec.Namespace).Create(replicaSet)

	if err != nil {
		// TODO(bryk): Roll back created resources in case of error.
		return err
	}

	if len(spec.PortMappings) > 0 {
		service := &api.Service{
			ObjectMeta: objectMeta,
			Spec: api.ServiceSpec{
				Selector: labels,
			},
		}

		if spec.IsExternal {
			service.Spec.Type = api.ServiceTypeLoadBalancer
		} else {
			service.Spec.Type = api.ServiceTypeNodePort
		}

		for _, portMapping := range spec.PortMappings {
			servicePort :=
				api.ServicePort{
					Protocol: portMapping.Protocol,
					Port:     portMapping.Port,
					TargetPort: util.IntOrString{
						Kind:   util.IntstrInt,
						IntVal: portMapping.TargetPort,
					},
				}
			service.Spec.Ports = append(service.Spec.Ports, servicePort)
		}

		_, err = client.Services(spec.Namespace).Create(service)

		// TODO(bryk): Roll back created resources in case of error.

		return err
	} else {
		return nil
	}
}