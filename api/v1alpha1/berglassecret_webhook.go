/*


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
	"context"
	"maps"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	myberglas "github.com/kitagry/berglas-secret-controller/internal/berglas"
)

// log is for logging in this package.
var berglassecretlog = logf.Log.WithName("berglassecret-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *BerglasSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-batch-kitagry-github-io-v1alpha1-berglassecret,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.kitagry.github.io,resources=berglassecrets,verbs=create;update,versions=v1alpha1,name=vberglassecret.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &BerglasSecret{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateCreate() (admission.Warnings, error) {
	ctx := context.Background()

	berglasClient, err := myberglas.New(ctx)
	if err != nil {
		berglassecretlog.Error(err, "failed to create berglas client")
		return nil, nil
	}

	return r.validate(ctx, berglasClient)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	oldBerglasSecret, ok := old.(*BerglasSecret)
	if !ok {
		return nil, nil
	}

	if maps.Equal(r.Spec.Data, oldBerglasSecret.Spec.Data) {
		return nil, nil
	}

	ctx := context.Background()

	berglasClient, err := myberglas.New(ctx)
	if err != nil {
		berglassecretlog.Error(err, "failed to create berglas client")
		return nil, nil
	}

	return r.validate(ctx, berglasClient)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

type berglasClient interface {
	Resolve(ctx context.Context, ref string) ([]byte, error)
}

func (r *BerglasSecret) validate(ctx context.Context, berglasClient berglasClient) (admission.Warnings, error) {
	var allErrs field.ErrorList
	for key, secret := range r.Spec.Data {
		ref, err := berglas.ParseReference(secret)
		if err != nil {
			continue
		}

		_, err = berglasClient.Resolve(ctx, ref.String())
		if err != nil {
			allErrs = append(allErrs, &field.Error{
				Type:     field.ErrorTypeNotFound,
				Field:    "spec.data." + key,
				BadValue: secret,
				Detail:   err.Error(),
			})
		}
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	groupVersionKind := r.GroupVersionKind()
	return nil, apierrors.NewInvalid(
		schema.GroupKind{Group: groupVersionKind.Group, Kind: groupVersionKind.Kind},
		r.Name,
		allErrs,
	)
}
