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
	ProblemEnvironmentConditionInitialized ProblemEnvironmentConditionType = "Initialized"

	ProblemEnvironmentConditionScheduled ProblemEnvironmentConditionType = "Scheduled"

	ProblemEnvironmentConditionReady ProblemEnvironmentConditionType = "Ready"
)

// ProblemEnvironmentSpec defines the desired state of ProblemEnvironment
type ProblemEnvironmentSpec struct {
	ProblemRef ProblemReference `json:"problemRef"`

	WorkerName string `json:"workerName,omitempty"`
}

type ProblemReference struct {
	Name string `json:"name"`
}

// ProblemEnvironmentStatus defines the observed state of ProblemEnvironment
type ProblemEnvironmentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProblemEnvironment is the Schema for the problemenvironments API
type ProblemEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProblemEnvironmentSpec   `json:"spec,omitempty"`
	Status ProblemEnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProblemEnvironmentList contains a list of ProblemEnvironment
type ProblemEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProblemEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProblemEnvironment{}, &ProblemEnvironmentList{})
}