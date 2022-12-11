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

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName={p,prob}

// Problem is the Schema for the problems API
type Problem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProblemSpec `json:"spec,omitempty"`
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
