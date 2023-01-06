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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var problemlog = logf.Log.WithName("problem-resource")

func (r *Problem) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-netcon-janog-gr-jp-v1alpha1-problem,mutating=true,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problems,verbs=create;update,versions=v1alpha1,name=mproblem.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Problem{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Problem) Default() {
	problemlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-netcon-janog-gr-jp-v1alpha1-problem,mutating=false,failurePolicy=fail,sideEffects=None,groups=netcon.janog.gr.jp,resources=problems,verbs=create;update,versions=v1alpha1,name=vproblem.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Problem{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Problem) ValidateCreate() error {
	problemlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Problem) ValidateUpdate(old runtime.Object) error {
	problemlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Problem) ValidateDelete() error {
	problemlog.Info("validate delete", "name", r.Name)

	return nil
}
