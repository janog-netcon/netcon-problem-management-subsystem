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
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *ProblemEnvironment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-netcon-janog-gr-jp-v1alpha1-problemenvironment,mutating=true,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=create;update,versions=v1alpha1,name=mproblemenvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ProblemEnvironment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ProblemEnvironment) Default() {
}

//+kubebuilder:webhook:path=/validate-netcon-janog-gr-jp-v1alpha1-problemenvironment,mutating=false,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=create;update,versions=v1alpha1,name=vproblemenvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ProblemEnvironment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateCreate() (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	or := old.(*ProblemEnvironment)

	if or.Spec.WorkerName != "" && r.Spec.WorkerName != or.Spec.WorkerName {
		return nil, fmt.Errorf(".spec.workerName: workerName can't be updated after scheduling")
	}

	if !reflect.DeepEqual(r.Spec.TopologyFile, or.Spec.TopologyFile) {
		return nil, fmt.Errorf(".spec.containerLabManifest: containerLabManifest can't be updated")
	}

	if !reflect.DeepEqual(r.Spec.WorkerSelectors, or.Spec.WorkerSelectors) {
		return nil, fmt.Errorf(".spec.workerSelectors: workerSelectors can't be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
