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

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var berglassecretlog = logf.Log.WithName("berglassecret-resource")

func (r *BerglasSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:verbs=create;update,path=/validate-batch-kitagry-github-io-v1alpha1-berglassecret,mutating=false,sideEffects=none,failurePolicy=fail,groups=batch.kitagry.github.io,resources=berglassecrets,versions=v1alpha1,name=kitagry.github.io,admissionReviewVersions={v1}

var _ webhook.Validator = &BerglasSecret{}

type berglasClient interface {
	Resolve(ctx context.Context, s string) ([]byte, error)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateCreate() error {
	berglassecretlog.Info("validate create", "name", r.Name)

	ctx := context.Background()
	berglasClient, err := newBerglasClient(ctx)
	if err != nil {
		return nil
	}

	for _, d := range r.Spec.Data {
		ref, err := berglas.ParseReference(d)
		if err != nil {
			continue
		}

		_, err = berglasClient.Resolve(ctx, ref.String())
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateUpdate(old runtime.Object) error {
	berglassecretlog.Info("validate update", "name", r.Name)

	oldSecret, ok := old.(*BerglasSecret)
	if !ok {
		return nil
	}

	ctx := context.Background()
	berglasClient, err := newBerglasClient(ctx)
	if err != nil {
		return nil
	}

	for key, value := range r.Spec.Data {
		oldValue, ok := oldSecret.Spec.Data[key]
		if ok && value == oldValue {
			continue
		}

		ref, err := berglas.ParseReference(value)
		if err != nil {
			continue
		}

		_, err = berglasClient.Resolve(ctx, ref.String())
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BerglasSecret) ValidateDelete() error {
	berglassecretlog.Info("validate delete", "name", r.Name)

	return nil
}
