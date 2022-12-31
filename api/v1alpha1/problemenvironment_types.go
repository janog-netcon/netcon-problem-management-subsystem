/*
Copyright 2022 NETCON developers.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProblemEnvironmentConditionType string

const (
	// Scheduled will be True when:
	// * ProblemEnvironment is scheduled
	// * Worker where ProblemEnvironment is scheduled exists
	ProblemEnvironmentConditionScheduled ProblemEnvironmentConditionType = "Scheduled"

	// Deployed will be True when:
	// * ProblemEnvironment is deployed on Worker
	ProblemEnvironmentConditionDeployed ProblemEnvironmentConditionType = "Deployed"

	// Ready will be True when:
	// * ProblemEnvironment is ready on Worker
	ProblemEnvironmentConditionReady ProblemEnvironmentConditionType = "Ready"

	// Assigned will be True when:
	// * ProblemEnvironment is assigned to some users
	ProblemEnvironmentConditionAssigned ProblemEnvironmentConditionType = "Assigned"
)

// ProblemEnvironmentSpec defines the desired state of ProblemEnvironment
type ProblemEnvironmentSpec struct {
	// TopologyFile will be placed as `topology.yml`
	TopologyFile FileSource `json:"topologyFile" yaml:"topologyFile"`

	// ConfigFiles will be placed under the directory `config`
	ConfigFiles []FileSource `json:"configFiles,omitempty" yaml:"configFiles,omitempty"`

	WorkerName string `json:"workerName,omitempty" yaml:"workername,omitempty"`
}

type FileSource struct {
	ConfigMapRef ConfigMapFileSource `json:"configMapRef" yaml:"configMapRef"`
}

type ConfigMapFileSource struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ProblemEnvironmentStatus defines the observed state of ProblemEnvironment
type ProblemEnvironmentStatus struct {
	Containers *ContainersStatus `json:"containers,omitempty" yaml:"containers,omitempty"`

	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type ContainersStatus struct {
	Summary string                  `json:"summary" yaml:"summary"`
	Details []ContainerDetailStatus `json:"details" yaml:"details"`
}

type ContainerDetailStatus struct {
	Name                string `json:"name" yaml:"name"`
	Image               string `json:"image" yaml:"image"`
	ContainerID         string `json:"containerID" yaml:"containerID"`
	ContainerName       string `json:"containerName" yaml:"containerName"`
	Ready               bool   `json:"ready" yaml:"ready"`
	ManagementIPAddress string `json:"managementIPAddress" yaml:"managementIPAddress"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName={pe,probenv}
//+kubebuilder:printcolumn:name=SCHEDULED,type=string,JSONPath=.status.conditions[?(@.type=="Scheduled")].status
//+kubebuilder:printcolumn:name=WORKER,type=string,JSONPath=.spec.workerName,priority=1
//+kubebuilder:printcolumn:name=DEPLOYED,type=string,JSONPath=.status.conditions[?(@.type=="Deployed")].status
//+kubebuilder:printcolumn:name=READY,type=string,JSONPath=.status.conditions[?(@.type=="Ready")].status
//+kubebuilder:printcolumn:name=ASSIGNED,type=string,JSONPath=.status.conditions[?(@.type=="Assigned")].status
//+kubebuilder:printcolumn:name=CONTAINERS,type=string,JSONPath=.status.containers.summary,priority=1
//+kubebuilder:printcolumn:name=PASSWORD,type=string,JSONPath=.status.password,priority=1

// ProblemEnvironment is the Schema for the problemenvironments API
type ProblemEnvironment struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   ProblemEnvironmentSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status ProblemEnvironmentStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProblemEnvironmentList contains a list of ProblemEnvironment
type ProblemEnvironmentList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []ProblemEnvironment `json:"items" yaml:"items"`
}

func init() {
	SchemeBuilder.Register(&ProblemEnvironment{}, &ProblemEnvironmentList{})
}
