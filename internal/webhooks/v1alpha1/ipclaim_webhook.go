/*
Copyright 2020 The Kubernetes Authors.
Copyright 2025 The Metal3 Authors.

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

package webhooks

import (
	"context"
	"fmt"

	ipamv1 "github.com/metal3-io/ip-address-manager/api/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (webhook *IPClaim) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&ipamv1.IPClaim{}).
		WithDefaulter(webhook, admission.DefaulterRemoveUnknownOrOmitableFields).
		WithValidator(webhook).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-ipam-metal3-io-v1alpha1-ipclaim,mutating=false,failurePolicy=fail,groups=ipam.metal3.io,resources=ipclaims,versions=v1alpha1,name=validation.ipclaim.ipam.metal3.io,matchPolicy=Equivalent,sideEffects=None,admissionReviewVersions=v1;v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-ipam-metal3-io-v1alpha1-ipclaim,mutating=true,failurePolicy=fail,groups=ipam.metal3.io,resources=ipclaims,versions=v1alpha1,name=default.ipclaim.ipam.metal3.io,matchPolicy=Equivalent,sideEffects=None,admissionReviewVersions=v1;v1beta1

// IPClaim implements a validation and defaulting webhook for IPClaim.
type IPClaim struct{}

var _ webhook.CustomDefaulter = &IPClaim{}
var _ webhook.CustomValidator = &IPClaim{}

func (webhook *IPClaim) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (webhook *IPClaim) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	c, ok := obj.(*ipamv1.IPClaim)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a IPClaim but got a %T", obj))
	}

	allErrs := field.ErrorList{}
	if c.Spec.Pool.Name == "" {
		allErrs = append(allErrs,
			field.Invalid(
				field.NewPath("spec", "pool", "name"),
				c.Spec.Pool.Name,
				"cannot be empty",
			),
		)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(ipamv1.GroupVersion.WithKind("IPClaim").GroupKind(), c.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (webhook *IPClaim) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	allErrs := field.ErrorList{}
	oldIPClaim, ok := oldObj.(*ipamv1.IPClaim)
	if !ok || oldIPClaim == nil {
		return nil, apierrors.NewInternalError(errors.New("unable to convert existing object"))
	}

	newIPClaim, ok := newObj.(*ipamv1.IPClaim)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a IPClaim but got a %T", newObj))
	}

	if newIPClaim.Spec.Pool.Name != oldIPClaim.Spec.Pool.Name {
		allErrs = append(allErrs,
			field.Invalid(
				field.NewPath("spec", "pool"),
				newIPClaim.Spec.Pool,
				"cannot be modified",
			),
		)
	} else if newIPClaim.Spec.Pool.Namespace != oldIPClaim.Spec.Pool.Namespace {
		allErrs = append(allErrs,
			field.Invalid(
				field.NewPath("spec", "pool"),
				newIPClaim.Spec.Pool,
				"cannot be modified",
			),
		)
	} else if newIPClaim.Spec.Pool.Kind != oldIPClaim.Spec.Pool.Kind {
		allErrs = append(allErrs,
			field.Invalid(
				field.NewPath("spec", "pool"),
				newIPClaim.Spec.Pool,
				"cannot be modified",
			),
		)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(ipamv1.GroupVersion.WithKind("IPClaim").GroupKind(), newIPClaim.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (webhook *IPClaim) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
