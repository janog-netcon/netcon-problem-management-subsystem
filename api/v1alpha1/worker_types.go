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

type WorkerConditionType string

const WorkerConditionReady WorkerConditionType = "Ready"

const (
	WorkerEventReady    string = "Ready"
	WorkerEventNotReady string = "NotReady"
)

// WorkerStatus defines the desired state of Worker
type WorkerSpec struct {
	DisableSchedule bool `json:"disableSchedule"`
}

// WorkerStatus defines the observed state of Worker
type WorkerStatus struct {
	WorkerInfo WorkerInfo `json:"workerInfo"`

	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type WorkerInfo struct {
	ExternalIPAddress string `json:"externalIPAddress"`
	ExternalPort      uint16 `json:"externalPort"`
	Hostname          string `json:"hostname"`
	MemoryUsedPercent string `json:"memoryUsedPercent"`
	CPUUsedPercent    string `json:"cpuUsedPercent"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name=READY,type=string,JSONPath=.status.conditions[?(@.type=="Ready")].status
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=.metadata.creationTimestamp

// Worker is the Schema for the workers API
type Worker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkerSpec `json:"spec,omitempty"`

	Status WorkerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkerList contains a list of Worker
type WorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Worker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Worker{}, &WorkerList{})
}
