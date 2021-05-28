package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	batchv1alpha1 "github.com/kitagry/berglas-secret-controller/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretAnnotationKey = "kitagry.github.io/berglasSecret"
)

func (r *BerglasSecretReconciler) reconcileSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret) error {
	var secrets v1.SecretList
	if err := r.List(ctx, &secrets, client.MatchingFields(map[string]string{ownerControllerField: req.Name})); err != nil {
		return err
	}

	if len(secrets.Items) == 0 {
		return r.createSecret(ctx, req, bs)
	}
	return r.updateSecret(ctx, req, bs, &secrets)
}

func (r *BerglasSecretReconciler) createSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret) error {
	data, err := r.resolveBerglasSchemas(ctx, bs.Spec.Data)
	if err != nil {
		return err
	}

	annotationData, err := json.Marshal(bs.Spec.Data)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Annotations: map[string]string{
				secretAnnotationKey: string(annotationData),
			},
		},
		StringData: data,
	}
	if err := ctrl.SetControllerReference(bs, secret, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (r *BerglasSecretReconciler) resolveBerglasSchemas(ctx context.Context, data map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(data))
	for key, value := range data {
		ref, err := berglas.ParseReference(value)
		if err != nil {
			result[key] = value
			continue
		}

		plaintext, err := r.Berglas.Resolve(ctx, ref.String())
		if err != nil {
			return nil, err
		}
		result[key] = string(plaintext)
	}
	return result, nil
}

func (r *BerglasSecretReconciler) updateSecret(ctx context.Context, req ctrl.Request, bs *batchv1alpha1.BerglasSecret, secrets *v1.SecretList) error {
	if len(secrets.Items) > 1 {
		return fmt.Errorf("owned secrets should be one item, but got %d", len(secrets.Items))
	}

	secret := secrets.Items[0]
	anntationDataStr := secret.Annotations[secretAnnotationKey]
	var annotationData map[string]string
	if err := json.Unmarshal([]byte(anntationDataStr), &annotationData); err != nil {
		return fmt.Errorf("failed to get annotation data: %w", err)
	}

	if !isChanged(annotationData, bs.Spec.Data) {
		return nil
	}

	// When we update both a berglasSecret and a pod which volume the berglasSecret,
	// the pod might volume secret which is not updated yet.
	// So, we delete secret firstly, and then create new secret.
	err := r.Delete(ctx, &secret)
	if err != nil {
		return fmt.Errorf("failed to update secret in the step of deleting old secret: %w", err)
	}
	return r.createSecret(ctx, req, bs)
}

func isChanged(v, u map[string]string) bool {
	if len(v) != len(u) {
		return true
	}

	for key, vValue := range v {
		uValue, ok := u[key]
		if !ok {
			return true
		}

		if vValue != uValue {
			return true
		}
	}
	return false
}
