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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
)

// BerglasSecretReconciler reconciles a BerglasSecret object
type BerglasSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.kitagry.github.io,resources=berglassecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.kitagry.github.io,resources=berglassecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets/status,verbs=get

func (r *BerglasSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("berglassecret", req.NamespacedName)

	// your logic here
	var berglasSecret batchv1alpha1.BerglasSecret
	if err := r.Get(ctx, req.NamespacedName, &berglasSecret); err != nil {
		logger.Error(err, "failed to fetch berglas_secret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func (r *BerglasSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.BerglasSecret{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
