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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var problemenvironmentlog = logf.Log.WithName("problemenvironment-resource")

func (r *ProblemEnvironment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-netcon-janog-gr-jp-v1alpha1-problemenvironment,mutating=true,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=create;update,versions=v1alpha1,name=mproblemenvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ProblemEnvironment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ProblemEnvironment) Default() {
	problemenvironmentlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-netcon-janog-gr-jp-v1alpha1-problemenvironment,mutating=false,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problemenvironments,verbs=create;update,versions=v1alpha1,name=vproblemenvironment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ProblemEnvironment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateCreate() error {
	problemenvironmentlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateUpdate(old runtime.Object) error {
	problemenvironmentlog.Info("validate update", "name", r.Name)

	or := old.(*ProblemEnvironment)

	if or.Spec.WorkerName != "" && r.Spec.WorkerName != or.Spec.WorkerName {
		return fmt.Errorf(".spec.workerName: workerName can't be updated after scheduling")
	}

	if !reflect.DeepEqual(r.Spec.TopologyFile, or.Spec.TopologyFile) {
		return fmt.Errorf(".spec.containerLabManifest: containerLabManifest can't be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProblemEnvironment) ValidateDelete() error {
	problemenvironmentlog.Info("validate delete", "name", r.Name)

	return nil
}
