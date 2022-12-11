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

// ProblemSpec defines the desired state of Problem
type ProblemSpec struct {
	Template *ProblemEnvironmentTemplate `json:"template"`

	AssignableReplicas int `json:"assignableReplicas"`
}

type ProblemEnvironmentTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProblemEnvironmentSpec `json:"spec,omitempty"`
}

// ProblemStatus defines the observed state of Problem
type ProblemStatus struct {
	Replicas ProblemReplicas `json:"replicas"`
}

type ProblemReplicas struct {
	// Total is the total number of ProblemEnvironments
	Total int `json:"total"`

	// Scheduled is the number of ProblemEnvironments which is scheduled but not ready
	Scheduled int `json:"scheduled"`

	// Assignable is the number of ProblemEnvironments which is ready but not assigned
	Assignable int `json:"assignable"`

	// Assigned is the number of ProblemEnvironments which is assigned
	Assigned int `json:"assigned"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName={p,prob}
//+kubebuilder:printcolumn:name=DESIRED,type=integer,JSONPath=.spec.assignableReplicas
//+kubebuilder:printcolumn:name=SCHEDULED,type=integer,JSONPath=.status.replicas.scheduled,priority=1
//+kubebuilder:printcolumn:name=ASSIGNABLE,type=integer,JSONPath=.status.replicas.assignable
//+kubebuilder:printcolumn:name=ASSIGNED,type=integer,JSONPath=.status.replicas.assigned,priority=1
//+kubebuilder:printcolumn:name=TOTAL,type=integer,JSONPath=.status.replicas.total,priority=1

// Problem is the Schema for the problems API
type Problem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProblemSpec   `json:"spec,omitempty"`
	Status ProblemStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProblemList contains a list of Problem
type ProblemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Problem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Problem{}, &ProblemList{})
}
